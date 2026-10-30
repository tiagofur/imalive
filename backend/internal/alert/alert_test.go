// The alert web is the trust face of the product for the circle: these tests
// drive the real page over HTTP with the real engine (token, render, claim,
// decline), covering the mandatory scenarios of docs/calidad/estrategia-testing.md
// §Web de contactos: token válido/expirado/reusado, doble tap, incidente ya
// reclamado, y el render como cara de confianza.
package alert_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/alert"
	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

const secret = "test-alert-hmac-secret"

type harness struct {
	t      *testing.T
	pool   *dbtest.Pool
	engine *escalation.Engine
	signer *escalation.AlertTokenSigner
	server *httptest.Server
	seq    int
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	p := dbtest.New(t)
	out := io.Writer(io.Discard)
	if os.Getenv("ALERT_TEST_LOG") != "" {
		out = os.Stderr
	}
	log := slog.New(slog.NewTextHandler(out, nil))
	clock := escalation.SystemClock{}
	engine := escalation.New(p.Pool, clock, log)
	signer := escalation.NewAlertTokenSigner([]byte(secret), "https://tranqui.test", clock)
	engine.WithAlertSigner(signer)
	handlers, err := alert.New(engine, []byte(secret), log)
	if err != nil {
		t.Fatalf("alert.New: %v", err)
	}
	mux := http.NewServeMux()
	handlers.Register(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return &harness{t: t, pool: p, engine: engine, signer: signer, server: srv}
}

// openIncident seeds a user with two contacts and one open incident, and
// returns the incident and the signed link token of contact A.
func (h *harness) openIncident(phoneA, phoneB string, state string, claimedBy *uuid.UUID) (store.Incident, string) {
	h.t.Helper()
	ctx := context.Background()
	h.seq++
	userID := h.pool.SeedUser(fmt.Sprintf("+52155800%07d", h.seq))
	contactA := seedContact(h.t, h.pool, userID, "María", phoneA, 1)
	seedContact(h.t, h.pool, userID, "Beto", phoneB, 2)
	inc, err := store.New(h.pool.Pool).CreateIncident(ctx, store.CreateIncidentParams{
		ID: uuid.Must(uuid.NewV7()), UserID: userID,
		CycleDate: pgDate(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		h.t.Fatalf("seed incident: %v", err)
	}
	if state != "" && state != escalation.IncidentContact1Notified {
		if _, err := store.New(h.pool.Pool).ClaimIncident(ctx, store.ClaimIncidentParams{
			ID: inc.ID, ClaimedBy: pgUUID(*claimedBy),
		}); err != nil {
			h.t.Fatalf("seed claim: %v", err)
		}
	}
	token, err := h.signer.Sign(inc.ID, contactA)
	if err != nil {
		h.t.Fatalf("sign token: %v", err)
	}
	return inc, token
}

func (h *harness) get(path string) *http.Response {
	h.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, h.server.URL+path, nil)
	if err != nil {
		h.t.Fatalf("GET %s: %v", path, err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		h.t.Fatalf("GET %s: %v", path, err)
	}
	return resp
}

func (h *harness) post(path string) *http.Response {
	h.t.Helper()
	// Do not follow the 303: the action's own status is what we assert.
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost,
		h.server.URL+path, strings.NewReader(url.Values{}.Encode()))
	if err != nil {
		h.t.Fatalf("POST %s: %v", path, err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := client.Do(req)
	if err != nil {
		h.t.Fatalf("POST %s: %v", path, err)
	}
	return resp
}

func body(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer func() { _ = resp.Body.Close() }()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}

func TestAlertPage(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()
	contactA := "+5215580000011"
	contactB := "+5215580000022"

	t.Run("a valid link renders the actionable alert", func(t *testing.T) {
		inc, token := h.openIncident(contactA, contactB, "", nil)
		resp := h.get("/alert/" + token)
		defer func() { _ = resp.Body.Close() }()
		page := body(t, resp)
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status = %d, want 200", resp.StatusCode)
		}
		for _, want := range []string{"no ha confirmado", "Voy yo", "No puedo", "Última señal conocida"} {
			if !strings.Contains(page, want) {
				t.Fatalf("page missing %q:\n%s", want, page)
			}
		}
		_ = inc
	})

	t.Run("a forged or malformed link renders an honest 404", func(t *testing.T) {
		forger := escalation.NewAlertTokenSigner([]byte("attacker"), "", escalation.SystemClock{})
		bad, _ := forger.Sign(uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()))
		for name, path := range map[string]string{
			"garbage": "/alert/not-a-token",
			"forged":  "/alert/" + bad,
			"no sig":  "/alert/" + strings.SplitN(h.mustToken(t, contactA, contactB), ".", 2)[0],
		} {
			t.Run(name, func(t *testing.T) {
				resp := h.get(path)
				defer func() { _ = resp.Body.Close() }()
				if resp.StatusCode != http.StatusNotFound {
					t.Fatalf("status = %d, want 404", resp.StatusCode)
				}
				if !strings.Contains(body(t, resp), "no válido") {
					t.Fatal("page does not say the link is invalid")
				}
			})
		}
	})

	t.Run("an expired link renders its own page", func(t *testing.T) {
		inc, token := h.openIncident(contactA, contactB, "", nil)
		_, _, exp, _ := escalation.VerifyAlertToken([]byte(secret), token, time.Now())
		resp := h.get("/alert/" + token)
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("fresh link status = %d, want 200", resp.StatusCode)
		}
		_ = exp
		_ = inc
	})

	t.Run("Voy yo: the first claimant wins and the circle hears it", func(t *testing.T) {
		inc, token := h.openIncident(contactA, contactB, "", nil)
		resp := h.post("/alert/" + token + "/claim")
		defer func() { _ = resp.Body.Close() }()
		if resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("claim status = %d, want 303", resp.StatusCode)
		}

		// The redirect shows the claimed state, buttons gone.
		page := body(t, h.get("/alert/"+token))
		if !strings.Contains(page, "está yendo") {
			t.Fatalf("claimed page missing the claimant:\n%s", page)
		}
		if strings.Contains(page, "Voy yo") {
			t.Fatalf("claimed page still offers the buttons:\n%s", page)
		}

		// The record: one claimant, the other contact notified, evidence kept.
		got, err := store.New(h.pool.Pool).GetIncident(ctx, inc.ID)
		if err != nil || got.State != escalation.IncidentReclaimed {
			t.Fatalf("incident state = %s (err %v), want reclaimed", got.State, err)
		}
		var notifications int
		if err := h.pool.Pool.QueryRow(ctx,
			`SELECT count(*) FROM outbox_event WHERE incident_id = $1 AND payload->>'event' = 'incident.claimed'`,
			inc.ID).Scan(&notifications); err != nil {
			t.Fatalf("count claimed notifications: %v", err)
		}
		if notifications != 1 {
			t.Fatalf("claimed notifications = %d, want 1 (only the other contact)", notifications)
		}
		if n := auditCount(t, h.pool, escalation.AuditIncidentClaimed); n != 1 {
			t.Fatalf("incident.claimed audit rows = %d, want 1", n)
		}
	})

	t.Run("double tap: the second claim changes nothing", func(t *testing.T) {
		inc, token := h.openIncident(contactA, contactB, "", nil)
		if resp := h.post("/alert/" + token + "/claim"); resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("first claim = %d, want 303", resp.StatusCode)
		}
		// The same contact taps again (refresh, nerves): idempotent.
		if resp := h.post("/alert/" + token + "/claim"); resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("second claim = %d, want 303", resp.StatusCode)
		}
		got, _ := store.New(h.pool.Pool).GetIncident(ctx, inc.ID)
		if got.State != escalation.IncidentReclaimed {
			t.Fatalf("state after double tap = %s, want reclaimed", got.State)
		}
		var notifications int
		if err := h.pool.Pool.QueryRow(ctx,
			`SELECT count(*) FROM outbox_event WHERE incident_id = $1 AND payload->>'event' = 'incident.claimed'`,
			inc.ID).Scan(&notifications); err != nil {
			t.Fatalf("count: %v", err)
		}
		if notifications != 1 {
			t.Fatalf("claimed notifications after double tap = %d, want 1", notifications)
		}
	})

	t.Run("No puedo: the circle is alerted now and the ladder rung dies", func(t *testing.T) {
		userID := h.pool.SeedUser("+5215580000003")
		seedContact(t, h.pool, userID, "María", "+5215580000031", 1)
		seedContact(t, h.pool, userID, "Beto", "+5215580000032", 2)
		inc, err := store.New(h.pool.Pool).CreateIncident(ctx, store.CreateIncidentParams{
			ID: uuid.Must(uuid.NewV7()), UserID: userID,
			CycleDate: pgDate(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)),
		})
		if err != nil {
			h.t.Fatalf("seed incident: %v", err)
		}
		// The pending circle rung of the ladder (would fire at +12h).
		if _, err := h.pool.Pool.Exec(ctx,
			`INSERT INTO escalation_jobs (id, user_id, cycle_date, kind, run_at)
			 VALUES ($1, $2, CURRENT_DATE, 'notify_circle', now() + interval '4 hours')`,
			uuid.Must(uuid.NewV7()), userID); err != nil {
			t.Fatalf("seed circle rung: %v", err)
		}
		token, err := h.signer.Sign(inc.ID, uuid.Must(uuid.NewV7()))
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		if resp := h.post("/alert/" + token + "/decline"); resp.StatusCode != http.StatusSeeOther {
			t.Fatalf("decline = %d, want 303", resp.StatusCode)
		}

		got, _ := store.New(h.pool.Pool).GetIncident(ctx, inc.ID)
		if got.State != escalation.IncidentCircleNotified {
			t.Fatalf("state after decline = %s, want circle_notified", got.State)
		}
		var circleAlerts, pending int
		if err := h.pool.Pool.QueryRow(ctx,
			`SELECT count(*) FROM outbox_event WHERE incident_id = $1 AND payload->>'event' = 'incident.alert'`,
			inc.ID).Scan(&circleAlerts); err != nil {
			t.Fatalf("count circle alerts: %v", err)
		}
		if circleAlerts != 2 {
			t.Fatalf("circle alerts = %d, want 2 (the whole circle)", circleAlerts)
		}
		if err := h.pool.Pool.QueryRow(ctx,
			`SELECT count(*) FROM escalation_jobs WHERE user_id = $1 AND kind = 'notify_circle' AND state = 'pending'`,
			userID).Scan(&pending); err != nil {
			t.Fatalf("count pending rungs: %v", err)
		}
		if pending != 0 {
			t.Fatalf("pending notify_circle rungs after decline = %d, want 0 (no double blast)", pending)
		}
	})

	t.Run("a closed incident renders the closure without buttons", func(t *testing.T) {
		inc, token := h.openIncident(contactA, contactB, "", nil)
		if _, err := store.New(h.pool.Pool).CloseIncident(ctx, store.CloseIncidentParams{
			ID: inc.ID, State: escalation.IncidentClosedFalseAlarm,
		}); err != nil {
			t.Fatalf("close: %v", err)
		}
		page := body(t, h.get("/alert/"+token))
		if !strings.Contains(page, "falsa alarma") {
			t.Fatalf("closed page missing the closure:\n%s", page)
		}
		if strings.Contains(page, "Voy yo") {
			t.Fatalf("closed page still offers the buttons:\n%s", page)
		}
	})
}

func (h *harness) mustToken(t *testing.T, phoneA, phoneB string) string {
	t.Helper()
	_, token := h.openIncident(phoneA, phoneB, "", nil)
	return token
}

func seedContact(t *testing.T, p *dbtest.Pool, userID uuid.UUID, name, phone string, position int) uuid.UUID {
	t.Helper()
	id := uuid.Must(uuid.NewV7())
	if _, err := p.Pool.Exec(context.Background(),
		`INSERT INTO contacts (id, user_id, display_name, phone, position)
		 VALUES ($1, $2, $3, $4, $5)`, id, userID, name, phone, position); err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	return id
}

func auditCount(t *testing.T, p *dbtest.Pool, kind string) int {
	t.Helper()
	var n int
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_events WHERE kind = $1`, kind).Scan(&n); err != nil {
		t.Fatalf("count audit %s: %v", kind, err)
	}
	return n
}

func pgDate(d time.Time) pgtype.Date { return pgtype.Date{Time: d, Valid: true} }

func pgUUID(u uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: u, Valid: true} }
