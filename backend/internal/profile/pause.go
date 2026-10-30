package profile

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/platform/httpapi"
)

// SetPause activates the pause/viaje mode (R-11): the escalation ladder
// freezes until the given instant (must be an RFC 3339 instant in the
// future). The streak freezes instead of breaking while the pause is live.
func (h *Handlers) SetPause(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Until time.Time `json:"until"`
	}
	if err := httpapi.DecodeJSON(r, &body); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed, err.Error())
		return
	}
	if body.Until.IsZero() || !body.Until.After(time.Now()) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
			"until must be an RFC 3339 instant in the future.")
		return
	}
	uid := uuid.MustParse(auth.UserIDFromContext(r.Context()))
	err := h.users.SetUserPause(r.Context(), authstore.SetUserPauseParams{
		ID:         uid,
		PauseUntil: pgtype.Timestamptz{Time: body.Until.UTC(), Valid: true},
	})
	if err != nil {
		internal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// CancelPause cancels the active pause; the ladder resumes immediately.
func (h *Handlers) CancelPause(w http.ResponseWriter, r *http.Request) {
	uid := uuid.MustParse(auth.UserIDFromContext(r.Context()))
	err := h.users.SetUserPause(r.Context(), authstore.SetUserPauseParams{ID: uid})
	if err != nil {
		internal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// PauseActive reports whether the user has a live pause and until when
// (the escalation engine consults this; issue #54).
func (h *Handlers) PauseActive(ctx context.Context, userID uuid.UUID) (bool, time.Time) {
	until, err := h.users.GetPauseUntil(ctx, userID)
	if err != nil || !until.Valid {
		return false, time.Time{}
	}
	return until.Time.After(time.Now()), until.Time
}
