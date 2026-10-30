package otp

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// DefaultTTL is how long an issued code stays verifiable.
const DefaultTTL = 10 * time.Minute

// Store keeps one hashed code per phone in Redis. Semantics (documented in
// the contract's 400 examples): the first verification attempt consumes the
// code atomically (GETDEL) — a wrong code yields OTP_INVALID, and any later
// attempt finds no key and yields OTP_EXPIRED. A re-issued code overwrites
// the previous one and resets the TTL.
type Store struct {
	rdb    redis.Cmdable
	secret string
}

// NewStore builds the Redis-backed store. The secret keys the HMAC: the
// same one from config.OtpSecret, never a per-environment different value.
func NewStore(rdb redis.Cmdable, secret string) *Store {
	return &Store{rdb: rdb, secret: secret}
}

func key(phone string) string { return "otp:" + phone }

// Issue hashes the code and stores it with the TTL, replacing any previous
// code for the phone.
func (s *Store) Issue(ctx context.Context, phone, code string, ttl time.Duration) error {
	err := s.rdb.Set(ctx, key(phone), Hash(s.secret, phone, code), ttl).Err()
	if err != nil {
		return fmt.Errorf("otp: issue: %w", err)
	}
	return nil
}

// Outcome distinguishes why a verification failed.
type Outcome int

const (
	// Verified means the code matched and the OTP is consumed.
	Verified Outcome = iota
	// Expired means no live code exists for the phone (never issued, TTL
	// elapsed, or already consumed by a previous attempt).
	Expired
	// Invalid means a live code exists but the candidate does not match.
	// The code was consumed by this attempt regardless (single-attempt rule).
	Invalid
)

// Consume atomically takes the live code for the phone and verifies it.
func (s *Store) Consume(ctx context.Context, phone, code string) (Outcome, error) {
	stored, err := s.rdb.GetDel(ctx, key(phone)).Result()
	if err == redis.Nil {
		return Expired, nil
	}
	if err != nil {
		return Expired, fmt.Errorf("otp: consume: %w", err)
	}
	if !Verify(s.secret, phone, code, stored) {
		return Invalid, nil
	}
	return Verified, nil
}
