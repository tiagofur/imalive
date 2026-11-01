package profile_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
	"github.com/tiagofur/imalive/backend/internal/profile"
	profilestore "github.com/tiagofur/imalive/backend/internal/profile/store"
)

// api mounts the real stack (migrated Postgres + miniredis + auth + profile
// handlers) exactly like cmd/api does, with signup helpers the tests share
// on the SAME server (one database, real cross-user semantics).
type api struct {
	server *httptest.Server
	sender *codeRecorder
	svc    *auth.Service
	pool   *pgxpool.Pool
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
	authHandlers := auth.NewHandlers(svc, auth.NewRateLimiter(rdb))
	engine := escalation.New(p.Pool, escalation.SystemClock{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	profileHandlers := profile.NewHandlers(p.Pool, authstore.New(p.Pool), profilestore.New(p.Pool), engine)
	authMW := auth.NewMiddleware(svc)

	mux := http.NewServeMux()
	authHandlers.Register(mux)
	profileHandlers.Register(mux, authMW)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &api{server: srv, sender: sender, svc: svc, pool: p.Pool}
}

// signUp runs the full OTP flow for the phone and returns a bearer token.
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

type codeRecorder struct{ codes map[string]string }

func (c *codeRecorder) Send(_ context.Context, phone, code string) error {
	c.codes[phone] = code
	return nil
}

func (c *codeRecorder) last(phone string) string { return c.codes[phone] }

func newCodeRecorder() *codeRecorder { return &codeRecorder{codes: map[string]string{}} }

func callAPI(t *testing.T, method, url, token string, payload any) *http.Response {
	t.Helper()
	var body bytes.Buffer
	if payload != nil {
		if err := json.NewEncoder(&body).Encode(payload); err != nil {
			t.Fatalf("encode: %v", err)
		}
	}
	req, err := http.NewRequestWithContext(context.Background(), method, url, &body)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
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

func TestProfileFlow(t *testing.T) {
	api := newAPI(t)
	phone := "+5215540440001"
	srv, token := api.server, api.signUp(t, phone)

	t.Run("GET /me returns the signup placeholder profile", func(t *testing.T) {
		resp := callAPI(t, http.MethodGet, srv.URL+"/v1/me", token, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		if body["name"] != "Usuario" || body["phone"] != phone {
			t.Fatalf("profile = %v, want placeholder name and the phone", body)
		}
		if body["checkinTime"] != "21:00" || body["escalationWindowMinutes"] != float64(60) {
			t.Fatalf("profile defaults = %v, want 21:00 and 60", body)
		}
		if body["streakDays"] != float64(0) {
			t.Fatalf("streak = %v, want 0 without check-ins", body["streakDays"])
		}
	})

	t.Run("the streak counts the seeded check-in days (R-12)", func(t *testing.T) {
		streakPhone := "+5215540440011"
		streakToken := api.signUp(t, streakPhone)
		var uid uuid.UUID
		if err := api.pool.QueryRow(context.Background(),
			`SELECT id FROM users WHERE phone = $1`, streakPhone).Scan(&uid); err != nil {
			t.Fatalf("find user: %v", err)
		}
		nowUTC := time.Now().UTC()
		for i := 1; i <= 3; i++ { // yesterday and two days before: streak alive
			day := nowUTC.AddDate(0, 0, -i)
			if _, err := api.pool.Exec(context.Background(),
				`INSERT INTO checkins (id, user_id, cycle_date, checked_in_at)
				 VALUES ($1, $2, $3, $4)`,
				uuid.Must(uuid.NewV7()), uid,
				pgtype.Date{Time: time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC), Valid: true},
				pgtype.Timestamptz{Time: day, Valid: true}); err != nil {
				t.Fatalf("seed checkin %d: %v", i, err)
			}
		}

		resp := callAPI(t, http.MethodGet, srv.URL+"/v1/me", streakToken, nil)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		if body["streakDays"] != float64(3) {
			t.Fatalf("streak = %v, want 3 (today is still pending)", body["streakDays"])
		}
	})

	t.Run("PATCH updates only the provided fields", func(t *testing.T) {
		resp := callAPI(t, http.MethodPatch, srv.URL+"/v1/me", token,
			map[string]any{"name": "Ana", "timezone": "America/Mexico_City", "checkinTime": "22:30"})
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		body := decodeBody(t, resp)
		if body["name"] != "Ana" || body["timezone"] != "America/Mexico_City" || body["checkinTime"] != "22:30" {
			t.Fatalf("patched = %v", body)
		}
		// The contract's UserProfile requires streakDays on EVERY 200
		// (issue #86: the PATCH response omitted it and broke v0.5.0 apps).
		if _, ok := body["streakDays"]; !ok {
			t.Fatalf("patched = %v, want streakDays present (required by the schema)", body)
		}
		if body["escalationWindowMinutes"] != float64(60) {
			t.Fatalf("untouched field = %v, want 60 (omitted fields keep values)", body)
		}
	})

	t.Run("empty patch is rejected by minProperties", func(t *testing.T) {
		resp := callAPI(t, http.MethodPatch, srv.URL+"/v1/me", token, map[string]any{})
		body := decodeBody(t, resp)
		if resp.StatusCode != http.StatusBadRequest || body["code"] != "VALIDATION_FAILED" {
			t.Fatalf("empty patch = %d %v; want 400 VALIDATION_FAILED", resp.StatusCode, body["code"])
		}
	})

	t.Run("invalid values yield 400 with clear details", func(t *testing.T) {
		cases := []struct {
			name    string
			payload map[string]any
		}{
			{"bad checkinTime", map[string]any{"checkinTime": "25:99"}},
			{"bad timezone", map[string]any{"timezone": "Mars/Olympus"}},
			{"escalation out of range", map[string]any{"escalationWindowMinutes": 2000}},
			{"name too long", map[string]any{"name": strings.Repeat("x", 81)}},
		}
		for _, tc := range cases {
			resp := callAPI(t, http.MethodPatch, srv.URL+"/v1/me", token, tc.payload)
			body := decodeBody(t, resp)
			if resp.StatusCode != http.StatusBadRequest || body["code"] != "VALIDATION_FAILED" {
				t.Fatalf("%s: = %d %v; want 400 VALIDATION_FAILED", tc.name, resp.StatusCode, body["code"])
			}
		}
	})

	t.Run("no token yields 401", func(t *testing.T) {
		resp := callAPI(t, http.MethodGet, srv.URL+"/v1/me", "", nil)
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})

	t.Run("another user only sees their own profile", func(t *testing.T) {
		other := "+5215540440002"
		otherToken := api.signUp(t, other)
		resp := callAPI(t, http.MethodGet, srv.URL+"/v1/me", otherToken, nil)
		body := decodeBody(t, resp)
		if body["phone"] != other {
			t.Fatalf("phone = %v, want the other user's own phone", body["phone"])
		}
	})
}

// TestPauseFreezesLadder exercises the R-11 invariant end to end: activating
// the pause cancels the user's pending ladder jobs in the SAME transaction
// (a pause without the freeze is impossible).
func TestPauseFreezesLadder(t *testing.T) {
	api := newAPI(t)
	phone := "+5215540440010"
	token := api.signUp(t, phone)

	var uid uuid.UUID
	if err := api.pool.QueryRow(context.Background(),
		`SELECT id FROM users WHERE phone = $1`, phone).Scan(&uid); err != nil {
		t.Fatalf("find user: %v", err)
	}
	jobID := uuid.Must(uuid.NewV7())
	if _, err := api.pool.Exec(context.Background(),
		`INSERT INTO escalation_jobs (id, user_id, cycle_date, kind, run_at)
		 VALUES ($1, $2, CURRENT_DATE, 'notify_circle', now() + interval '2 hours')`, jobID, uid); err != nil {
		t.Fatalf("seed job: %v", err)
	}

	put := callAPI(t, http.MethodPut, api.server.URL+"/v1/me/pause", token,
		map[string]any{"until": time.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)})
	if put.StatusCode != http.StatusNoContent {
		body := decodeBody(t, put)
		t.Fatalf("pause = %d %v; want 204", put.StatusCode, body)
	}
	// The streak's freeze window is [pause_since, pause_until] (R-12).
	var since, until *time.Time
	if err := api.pool.QueryRow(context.Background(),
		`SELECT pause_since, pause_until FROM users WHERE id = $1`, uid).Scan(&since, &until); err != nil {
		t.Fatalf("read pause window: %v", err)
	}
	if since == nil || until == nil {
		t.Fatalf("pause window after activation = since %v until %v, want both set", since, until)
	}

	var state string
	if err := api.pool.QueryRow(context.Background(),
		`SELECT state FROM escalation_jobs WHERE id = $1`, jobID).Scan(&state); err != nil {
		t.Fatalf("job state: %v", err)
	}
	if state != "cancelled" {
		t.Fatalf("pending job state after the pause = %s, want cancelled (the ladder freezes atomically, R-11)", state)
	}
	var audits int
	if err := api.pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_events WHERE user_id = $1 AND kind = 'pause.activated'`, uid).Scan(&audits); err != nil {
		t.Fatalf("audit count: %v", err)
	}
	if audits != 1 {
		t.Fatalf("pause.activated audit rows = %d, want 1", audits)
	}

	// Cancelling clears the whole window: the streak counts normally again.
	if del := callAPI(t, http.MethodDelete, api.server.URL+"/v1/me/pause", token, nil); del.StatusCode != http.StatusNoContent {
		t.Fatalf("cancel = %d, want 204", del.StatusCode)
	}
	if err := api.pool.QueryRow(context.Background(),
		`SELECT pause_since, pause_until FROM users WHERE id = $1`, uid).Scan(&since, &until); err != nil {
		t.Fatalf("read window after cancel: %v", err)
	}
	if since != nil || until != nil {
		t.Fatalf("window after cancel = since %v until %v, want both NULL", since, until)
	}
}
