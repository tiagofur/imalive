// Package auth implements phone OTP authentication (R-01): issue a code,
// verify it (creating the user on first sign-in), and maintain refresh-token
// sessions with rotation and family revocation. Access tokens are short-lived
// JWTs with no PII beyond the user id; refresh tokens are opaque random
// strings tracked only by their SHA-256 in Redis.
package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
)

// Service-level timings (docs/stack/backend-go.md + plan of issue #39).
const (
	// AccessTTL is the life of a JWT access token.
	AccessTTL = 15 * time.Minute
	// RefreshTTL is the sliding life of a refresh session family.
	RefreshTTL  = 30 * 24 * time.Hour
	issuer      = "tranqui"
	refreshSize = 32 // bytes of entropy per refresh secret
	// displayNamePlaceholder fills the NOT NULL column at first sign-in; the
	// profile flow (R-02, issue #40) replaces it with the real name.
	displayNamePlaceholder = "Usuario"
)

// Sentinel errors mapped by the handlers to contract codes (400 Problem
// examples): the caller never learns more than the contract already publishes.
var (
	// ErrOtpExpired maps to OTP_EXPIRED (no live code: never issued, TTL
	// elapsed, or consumed by a previous attempt — indistinguishable by
	// design, see otp.Store).
	ErrOtpExpired = errors.New("auth: otp expired")
	// ErrOtpInvalid maps to OTP_INVALID (live code, wrong candidate).
	ErrOtpInvalid = errors.New("auth: otp invalid")
	// ErrSessionRevoked maps to 401 on refresh after reuse detection: the
	// whole family is revoked.
	ErrSessionRevoked = errors.New("auth: session family revoked")
)

// UsersStore is the persistence the service needs (implemented by the sqlc
// store; faked in unit tests). Users are created on first sign-in — verify
// IS the signup.
type UsersStore interface {
	CreateUser(ctx context.Context, arg store.CreateUserParams) (store.User, error)
	GetUserByPhone(ctx context.Context, phone string) (store.User, error)
}

// Service issues and verifies OTPs and mints token pairs.
type Service struct {
	otp       *otp.Store
	sender    otp.Sender
	rdb       redis.Cmdable
	users     UsersStore
	jwtSecret []byte
	now       func() time.Time
}

// NewService wires the auth service. The otp store, sender and Redis handle
// are shared with the rest of the platform (single factory, issue #39).
func NewService(otpStore *otp.Store, sender otp.Sender, rdb redis.Cmdable, users UsersStore, jwtSecret string) *Service {
	return &Service{
		otp:       otpStore,
		sender:    sender,
		rdb:       rdb,
		users:     users,
		jwtSecret: []byte(jwtSecret),
		now:       time.Now,
	}
}

// Request delivers a fresh OTP to the phone. Idempotent from the client's
// point of view (always success); delivery failures DO fail (parity with a
// real provider).
func (s *Service) Request(ctx context.Context, phone string) error {
	code, err := otp.Generate()
	if err != nil {
		return fmt.Errorf("auth: request: %w", err)
	}
	if err := s.otp.Issue(ctx, phone, code, otp.DefaultTTL); err != nil {
		return fmt.Errorf("auth: request: %w", err)
	}
	if err := s.sender.Send(ctx, phone, code); err != nil {
		return fmt.Errorf("auth: request: %w", err)
	}
	return nil
}

// Tokens is the contract's Tokens schema payload.
type Tokens struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
}

// Verify consumes the OTP and returns a fresh token pair, creating the user
// on first sign-in (verify IS the signup).
func (s *Service) Verify(ctx context.Context, phone, code string) (Tokens, error) {
	outcome, err := s.otp.Consume(ctx, phone, code)
	if err != nil {
		return Tokens{}, fmt.Errorf("auth: verify: %w", err)
	}
	switch outcome {
	case otp.Expired:
		return Tokens{}, ErrOtpExpired
	case otp.Invalid:
		return Tokens{}, ErrOtpInvalid
	}

	user, err := s.users.GetUserByPhone(ctx, phone)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		created, createErr := s.users.CreateUser(ctx, store.CreateUserParams{
			ID:          newUUID(),
			Phone:       phone,
			DisplayName: displayNamePlaceholder,
			Timezone:    "UTC", // the profile PATCH sets the real one (R-02)
		})
		if createErr != nil {
			return Tokens{}, fmt.Errorf("auth: verify: %w", createErr)
		}
		user = created
	case err != nil:
		return Tokens{}, fmt.Errorf("auth: verify: %w", err)
	}

	pair, err := s.startSession(ctx, user.ID.String())
	if err != nil {
		return Tokens{}, err
	}
	return pair, nil
}

