// Package alert serves the actionable web alert (R-08): a server-rendered
// page, reached by the signed magic link that travels in the incident
// notifications, where a contact answers "Voy yo" or "No puedo" — without an
// app and without an account. The link is a stateless capability: the HMAC
// signature and the incident state machine (never token storage) make every
// action idempotent, and one claimant is THE claimant (R-09).
//
// The page is the trust face of the product for the circle: plain Spanish,
// one screen, buttons big enough for a scared 2 a.m. tap. No JavaScript.
package alert

import (
	"embed"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
	"time"

	"github.com/tiagofur/imalive/backend/internal/escalation"
)

//go:embed templates/*.html
var templatesFS embed.FS

// Handlers serve the alert page and its two actions. Public routes: the
// signed token IS the authentication (capability URL).
type Handlers struct {
	engine *escalation.Engine
	secret []byte
	tpl    *template.Template
	log    *slog.Logger
}

// New parses the embedded template once.
func New(engine *escalation.Engine, secret []byte, log *slog.Logger) (*Handlers, error) {
	tpl, err := template.ParseFS(templatesFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &Handlers{engine: engine, secret: secret, tpl: tpl, log: log}, nil
}

// Register mounts the public alert routes.
func (h *Handlers) Register(mux *http.ServeMux) {
	mux.Handle("GET /alert/{token}", http.HandlerFunc(h.page))
	mux.Handle("POST /alert/{token}/claim", http.HandlerFunc(h.claim))
	mux.Handle("POST /alert/{token}/decline", http.HandlerFunc(h.decline))
}

// pageView is the template's read model. The booleans select the section;
// the template never sees raw errors.
type pageView struct {
	Token         string
	UserName      string
	ClaimerName   string
	OpenedAt      string
	LastSignal    string
	ClosedMessage string
	Invalid       bool
	Expired       bool
	Closed        bool
}

// resolve verifies the token and loads the incident view. ok=false means the
// response is already written: an expired link renders its own page (410, no
// detail leaked), any other failure renders an invalid link (404).
func (h *Handlers) resolve(w http.ResponseWriter, r *http.Request) (escalation.IncidentView, string, bool) {
	token := r.PathValue("token")
	incidentID, _, _, err := escalation.VerifyAlertToken(h.secret, token, time.Now())
	if errors.Is(err, escalation.ErrExpiredToken) {
		h.render(w, pageView{Token: token, Expired: true}, http.StatusGone)
		return escalation.IncidentView{}, token, false
	}
	if err != nil {
		h.render(w, pageView{Token: token, Invalid: true}, http.StatusNotFound)
		return escalation.IncidentView{}, token, false
	}
	view, err := h.engine.View(r.Context(), incidentID)
	if err != nil {
		// A valid signature for an incident that does not exist: same answer
		// as a bad link, no oracle about what exists.
		h.render(w, pageView{Token: token, Invalid: true}, http.StatusNotFound)
		return escalation.IncidentView{}, token, false
	}
	return view, token, true
}

// page renders the current state of the incident: alert + buttons while it
// is open and unclaimed, "X está yendo" once claimed, the closure after.
func (h *Handlers) page(w http.ResponseWriter, r *http.Request) {
	view, token, ok := h.resolve(w, r)
	if !ok {
		return
	}
	v := pageView{
		Token:       token,
		UserName:    view.UserName,
		ClaimerName: view.Claimer,
		OpenedAt:    view.Incident.OpenedAt.Time.UTC().Format("02/01 15:04"),
	}
	if view.Incident.LastSignalAt.Valid {
		v.LastSignal = view.Incident.LastSignalAt.Time.UTC().Format("02/01 15:04")
	} else {
		v.LastSignal = "ninguna registrada"
	}
	switch view.Incident.State {
	case escalation.IncidentClosedResolved:
		v.Closed = true
		v.ClosedMessage = "todo en orden ✅"
	case escalation.IncidentClosedFalseAlarm:
		v.Closed = true
		v.ClosedMessage = "falsa alarma, " + view.UserName + " confirmó que está bien"
	}
	h.render(w, v, http.StatusOK)
}

// claim answers "Voy yo": the first contact wins (R-09), everyone else sees
// the winner. Post-Redirect-Get keeps refreshes harmless.
func (h *Handlers) claim(w http.ResponseWriter, r *http.Request) {
	view, token, ok := h.resolve(w, r)
	if !ok {
		return
	}
	_, contactID, _, err := escalation.VerifyAlertToken(h.secret, token, time.Now())
	if err != nil {
		h.render(w, pageView{Token: token, Invalid: true}, http.StatusNotFound)
		return
	}
	if _, err := h.engine.Claim(r.Context(), view.Incident.ID, contactID); err != nil {
		h.log.Error("alert: claim", slog.String("error", err.Error()))
		http.Error(w, "No se pudo registrar. Intenta de nuevo.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/alert/"+token, http.StatusSeeOther)
}

// decline answers "No puedo": the alert escalates to the whole circle right
// now and the pending ladder rung for the circle dies with it.
func (h *Handlers) decline(w http.ResponseWriter, r *http.Request) {
	view, token, ok := h.resolve(w, r)
	if !ok {
		return
	}
	if _, err := h.engine.Decline(r.Context(), view.Incident.ID); err != nil {
		h.log.Error("alert: decline", slog.String("error", err.Error()))
		http.Error(w, "No se pudo registrar. Intenta de nuevo.", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/alert/"+token, http.StatusSeeOther)
}

func (h *Handlers) render(w http.ResponseWriter, v pageView, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := h.tpl.ExecuteTemplate(w, "alert.html", v); err != nil {
		h.log.Error("alert: render", slog.String("error", err.Error()))
	}
}
