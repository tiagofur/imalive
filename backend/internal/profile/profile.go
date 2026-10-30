// Package profile implements R-02: the authenticated user's profile —
// GET /v1/me and the partial PATCH /v1/me. The store underneath is the
// sqlc-generated users store; the column-to-contract mapping is
// display_name -> name (documented since issue #38).
package profile

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/platform/httpapi"
)

// Handlers serve GET/PATCH /v1/me for the authenticated user.
type Handlers struct {
	pool    *pgxpool.Pool
	users   *authstore.Queries
	freezer LadderFreezer
}

// NewHandlers wires the handlers with the pool (the pause transaction), the
// users store and the escalation engine's freezer.
func NewHandlers(pool *pgxpool.Pool, users *authstore.Queries, freezer LadderFreezer) *Handlers {
	return &Handlers{pool: pool, users: users, freezer: freezer}
}

// inTx runs fn in a transaction, rolling back on error.
func inTx(ctx context.Context, pool *pgxpool.Pool, fn func(context.Context, pgx.Tx) error) error {
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

// Register mounts the routes wrapped by the auth middleware: all protected.
func (h *Handlers) Register(mux *http.ServeMux, mw auth.Middleware) {
	mux.Handle("GET /v1/me", mw.Wrap(http.HandlerFunc(h.get)))
	mux.Handle("PATCH /v1/me", mw.Wrap(http.HandlerFunc(h.patch)))
	mux.Handle("PUT /v1/me/pause", mw.Wrap(http.HandlerFunc(h.SetPause)))
	mux.Handle("DELETE /v1/me/pause", mw.Wrap(http.HandlerFunc(h.CancelPause)))
}

// profilePatch mirrors the contract's ProfilePatch: pointer fields keep the
// omitted-fields-keep-values semantics. An empty object is rejected (the
// contract sets minProperties: 1).
type profilePatch struct {
	Name                    *string `json:"name"`
	CheckinTime             *string `json:"checkinTime"`
	Timezone                *string `json:"timezone"`
	EscalationWindowMinutes *int    `json:"escalationWindowMinutes"`
}

func (h *Handlers) get(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	user, err := h.users.GetUser(r.Context(), uuid.MustParse(userID))
	if err != nil {
		internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profileOf(user))
}

func (h *Handlers) patch(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var patch profilePatch
	if err := httpapi.DecodeJSON(r, &patch); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed, err.Error())
		return
	}
	if patch.Name == nil && patch.CheckinTime == nil && patch.Timezone == nil && patch.EscalationWindowMinutes == nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
			"At least one field is required (the contract sets minProperties: 1).")
		return
	}

	params := authstore.UpdateUserProfileParams{ID: uuid.MustParse(userID)}
	if patch.Name != nil {
		if n := len(*patch.Name); n < 1 || n > 80 {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
				"name must be 1..80 characters.")
			return
		}
		params.DisplayName = patch.Name
	}
	if patch.CheckinTime != nil {
		if !checkinTimePattern.MatchString(*patch.CheckinTime) {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
				"checkinTime must be HH:MM (24h), e.g. 21:00.")
			return
		}
		params.CheckinTime = pgTime(*patch.CheckinTime)
	}
	if patch.Timezone != nil {
		if _, err := time.LoadLocation(*patch.Timezone); err != nil {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
				"timezone must be a valid IANA name, e.g. America/Mexico_City.")
			return
		}
		params.Timezone = patch.Timezone
	}
	if patch.EscalationWindowMinutes != nil {
		if m := *patch.EscalationWindowMinutes; m < 0 || m > 1440 {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
				"escalationWindowMinutes must be 0..1440.")
			return
		}
		m := int16(*patch.EscalationWindowMinutes)
		params.EscalationWindowMinutes = &m
	}

	user, err := h.users.UpdateUserProfile(r.Context(), params)
	if err != nil {
		internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profileOf(user))
}

var checkinTimePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// profileOf maps the store row to the contract's UserProfile schema.
// PauseUntil is additive in contract v0.2.0 (issue #53).
func profileOf(u authstore.User) map[string]any {
	var pauseUntil any
	if u.PauseUntil.Valid {
		pauseUntil = u.PauseUntil.Time.UTC()
	}
	return map[string]any{
		"id":                      u.ID,
		"name":                    u.DisplayName,
		"phone":                   u.Phone,
		"checkinTime":             hhmm(u.CheckinTime),
		"timezone":                u.Timezone,
		"escalationWindowMinutes": u.EscalationWindowMinutes,
		"pauseUntil":              pauseUntil,
	}
}

// hhmm renders the stored microseconds-since-midnight as "HH:MM" (contract
// pattern). A zero-value time (unstored edge) renders "00:00".
func hhmm(t pgtype.Time) string {
	if !t.Valid {
		return "00:00"
	}
	total := time.Duration(t.Microseconds) * time.Microsecond
	return fmt.Sprintf("%02d:%02d", int(total/time.Hour), int((total%time.Hour)/time.Minute))
}

// pgTime converts an "HH:MM" string to the pgtype the store expects
// (microseconds since midnight).
func pgTime(hhmm string) pgtype.Time {
	t, err := time.Parse("15:04", hhmm)
	if err != nil { // unreachable: the caller validated the pattern first
		return pgtype.Time{Valid: false}
	}
	micro := int64(t.Hour())*3_600_000_000 + int64(t.Minute())*60_000_000
	return pgtype.Time{Microseconds: micro, Valid: true}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func internal(w http.ResponseWriter, r *http.Request, _ error) {
	httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
}