// Refresh rotates the session: the presented refresh token is consumed and a
// new pair is issued within the same family. Presenting a token whose secret
// no longer matches the family's live hash is reuse/theft: the whole family
// is revoked and access is denied (401).
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	family, secret, ok := splitRefresh(refreshToken)
	if !ok {
		return Tokens{}, ErrSessionRevoked
	}
	stored, err := s.rdb.Get(ctx, refreshLiveKey(family)).Result()
	if errors.Is(err, redis.Nil) || (err == nil && stored != hashRefresh(secret)) {
		s.revokeFamily(ctx, family)
		return Tokens{}, ErrSessionRevoked
	}
	if err != nil {
		return Tokens{}, fmt.Errorf("auth: refresh: %w", err)
	}

	userID, err := s.rdb.Get(ctx, refreshUserKey(family)).Result()
	if err != nil {
		s.revokeFamily(ctx, family)
		return Tokens{}, ErrSessionRevoked
	}

	next := newToken()
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, refreshLiveKey(family), hashRefresh(next), RefreshTTL)
	pipe.Expire(ctx, refreshUserKey(family), RefreshTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return Tokens{}, fmt.Errorf("auth: refresh: %w", err)
	}

	access, err := s.mintAccess(userID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: family + "." + next, ExpiresInSeconds: int(AccessTTL.Seconds())}, nil
}

// UserID validates an access JWT and returns the user id (sub). The
// middleware calls this: signature check is local, no roundtrip.
func (s *Service) UserID(accessToken string) (string, error) {
	token, err := jwt.Parse(accessToken, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	}, jwt.WithIssuer(issuer), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		return "", fmt.Errorf("auth: invalid access token")
	}
	sub, err := token.Claims.GetSubject()
	if err != nil || sub == "" {
		return "", fmt.Errorf("auth: invalid access token")
	}
	return sub, nil
}

// startSession creates the first refresh token of a family.
func (s *Service) startSession(ctx context.Context, userID string) (Tokens, error) {
	family := newToken()
	secret := newToken()
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, refreshLiveKey(family), hashRefresh(secret), RefreshTTL)
	pipe.Set(ctx, refreshUserKey(family), userID, RefreshTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return Tokens{}, fmt.Errorf("auth: session: %w", err)
	}

	access, err := s.mintAccess(userID)
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: family + "." + secret, ExpiresInSeconds: int(AccessTTL.Seconds())}, nil
}

func (s *Service) mintAccess(userID string) (string, error) {
	now := s.now()
	claims := jwt.MapClaims{
		"iss": issuer,
		"sub": userID,
		"iat": now.Unix(),
		"exp": now.Add(AccessTTL).Unix(),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("auth: sign: %w", err)
	}
	return signed, nil
}

func (s *Service) revokeFamily(ctx context.Context, family string) {
	_ = s.rdb.Del(ctx, refreshLiveKey(family), refreshUserKey(family)).Err()
}

// --- token plumbing (package-private, unit-tested) ---

func refreshLiveKey(family string) string { return "sess:" + family + ":live" }

func refreshUserKey(family string) string { return "sess:" + family + ":user" }

func hashRefresh(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// newToken returns a fresh 256-bit random secret, hex-encoded.
func newToken() string {
	buf := make([]byte, refreshSize)
	if _, err := rand.Read(buf); err != nil {
		panic(fmt.Sprintf("auth: entropy: %v", err)) // crypto/rand exhaustion is not recoverable
	}
	return hex.EncodeToString(buf)
}

// splitRefresh parses "{family}.{secret}".
func splitRefresh(token string) (family, secret string, ok bool) {
	idx := -1
	for i := 0; i < len(token); i++ {
		if token[i] == '.' {
			idx = i
			break
		}
	}
	if idx <= 0 || idx == len(token)-1 {
		return "", "", false
	}
	return token[:idx], token[idx+1:], true
}

func newUUID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("auth: uuid v7: %v", err)) // crypto/rand exhaustion is not recoverable
	}
	return id
}
