// Package checkin implements R-04: today's "I'm OK" signal. The HTTP layer
// (this file) maps the contract routes to the sqlc store; the cycle date is
// computed in the user's timezone (see migration 000003 and the package doc).
package checkin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	auth "github.com/tiagofur/imalive/backend/internal/auth"
	"github.com/tiagofur/imalive/backend/internal/checkin/store"
	"github.com/tiagofur/imalive/backend/internal/platform/httpapi"
)

// Handlers serve POST /v1/checkins and GET /v1/checkins/today.
type Handlers struct {
	checkins *store.Queries
}

// NewHandlers wires the check-in handlers with the sqlc store.
func NewHandlers(checkins *store.Queries) *Handlers {
	return &Handlers{checkins: checkins}
}

// Register mounts the routes wrapped by the auth middleware (both protected).
func (h *Handlers) Register(mux *http.ServeMux, mw auth.Middleware) {
	mux.Handle("POST /v1/checkins", mw.Wrap(http.HandlerFunc(h.create)))
	mux.Handle("GET /v1/checkins/today", mw.Wrap(http.HandlerFunc(h.today)))
}

// MaxIdempotencyKey is the contract's Idempotency-Key header ceiling.
const MaxIdempotencyKey = 64

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	key := r.Header.Get("Idempotency-Key")
	if len(key) > MaxIdempotencyKey {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
			"Idempotency-Key must be at most 64 characters.")
		return
	}

	cycleDate, tz, err := h.cycleDate(r.Context(), userID)
	if err != nil {
		internal(w, r, err)
		return
	}
	now := time.Now().UTC()

	if key != "" {
		if replay, err := h.checkins.GetCheckinByIdempotencyKey(r.Context(), store.GetCheckinByIdempotencyKeyParams{
			UserID:         uuid.MustParse(userID),
			IdempotencyKey: &key,
		}); err == nil {
			writeJSON(w, http.StatusOK, checkinJSON(replay, true))
			return
		} else if !errors.Is(err, pgx.ErrNoRows) {
			internal(w, r, err)
			return
		}
	}

	created, err := h.checkins.CreateCheckin(r.Context(), store.CreateCheckinParams{
		ID:             newUUID(),
		UserID:         uuid.MustParse(userID),
		CycleDate:      cycleDate,
		CheckedInAt:    pgTz(now),
		IdempotencyKey: keyPtr(key),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		// Same-day replay (with or without a key) and idempotency races both
		// surface as unique violations: answer with the existing signal.
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			existing, getErr := h.checkins.GetCheckinByDate(r.Context(), store.GetCheckinByDateParams{
				UserID: uuid.MustParse(userID), CycleDate: cycleDate,
			})
			if getErr != nil {
				internal(w, r, getErr)
				return
			}
			writeJSON(w, http.StatusOK, checkinJSON(existing, true))
			return
		}
		internal(w, r, err)
		return
	}
	_ = tz
	writeJSON(w, http.StatusCreated, checkinJSON(created, false))
}

func (h *Handlers) today(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	cycleDate, tz, err := h.cycleDate(r.Context(), userID)
	if err != nil {
		internal(w, r, err)
		return
	}

	status := map[string]any{"date": cycleDate.Time.Format("2006-01-02"), "status": "pending"}
	if row, err := h.checkins.GetCheckinByDate(r.Context(), store.GetCheckinByDateParams{
		UserID: uuid.MustParse(userID), CycleDate: cycleDate,
	}); err == nil {
		status["status"] = "confirmed"
		status["checkedInAt"] = row.CheckedInAt.Time.UTC()
	} else if !errors.Is(err, pgx.ErrNoRows) {
		internal(w, r, err)
		return
	}
	status["nextCheckinAt"] = h.nextCheckinAt(r.Context(), uuid.MustParse(userID), tz, cycleDate)
	writeJSON(w, http.StatusOK, status)
}

// cycleDate resolves today's cycle date in the user's timezone (migration
// 000003: a UTC date would flip at mid-afternoon in LATAM).
func (h *Handlers) cycleDate(ctx context.Context, userID string) (pgtype.Date, *time.Location, error) {
	tzName, err := h.checkins.GetUserTimezone(ctx, uuid.MustParse(userID))
	if err != nil {
		return pgtype.Date{}, nil, fmt.Errorf("checkin: timezone: %w", err)
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		// The profile validates IANA names; a corrupt row fails loud.
		return pgtype.Date{}, nil, fmt.Errorf("checkin: timezone %q: %w", tzName, err)
	}
	local := time.Now().In(loc)
	return pgtype.Date{Time: local, Valid: true}, loc, nil
}

// nextCheckinAt computes the next expected signal instant in UTC from the
// user's configured check-in time: today's occurrence when still ahead,
// tomorrow's otherwise (S1 engine-free computation, contract example).
func (h *Handlers) nextCheckinAt(ctx context.Context, userID uuid.UUID, loc *time.Location, cycleDate pgtype.Date) time.Time {
	minutes := h.checkinMinutes(ctx, userID)
	target := time.Date(cycleDate.Time.Year(), cycleDate.Time.Month(), cycleDate.Time.Day(),
		minutes/60, minutes%60, 0, 0, loc)
	if !time.Now().In(loc).Before(target) {
		target = target.Add(24 * time.Hour)
	}
	return target.UTC()
}

// checkinMinutes reads the user's configured check-in time as minutes from
// midnight (product default 21:00, the users.checkin_time column default).
func (h *Handlers) checkinMinutes(ctx context.Context, userID uuid.UUID) int {
	t, err := h.checkins.GetUserCheckinTime(ctx, userID)
	if err != nil || !t.Valid {
		return 21 * 60
	}
	total := t.Microseconds / 1_000_000
	return int(total/60) % (24 * 60)
}

func checkinJSON(row store.Checkin, replayed bool) map[string]any {
	return map[string]any{
		"date":        row.CycleDate.Time.Format("2006-01-02"),
		"checkedInAt": row.CheckedInAt.Time.UTC(),
		"replayed":    replayed,
	}
}

func keyPtr(key string) *string {
	if key == "" {
		return nil
	}
	return &key
}

func newUUID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic("checkin: uuid v7: " + err.Error()) // crypto/rand exhaustion is not recoverable
	}
	return id
}

func pgTz(t time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: t, Valid: true} }

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func internal(w http.ResponseWriter, r *http.Request, _ error) {
	httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
}
