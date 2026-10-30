// Contract tests of GET /v1/incidents/current (v0.3.0): the app reads the
// incident state through the real auth stack.
package escalation_test

import (
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
	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
)

// pauseStub implements escalation.PauseReader with a settable state.
type pauseStub struct{ active bool }

func (p *pauseStub) PauseActive(context.Context, uuid.UUID) (bool, time.Time) {
	if !p.active {
		return false, time.Time{}
	}
	return true, time.Now().Add(time.Hour)
}

func TestSosEndpoints(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	recorder := &senderRecorder{codes: map[string]string{}}
	svc := auth.NewService(otp.NewStore(rdb, "test-secret"), recorder, rdb,
		authstore.New(p.Pool), "test-secret")
	engine := escalation.New(p.Pool, escalation.SystemClock{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	pauses := &pauseStub{}
	mux := http.NewServeMux()
	auth.NewHandlers(svc, auth.NewRateLimiter(rdb)).Register(mux)
	escalation.NewHandlers(engine, pauses).Register(mux, auth.NewMiddleware(svc))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	phone := "+5215596000001"
	if err := svc.Request(ctx, phone); err != nil {
		t.Fatalf("Request: %v", err)
	}
	pair, err := svc.Verify(ctx, phone, recorder.codes[phone])
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	token := pair.AccessToken
	var userID uuid.UUID
	if err := p.Pool.QueryRow(ctx, `SELECT id FROM users WHERE phone = $1`, phone).Scan(&userID); err != nil {
		t.Fatalf("find user: %v", err)
	}

	call := func(method string) *http.Response {
		t.Helper()
		req, err := http.NewRequestWithContext(ctx, method, srv.URL+"/v1/sos", nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("%s /v1/sos: %v", method, err)
		}
		return resp
	}

	t.Run("POST arms the countdown transactionally", func(t *testing.T) {
		resp := call(http.MethodPost)
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("status = %d, want 201", resp.StatusCode)
		}
		var got struct {
			RunAt time.Time `json:"runAt"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got.RunAt.Before(time.Now().UTC().Add(5 * time.Second)) {
			t.Fatalf("runAt = %v, want about now+10s", got.RunAt)
		}
		var pending int
		if err := p.Pool.QueryRow(ctx,
			`SELECT count(*) FROM escalation_jobs WHERE user_id = $1 AND kind = 'sos_alert' AND state = 'pending'`,
			userID).Scan(&pending); err != nil || pending != 1 {
			t.Fatalf("pending sos jobs = %d (err %v), want 1", pending, err)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditSosArmed); n != 1 {
			t.Fatalf("sos.armed audit rows = %d, want 1", n)
		}

		// A second POST is the same countdown, not a stacked one.
		resp2 := call(http.MethodPost)
		defer func() { _ = resp2.Body.Close() }()
		if resp2.StatusCode != http.StatusCreated {
			t.Fatalf("second POST = %d, want 201", resp2.StatusCode)
		}
		if err := p.Pool.QueryRow(ctx,
			`SELECT count(*) FROM escalation_jobs WHERE user_id = $1 AND kind = 'sos_alert' AND state = 'pending'`,
			userID).Scan(&pending); err != nil || pending != 1 {
			t.Fatalf("pending after re-arm = %d (err %v), want 1", pending, err)
		}
	})

	t.Run("DELETE cancels idempotently with evidence", func(t *testing.T) {
		for i := 0; i < 2; i++ { // the panicked double tap must never error
			resp := call(http.MethodDelete)
			defer func() { _ = resp.Body.Close() }()
			if resp.StatusCode != http.StatusNoContent {
				t.Fatalf("DELETE = %d, want 204", resp.StatusCode)
			}
		}
		var pending int
		if err := p.Pool.QueryRow(ctx,
			`SELECT count(*) FROM escalation_jobs WHERE user_id = $1 AND kind = 'sos_alert' AND state = 'pending'`,
			userID).Scan(&pending); err != nil || pending != 0 {
			t.Fatalf("pending after cancel = %d (err %v), want 0", pending, err)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditSosCancelled); n != 1 {
			t.Fatalf("sos.cancelled audit rows = %d, want 1", n)
		}
	})

	t.Run("devices: register, replace, revoke", func(t *testing.T) {
		post := func(payload string) *http.Response {
			t.Helper()
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, srv.URL+"/v1/devices", strings.NewReader(payload))
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("POST /v1/devices: %v", err)
			}
			return resp
		}
		del := func() *http.Response {
			t.Helper()
			req, err := http.NewRequestWithContext(ctx, http.MethodDelete, srv.URL+"/v1/devices", nil)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			req.Header.Set("Authorization", "Bearer "+token)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("DELETE /v1/devices: %v", err)
			}
			return resp
		}
		tokenOf := func() string {
			t.Helper()
			var tok *string
			if err := p.Pool.QueryRow(ctx, `SELECT push_token FROM users WHERE id = $1`, userID).Scan(&tok); err != nil {
				t.Fatalf("read token: %v", err)
			}
			if tok == nil {
				return "<NULL>"
			}
			return *tok
		}

		resp := post(`{"token":"fcm-token-beta"}`)
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("register = %d, want 204", resp.StatusCode)
		}
		if got := tokenOf(); got != "fcm-token-beta" {
			t.Fatalf("stored token = %v, want fcm-token-beta", got)
		}

		// Re-registering replaces; an empty token is rejected.
		resp2 := post(`{"token":"fcm-token-gamma"}`)
		defer func() { _ = resp2.Body.Close() }()
		if resp2.StatusCode != http.StatusNoContent {
			t.Fatalf("replace = %d, want 204", resp2.StatusCode)
		}
		if got := tokenOf(); got != "fcm-token-gamma" {
			t.Fatalf("replaced token = %v, want fcm-token-gamma", got)
		}
		resp3 := post(`{"token":"  "}`)
		defer func() { _ = resp3.Body.Close() }()
		if resp3.StatusCode != http.StatusBadRequest {
			t.Fatalf("empty token = %d, want 400", resp3.StatusCode)
		}

		for i := 0; i < 2; i++ { // revoking twice is fine
			r := del()
			defer func() { _ = r.Body.Close() }()
			if r.StatusCode != http.StatusNoContent {
				t.Fatalf("revoke = %d, want 204", r.StatusCode)
			}
		}
		if got := tokenOf(); got != "<NULL>" {
			t.Fatalf("token after revoke = %v, want NULL", got)
		}
	})

	t.Run("a live pause refuses the SOS with an explicit message", func(t *testing.T) {
		pauses.active = true
		defer func() { pauses.active = false }()
		resp := call(http.MethodPost)
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusConflict {
			t.Fatalf("status = %d, want 409", resp.StatusCode)
		}
		var got map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got["code"] != "SOS_PAUSED" {
			t.Fatalf("problem = %v, want code SOS_PAUSED", got)
		}
		var pending int
		if err := p.Pool.QueryRow(ctx,
			`SELECT count(*) FROM escalation_jobs WHERE user_id = $1 AND kind = 'sos_alert' AND state = 'pending'`,
			userID).Scan(&pending); err != nil || pending != 0 {
			t.Fatalf("paused POST armed %d jobs (err %v), want 0", pending, err)
		}
	})
}

func TestIncidentCurrent(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	recorder := &senderRecorder{codes: map[string]string{}}
	svc := auth.NewService(otp.NewStore(rdb, "test-secret"), recorder, rdb,
		authstore.New(p.Pool), "test-secret")
	engine := escalation.New(p.Pool, escalation.SystemClock{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	mux := http.NewServeMux()
	auth.NewHandlers(svc, auth.NewRateLimiter(rdb)).Register(mux)
	escalation.NewHandlers(engine, nil).Register(mux, auth.NewMiddleware(svc))
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	signUp := func(phone string) string {
		t.Helper()
		if err := svc.Request(ctx, phone); err != nil {
			t.Fatalf("Request: %v", err)
		}
		pair, err := svc.Verify(ctx, phone, recorder.codes[phone])
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		return pair.AccessToken
	}

	t.Run("no incidents yet answers 204", func(t *testing.T) {
		token := signUp("+5215590000001")
		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/incidents/current", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", resp.StatusCode)
		}
	})

	t.Run("the live incident comes back with its state", func(t *testing.T) {
		phone := "+5215590000002"
		token := signUp(phone)
		var userID uuid.UUID
		if err := p.Pool.QueryRow(ctx, `SELECT id FROM users WHERE phone = $1`, phone).Scan(&userID); err != nil {
			t.Fatalf("find user: %v", err)
		}
		inc, err := store.New(p.Pool).CreateIncident(ctx, store.CreateIncidentParams{
			ID: uuid.Must(uuid.NewV7()), UserID: userID,
			CycleDate: pgDate(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)),
		})
		if err != nil {
			t.Fatalf("seed incident: %v", err)
		}

		req, _ := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/incidents/current", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		var got map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got["state"] != escalation.IncidentContact1Notified {
			t.Fatalf("state = %v, want contact1_notified", got["state"])
		}
		if got["cycleDate"] != "2026-10-30" || got["claimedByName"] != nil {
			t.Fatalf("body = %v, want the cycle date and no claimant", got)
		}

		// The claim surfaces by name (R-09: "María está yendo").
		contact := uuid.Must(uuid.NewV7())
		if _, err := p.Pool.Exec(ctx,
			`INSERT INTO contacts (id, user_id, display_name, phone, position)
			 VALUES ($1, $2, 'María', '+5215590000021', 1)`, contact, userID); err != nil {
			t.Fatalf("seed contact: %v", err)
		}
		if _, err := store.New(p.Pool).ClaimIncident(ctx, store.ClaimIncidentParams{
			ID: inc.ID, ClaimedBy: pgUUID(contact),
		}); err != nil {
			t.Fatalf("claim: %v", err)
		}
		resp2, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET after claim: %v", err)
		}
		defer func() { _ = resp2.Body.Close() }()
		var got2 map[string]any
		if err := json.NewDecoder(resp2.Body).Decode(&got2); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if got2["state"] != escalation.IncidentReclaimed || got2["claimedByName"] != "María" {
			t.Fatalf("body = %v, want reclaimed by María", got2)
		}
	})

	t.Run("no token yields 401", func(t *testing.T) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/v1/incidents/current", nil)
		if err != nil {
			t.Fatalf("request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("GET: %v", err)
		}
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", resp.StatusCode)
		}
	})
}

type senderRecorder struct{ codes map[string]string }

func (s *senderRecorder) Send(_ context.Context, phone, code string) error {
	s.codes[phone] = code
	return nil
}
