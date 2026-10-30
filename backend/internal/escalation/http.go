package escalation

// The escalation HTTP surface (contract v0.3.0): the user's app reads the
// incident state so nobody discovers an incident after the fact (R-08/R-09).
// The web alert itself lives in internal/alert (public, token capability).

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/auth"
	"github.com/tiagofur/imalive/backend/internal/platform/httpapi"
)

// Handlers serve the authenticated incident endpoints.
type Handlers struct {
	engine *Engine
}

// NewHandlers wires the incident handlers with the engine.
func NewHandlers(engine *Engine) *Handlers { return &Handlers{engine: engine} }

// Register mounts the routes (all protected by the auth middleware).
func (h *Handlers) Register(mux *http.ServeMux, mw auth.Middleware) {
	mux.Handle("GET /v1/incidents/current", mw.Wrap(http.HandlerFunc(h.current)))
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
