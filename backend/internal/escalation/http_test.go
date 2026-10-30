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
	escalation.NewHandlers(engine).Register(mux, auth.NewMiddleware(svc))
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
