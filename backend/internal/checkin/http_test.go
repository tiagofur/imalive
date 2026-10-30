package checkin_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/checkin"
	"github.com/tiagofur/imalive/backend/internal/checkin/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
)

type codeRecorder struct{ codes map[string]string }

func newCodeRecorder() *codeRecorder { return &codeRecorder{codes: map[string]string{}} }

func (c *codeRecorder) Send(_ context.Context, phone, code string) error {
	c.codes[phone] = code
	return nil
}

func (c *codeRecorder) last(phone string) string { return c.codes[phone] }

// api mounts the real stack (migrated Postgres + miniredis + auth + checkin)
// exactly like cmd/api does.
type api struct {
	server *httptest.Server
	sender *codeRecorder
	pool   *pgxpool.Pool
	svc    *auth.Service
}

func newAPI(t *testing.T) *api {
	t.Helper()
	p := dbtest.New(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	sender := newCodeRecorder()

	svc := auth.NewService(otp.NewStore(rdb, "test-secret"), sender, rdb,
		authstore.New(p.Pool), "test-secret")
	mux := http.NewServeMux()
	auth.NewHandlers(svc, auth.NewRateLimiter(rdb)).Register(mux)
	checkin.NewHandlers(store.New(p.Pool)).Register(mux, auth.NewMiddleware(svc))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &api{server: srv, sender: sender, pool: p.Pool, svc: svc}
}

func (a *api) signUp(t *testing.T, phone string) string {
	t.Helper()
	if err := a.svc.Request(context.Background(), phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pair, err := a.svc.Verify(context.Background(), phone, a.sender.last(phone))
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return pair.AccessToken
}

func callAPI(t *testing.T, method, url, token string, payload any, headers map[string]string) *http.Response {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, url, bodyFor(t, payload))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func bodyFor(t *testing.T, payload any) *bytes.Reader {
	t.Helper()
	if payload == nil {
		return bytes.NewReader(nil)
	}
	b, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return bytes.NewReader(b)
}

func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return m
}

func TestCheckinFlow(t *testing.T) {
	api := newAPI(t)
	phone := "+5215577660001"
	token := api.signUp(t, phone)

	t.Run("first check-in of the day returns 201 with replayed=false", func(t *testing.T) {
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/checkins", token, nil, map[string]string{"Idempotency-Key": "key-1"})
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, body = %v; want 201", resp.StatusCode, body)
		}
		if body["replayed"] != false {
			t.Fatalf("replayed = %v, want false", body["replayed"])
		}
		if body["date"] == nil || body["checkedInAt"] == nil {
			t.Fatalf("body = %v, want date and checkedInAt", body)
		}
	})

	t.Run("replay with the same key returns 200 with replayed=true", func(t *testing.T) {
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/checkins", token, nil, map[string]string{"Idempotency-Key": "key-1"})
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusOK || body["replayed"] != true {
			t.Fatalf("replay = %d %v; want 200 replayed=true", resp.StatusCode, body["replayed"])
		}
	})

	t.Run("a second same-day check-in without a key also replays", func(t *testing.T) {
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/checkins", token, nil, nil)
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusOK || body["replayed"] != true {
			t.Fatalf("same-day = %d %v; want 200 replayed=true", resp.StatusCode, body["replayed"])
		}
	})

	t.Run("today reports confirmed with the checkedInAt", func(t *testing.T) {
		resp := callAPI(t, http.MethodGet, api.server.URL+"/v1/checkins/today", token, nil, nil)
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusOK || body["status"] != "confirmed" {
			t.Fatalf("today = %d %v; want 200 confirmed", resp.StatusCode, body)
		}
		if body["checkedInAt"] == nil {
			t.Fatalf("today = %v, want checkedInAt", body)
		}
		if _, ok := body["nextCheckinAt"]; !ok {
			t.Fatalf("today = %v, want nextCheckinAt", body)
		}
	})

	t.Run("keys are per-user (no cross-user replay)", func(t *testing.T) {
		otherToken := api.signUp(t, "+5215577660002")
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/checkins", otherToken, nil, map[string]string{"Idempotency-Key": "key-1"})
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusCreated || body["replayed"] != false {
			t.Fatalf("other user with the same key = %d %v; want 201 replayed=false", resp.StatusCode, body["replayed"])
		}
	})

	t.Run("keys longer than 64 chars are rejected", func(t *testing.T) {
		long := make([]byte, 65)
		for i := range long {
			long[i] = 'a'
		}
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/checkins", token, nil, map[string]string{"Idempotency-Key": string(long)})
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "VALIDATION_FAILED" {
			t.Fatalf("long key = %d %v; want 400 VALIDATION_FAILED", resp.StatusCode, body["code"])
		}
	})

	t.Run("no token yields 401", func(t *testing.T) {
		resp := callAPI(t, http.MethodPost, api.server.URL+"/v1/checkins", "", nil, nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	// The cycle date must respect the user's timezone: move the user to
	// Auckland (UTC+13) and verify today's date follows local time, not UTC.
	t.Run("today's date comes from the user timezone, not UTC", func(t *testing.T) {
		if _, err := api.pool.Exec(context.Background(),
			"UPDATE users SET timezone = 'Pacific/Auckland' WHERE phone = $1", phone); err != nil {
			t.Fatalf("set timezone: %v", err)
		}
		local := time.Now().In(time.FixedZone("NZ", 13*3600))
		resp := callAPI(t, http.MethodGet, api.server.URL+"/v1/checkins/today", token, nil, nil)
		body := decodeBody(t, resp)
		if body["date"] != local.Format("2006-01-02") {
			t.Fatalf("date = %v, want the Auckland-local date %v", body["date"], local.Format("2006-01-02"))
		}
	})
}
