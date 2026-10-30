package auth_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth"
	"github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
)

// recordingSender is the observable boundary of the service: tests read the
// code it was asked to deliver, exactly like the sandbox outbox does locally.
type recordingSender struct {
	mu    sync.Mutex
	codes map[string]string
}

func newRecordingSender() *recordingSender {
	return &recordingSender{codes: map[string]string{}}
}

func (s *recordingSender) Send(_ context.Context, phone, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[phone] = code
	return nil
}

func (s *recordingSender) last(phone string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.codes[phone]
}

// fakeUsers stands in for the sqlc store in unit tests (the store runs
// against real Postgres in the handlers' integration tests, PR D).
type fakeUsers struct {
	byPhone map[string]store.User
	created int
}

func (f *fakeUsers) CreateUser(_ context.Context, arg store.CreateUserParams) (store.User, error) {
	f.created++
	u := store.User{ID: arg.ID, Phone: arg.Phone, DisplayName: arg.DisplayName, Timezone: arg.Timezone}
	f.byPhone[arg.Phone] = u
	return u, nil
}

func (f *fakeUsers) GetUserByPhone(_ context.Context, phone string) (store.User, error) {
	u, ok := f.byPhone[phone]
	if !ok {
		return store.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func newService(t *testing.T, users *fakeUsers) (*auth.Service, *recordingSender) {
	t.Helper()
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	sender := newRecordingSender()
	svc := auth.NewService(otp.NewStore(rdb, "test-secret"), sender, rdb, users, "test-jwt-secret")
	return svc, sender
}

func TestVerifyCreatesUserOnFirstSignInAndIssuesTokens(t *testing.T) {
	users := &fakeUsers{byPhone: map[string]store.User{}}
	svc, sender := newService(t, users)
	ctx := context.Background()
	phone := "+5215511111111"

	if err := svc.Request(ctx, phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pair, err := svc.Verify(ctx, phone, sender.last(phone))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if pair.AccessToken == "" || pair.RefreshToken == "" {
		t.Fatalf("tokens empty: %+v", pair)
	}
	if users.created != 1 {
		t.Fatalf("users created = %d, want 1 (verify IS signup)", users.created)
	}
	id, err := svc.UserID(pair.AccessToken)
	if err != nil || id == "" {
		t.Fatalf("UserID = %q, %v; want a valid id", id, err)
	}
	if strings.Contains(pair.AccessToken, phone) {
		t.Fatal("the access token leaks the phone (PII)")
	}

	// Second sign-in with the same phone reuses the user (no new row).
	if err := svc.Request(ctx, phone); err != nil {
		t.Fatalf("second Request: %v", err)
	}
	if _, err := svc.Verify(ctx, phone, sender.last(phone)); err != nil {
		t.Fatalf("second Verify: %v", err)
	}
	if users.created != 1 {
		t.Fatalf("users created after re-signin = %d, want still 1", users.created)
	}
}

func TestVerifyMapsOtpOutcomesToSentinels(t *testing.T) {
	users := &fakeUsers{byPhone: map[string]store.User{}}
	svc, sender := newService(t, users)
	ctx := context.Background()
	phone := "+5215522222222"

	// Never issued: OTP_EXPIRED (indistinguishable from elapsed by design).
	if _, err := svc.Verify(ctx, phone, "111111"); !errors.Is(err, auth.ErrOtpExpired) {
		t.Fatalf("Verify never-issued = %v, want ErrOtpExpired", err)
	}

	if err := svc.Request(ctx, phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	// Wrong code: OTP_INVALID, and the attempt is consumed.
	if _, err := svc.Verify(ctx, phone, "000000"); !errors.Is(err, auth.ErrOtpInvalid) {
		t.Fatalf("Verify wrong code = %v, want ErrOtpInvalid", err)
	}
	if _, err := svc.Verify(ctx, phone, sender.last(phone)); !errors.Is(err, auth.ErrOtpExpired) {
		t.Fatalf("Verify after failed attempt = %v, want ErrOtpExpired", err)
	}
}

func TestRefreshRotationAndReuseRevocation(t *testing.T) {
	users := &fakeUsers{byPhone: map[string]store.User{}}
	svc, sender := newService(t, users)
	ctx := context.Background()
	phone := "+5215533333333"

	if err := svc.Request(ctx, phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pair, err := svc.Verify(ctx, phone, sender.last(phone))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}

	// Rotation: the old refresh token stops working, the new one works.
	next, err := svc.Refresh(ctx, pair.RefreshToken)
	if err != nil {
		t.Fatalf("Refresh: %v", err)
	}
	if next.RefreshToken == pair.RefreshToken {
		t.Fatal("rotation returned the same refresh token")
	}
	if _, err := svc.Refresh(ctx, pair.RefreshToken); !errors.Is(err, auth.ErrSessionRevoked) {
		t.Fatalf("reused refresh = %v, want ErrSessionRevoked", err)
	}
	// Reuse detected => the whole family dies, even the rotated token.
	if _, err := svc.Refresh(ctx, next.RefreshToken); !errors.Is(err, auth.ErrSessionRevoked) {
		t.Fatalf("refresh after family revocation = %v, want ErrSessionRevoked", err)
	}
}

func TestAccessTokensValidateLocally(t *testing.T) {
	users := &fakeUsers{byPhone: map[string]store.User{}}
	svc, sender := newService(t, users)
	ctx := context.Background()
	phone := "+5215544444444"

	if err := svc.Request(ctx, phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pair, err := svc.Verify(ctx, phone, sender.last(phone))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if id, err := svc.UserID(pair.AccessToken); err != nil || id == "" {
		t.Fatalf("fresh token invalid: %q, %v", id, err)
	}
	// Tampering with the signature breaks validation (local check).
	if _, err := svc.UserID(strings.Replace(pair.AccessToken, ".", "x.", 1)); err == nil {
		t.Fatal("tampered token accepted")
	}
}

func TestMiddlewareGuardsRoutes(t *testing.T) {
	users := &fakeUsers{byPhone: map[string]store.User{}}
	svc, sender := newService(t, users)
	ctx := context.Background()
	phone := "+5215555555555"

	if err := svc.Request(ctx, phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pair, err := svc.Verify(ctx, phone, sender.last(phone))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	mw := auth.NewMiddleware(svc)

	var gotUserID string
	handler := mw.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID = auth.UserIDFromContext(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	// No token -> 401 Problem.
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/me", nil))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("no-token status = %d, want 401", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.Contains(ct, "application/problem+json") {
		t.Fatalf("content type = %q, want problem+json", ct)
	}

	// Valid token -> passes with the user id in context.
	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/v1/me", nil)
	req.Header.Set("Authorization", "Bearer "+pair.AccessToken)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || gotUserID == "" {
		t.Fatalf("valid token = status %d, user %q; want 200 with id", rec.Code, gotUserID)
	}
}

func TestTTLsMatchThePlan(t *testing.T) {
	if auth.AccessTTL != 15*time.Minute {
		t.Fatalf("AccessTTL = %v, want 15m", auth.AccessTTL)
	}
	if auth.RefreshTTL != 30*24*time.Hour {
		t.Fatalf("RefreshTTL = %v, want 30d", auth.RefreshTTL)
	}
}
