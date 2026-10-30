package escalation

// The escalation HTTP surface (contract v0.3.0): the user's app reads the
// incident state so nobody discovers an incident after the fact (R-08/R-09).
// The web alert itself lives in internal/alert (public, token capability).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tiagofur/imalive/backend/internal/auth"
	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/httpapi"
)

// inPoolTx runs fn in a transaction over the engine's pool.
func inPoolTx(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context, pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// PauseReader reports the live pause (R-11), the same source the check-in
// and profile handlers consult.
type PauseReader interface {
	PauseActive(ctx context.Context, userID uuid.UUID) (bool, time.Time)
}

// Handlers serve the authenticated incident endpoints.
type Handlers struct {
	engine *Engine
	pauses PauseReader
}

// NewHandlers wires the incident handlers with the engine and the pause
// reader (the SOS refuses to arm behind a live pause, with an explicit
// message).
func NewHandlers(engine *Engine, pauses PauseReader) *Handlers {
	return &Handlers{engine: engine, pauses: pauses}
}

// Register mounts the routes (all protected by the auth middleware).
func (h *Handlers) Register(mux *http.ServeMux, mw auth.Middleware) {
	mux.Handle("GET /v1/incidents/current", mw.Wrap(http.HandlerFunc(h.current)))
	mux.Handle("POST /v1/sos", mw.Wrap(http.HandlerFunc(h.armSos)))
	mux.Handle("DELETE /v1/sos", mw.Wrap(http.HandlerFunc(h.cancelSos)))
	mux.Handle("POST /v1/devices", mw.Wrap(http.HandlerFunc(h.registerDevice)))
	mux.Handle("DELETE /v1/devices", mw.Wrap(http.HandlerFunc(h.revokeDevice)))
}

// currentIncidentJSON mirrors the contract schema IncidentCurrent (v0.3.0).
type currentIncidentJSON struct {
	State         string     `json:"state"`
	CycleDate     string     `json:"cycleDate"`
	OpenedAt      time.Time  `json:"openedAt"`
	ClosedAt      *time.Time `json:"closedAt"`
	LastSeenAt    *time.Time `json:"lastSeenAt"`
	ClaimedByName *string    `json:"claimedByName"`
}

func (h *Handlers) current(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFromContext(r.Context()))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	view, found, err := h.engine.CurrentIncident(r.Context(), userID)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	inc := view.Incident
	out := currentIncidentJSON{
		State:      inc.State,
		CycleDate:  inc.CycleDate.Time.Format("2006-01-02"),
		OpenedAt:   inc.OpenedAt.Time.UTC(),
		ClosedAt:   nullableTime(inc.ClosedAt),
		LastSeenAt: nullableTime(inc.LastSignalAt),
	}
	if view.Claimer != "" {
		out.ClaimedByName = &view.Claimer
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

func nullableTime(t pgtype.Timestamptz) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time.UTC()
	return &v
}

// SosCountdown is the R-10 cancel window: the alert job is armed due in 10
// seconds; the worker tick (30 s) means the alert actually lands between
// t+10s and t+40s, and a cancel inside the window always wins.
const SosCountdown = 10 * time.Second

// armSos answers POST /v1/sos: arm the countdown (transactional, idempotent)
// unless the pause is live — the pause holds with an explicit message (R-11
// vs R-10: an emergency must not be silently swallowed by a paused ladder).
func (h *Handlers) armSos(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFromContext(r.Context()))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	if h.pauses != nil {
		if paused, _ := h.pauses.PauseActive(r.Context(), userID); paused {
			httpapi.WriteProblem(w, r, http.StatusConflict, httpapi.CodeSosPaused,
				"El modo pausa está activo: el SOS no enviará alertas. Cancela la pausa primero si es una emergencia real.")
			return
		}
	}
	cycleDate, err := h.cycleDate(r.Context(), userID)
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	runAt := time.Now().UTC().Add(SosCountdown)
	err = inPoolTx(r.Context(), h.engine.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := h.engine.SosArmTx(ctx, tx, userID, cycleDate, runAt)
		return err
	})
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"runAt": runAt})
}

// cancelSos answers DELETE /v1/sos: idempotent by design (a panicked second
// tap must never error) and audited by SosCancelTx.
func (h *Handlers) cancelSos(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFromContext(r.Context()))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	err = inPoolTx(r.Context(), h.engine.pool, func(ctx context.Context, tx pgx.Tx) error {
		_, err := h.engine.SosCancelTx(ctx, tx, userID)
		return err
	})
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// cycleDate resolves the user-local day (the SOS job's cycle anchor).
func (h *Handlers) cycleDate(ctx context.Context, userID uuid.UUID) (pgtype.Date, error) {
	tzName, err := h.engine.store.GetUserTimezone(ctx, userID)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("sos: timezone: %w", err)
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return pgtype.Date{}, fmt.Errorf("sos: timezone %q: %w", tzName, err)
	}
	local := time.Now().In(loc)
	return pgtype.Date{Time: local, Valid: true}, nil
}

// registerDevice answers POST /v1/devices: the FCM token of the user's
// device for the daily push reminders (R-05 salida). Re-registering
// replaces (one active device in the MVP).
func (h *Handlers) registerDevice(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFromContext(r.Context()))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	var body struct {
		Token string `json:"token"`
	}
	if err := httpapi.DecodeJSON(r, &body); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed, err.Error())
		return
	}
	token := strings.TrimSpace(body.Token)
	if token == "" || len(token) > 4096 {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
			"token is required (max 4096 characters).")
		return
	}
	if err := h.engine.store.SetUserPushToken(r.Context(), store.SetUserPushTokenParams{
		ID: userID, PushToken: &token,
	}); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// revokeDevice answers DELETE /v1/devices: clears the push token
// (idempotent — revoking twice changes nothing).
func (h *Handlers) revokeDevice(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(auth.UserIDFromContext(r.Context()))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	if err := h.engine.store.ClearUserPushToken(r.Context(), userID); err != nil {
		httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
