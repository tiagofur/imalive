package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
)

func miniredisForTest(t *testing.T) *miniredis.Miniredis {
	t.Helper()
	return miniredis.RunT(t)
}

func redisClientForTest(t *testing.T, mr *miniredis.Miniredis) *redis.Client {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })
	return rdb
}

// newTestAPI mounts the real handlers over a real migrated Postgres
// (testcontainers) and a miniredis, mirroring the cmd/api wiring.
func newTestAPI(t *testing.T) (*httptest.Server, *recordingSender) {
	t.Helper()
	p := dbtest.New(t)
	mr := miniredisForTest(t)
	rdb := redisClientForTest(t, mr)
	sender := newRecordingSender()

	svc := auth.NewService(otp.NewStore(rdb, "test-secret"), sender, rdb,
		authstore.New(p.Pool), "test-secret")
	handlers := auth.NewHandlers(svc, auth.NewRateLimiter(rdb))
	mux := http.NewServeMux()
	handlers.Register(mux)

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, sender
}

func postJSON(t *testing.T, url string, payload any) *http.Response {
	t.Helper()
	body := mustJSON(t, payload)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func decode(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	var m map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return m
}

func TestAuthFlowAgainstTheContract(t *testing.T) {
	srv, sender := newTestAPI(t)
	phone := "+5215577889999"

	t.Run("request returns 204 and never leaks enumeration", func(t *testing.T) {
		resp := postJSON(t, srv.URL+"/v1/auth/otp/request", map[string]string{"phone": phone})
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
		if sender.last(phone) == "" {
			t.Fatal("no code delivered to the sender")
		}
		// Unknown phone: same 204 (anti-enumeration).
		resp2 := postJSON(t, srv.URL+"/v1/auth/otp/request", map[string]string{"phone": "+5299900000001"})
		defer func() { _ = resp2.Body.Close() }()
		if resp2.StatusCode != http.StatusNoContent {
			t.Fatalf("unknown phone status = %d, want 204", resp2.StatusCode)
		}
	})

	t.Run("malformed phone yields 400 PHONE_INVALID with the problem shape", func(t *testing.T) {
		resp := postJSON(t, srv.URL+"/v1/auth/otp/request", map[string]string{"phone": "5512345678"})
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("status = %d, want 400", resp.StatusCode)
		}
		body := decode(t, resp)
		if body["code"] != "PHONE_INVALID" {
			t.Fatalf("code = %v, want PHONE_INVALID", body["code"])
		}
		if !strings.HasPrefix(fmt.Sprint(body["type"]), "https://api.tranqui.app/problems/") {
			t.Fatalf("type = %v, want the problems URI base", body["type"])
		}
		if body["instance"] != "/v1/auth/otp/request" {
			t.Fatalf("instance = %v, want the request path", body["instance"])
		}
	})

	t.Run("verify returns the contract Tokens with 900 seconds", func(t *testing.T) {
		resp := postJSON(t, srv.URL+"/v1/auth/otp/verify", map[string]string{
			"phone": phone, "code": sender.last(phone),
		})
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			body := decode(t, resp)
			t.Fatalf("status = %d, body = %v; want 200", resp.StatusCode, body)
		}
		tokens := decode(t, resp)
		if tokens["accessToken"] == "" || tokens["refreshToken"] == "" {
			t.Fatalf("tokens = %v, want both present", tokens)
		}
		if tokens["expiresInSeconds"] != float64(900) {
			t.Fatalf("expiresInSeconds = %v, want 900", tokens["expiresInSeconds"])
		}
	})

	t.Run("wrong code yields 400 OTP_INVALID per the contract", func(t *testing.T) {
		other := "+5215577889990"
		if _, err := postBody(t, srv.URL+"/v1/auth/otp/request", map[string]string{"phone": other}); err != nil {
			t.Fatalf("request: %v", err)
		}
		resp := postJSON(t, srv.URL+"/v1/auth/otp/verify", map[string]string{"phone": other, "code": "000000"})
		defer func() { _ = resp.Body.Close() }()
		body := decode(t, resp)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "OTP_INVALID" {
			t.Fatalf("status = %d, code = %v; want 400 OTP_INVALID", resp.StatusCode, body["code"])
		}
	})

	t.Run("refresh rotates and reuse yields 401 UNAUTHORIZED", func(t *testing.T) {
		fresh := "+5215577889991"
		if _, err := postBody(t, srv.URL+"/v1/auth/otp/request", map[string]string{"phone": fresh}); err != nil {
			t.Fatalf("request: %v", err)
		}
		first := postJSON(t, srv.URL+"/v1/auth/otp/verify", map[string]string{"phone": fresh, "code": sender.last(fresh)})
		tokens := decode(t, first)
		defer func() { _ = first.Body.Close() }()

		rotated := postJSON(t, srv.URL+"/v1/auth/refresh", map[string]string{"refreshToken": fmt.Sprint(tokens["refreshToken"])})
		defer func() { _ = rotated.Body.Close() }()
		if rotated.StatusCode != http.StatusOK {
			t.Fatalf("refresh status = %d, want 200", rotated.StatusCode)
		}
		newTokens := decode(t, rotated)
		if newTokens["refreshToken"] == tokens["refreshToken"] {
			t.Fatal("rotation returned the same refresh token")
		}

		reuse := postJSON(t, srv.URL+"/v1/auth/refresh", map[string]string{"refreshToken": fmt.Sprint(tokens["refreshToken"])})
		defer func() { _ = reuse.Body.Close() }()
		reuseBody := decode(t, reuse)
		if reuse.StatusCode != http.StatusUnauthorized || reuseBody["code"] != "UNAUTHORIZED" {
			t.Fatalf("reuse = %d %v; want 401 UNAUTHORIZED", reuse.StatusCode, reuseBody["code"])
		}
	})
}

func postBody(t *testing.T, url string, payload any) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(mustJSON(t, payload)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	return http.DefaultClient.Do(req)
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return b
}

func TestRateLimiterYields429WithRetryAfter(t *testing.T) {
	srv, _ := newTestAPI(t)
	phone := "+5215599888777"

	var saw429 *http.Response
	for range 10 {
		resp := postJSON(t, srv.URL+"/v1/auth/otp/request", map[string]string{"phone": phone})
		if resp.StatusCode == http.StatusTooManyRequests {
			saw429 = resp
			break
		}
		_ = resp.Body.Close()
	}
	if saw429 == nil {
		t.Fatal("never hit the 429 after repeated requests")
	}
	defer func() { _ = saw429.Body.Close() }()
	if ra := saw429.Header.Get("Retry-After"); ra == "" {
		t.Fatal("429 without Retry-After header")
	}
	body := decode(t, saw429)
	if body["code"] != "RATE_LIMITED" {
		t.Fatalf("code = %v, want RATE_LIMITED", body["code"])
	}
}
