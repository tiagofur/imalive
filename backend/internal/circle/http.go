package circle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/tiagofur/imalive/backend/internal/auth"
	"github.com/tiagofur/imalive/backend/internal/circle/store"
	"github.com/tiagofur/imalive/backend/internal/platform/httpapi"
	"github.com/tiagofur/imalive/backend/internal/platform/msg"
)

// AppName is the work name until issue #5 decides the final one (the welcome
// template parameterizes it).
const AppName = "Tranqui"

// welcomeMessageV1 renders the welcome text (R-03) per the UX brief of
// issue #41: an anti-CTA — it asks nothing, explains the service in one
// sentence, declares its honest limit and promises only the unusual. The
// version suffix pins the template for the sandbox records and future
// analytics; text changes ship as V2.
func welcomeMessageV1(inviter, contact, app string) string {
	greeting := "Hola"
	if contact != "" {
		greeting = fmt.Sprintf("Hola %s", contact)
	}
	return fmt.Sprintf(
		"%s: %s te ha añadido como persona de confianza en %s, la app con la que confirma a diario que está bien. No tienes que hacer nada. Solo te escribiremos si %s pasa muchas horas sin confirmar, para que puedas comprobar cómo está. %s es un servicio de avisos entre personas; no es un servicio de emergencias.",
		greeting, inviter, app, inviter, app,
	)
}

// Handlers serve the trusted-circle routes (R-03).
type Handlers struct {
	contacts *store.Queries
	notifier msg.Notifier
}

// NewHandlers wires the circle handlers: contacts store (sqlc, includes the
// inviter-name lookup) and the message notifier.
func NewHandlers(contacts *store.Queries, notifier msg.Notifier) *Handlers {
	return &Handlers{contacts: contacts, notifier: notifier}
}

// Register mounts the routes wrapped by the auth middleware (all protected).
func (h *Handlers) Register(mux *http.ServeMux, mw auth.Middleware) {
	mux.Handle("GET /v1/contacts", mw.Wrap(http.HandlerFunc(h.list)))
	mux.Handle("POST /v1/contacts", mw.Wrap(http.HandlerFunc(h.create)))
	mux.Handle("DELETE /v1/contacts/{id}", mw.Wrap(http.HandlerFunc(h.remove)))
}

type contactInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	contacts, err := h.contacts.ListContactsByUser(r.Context(), uuid.MustParse(userID))
	if err != nil {
		internal(w, r, err)
		return
	}
	list := make([]map[string]any, 0, len(contacts))
	for _, c := range contacts {
		list = append(list, contactJSON(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"contacts": list})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	var in contactInput
	if err := httpapi.DecodeJSON(r, &in); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed, err.Error())
		return
	}
	if n := len(in.Name); n < 1 || n > 80 {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
			"name must be 1..80 characters.")
		return
	}
	if !auth.ValidPhone(in.Phone) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
			"phone must be E.164, e.g. +5215598765432.")
		return
	}
	uid := uuid.MustParse(userID)

	next, err := h.contacts.NextContactPosition(r.Context(), uid)
	if err != nil {
		internal(w, r, err)
		return
	}
	if next > MaxContacts {
		httpapi.WriteProblem(w, r, http.StatusUnprocessableEntity, httpapi.CodeContactLimitReached,
			"You already have the maximum number of contacts.")
		return
	}

	created, err := h.contacts.CreateContact(r.Context(), store.CreateContactParams{
		ID:          newUUID(),
		UserID:      uid,
		DisplayName: in.Name,
		Phone:       in.Phone,
		Position:    int16(next),
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed,
				"This phone is already in your circle.")
			return
		}
		internal(w, r, err)
		return
	}

	// The welcome message goes out asynchronously: a slow provider never
	// blocks the API response, and delivery failures only log (the contact
	// exists; the welcome can be re-sent manually at this stage).
	inviter := "Alguien"
	if name, err := h.contacts.GetUserName(r.Context(), uid); err == nil {
		inviter = name
	}
	go h.sendWelcome(inviter, created)

	writeJSON(w, http.StatusCreated, contactJSON(created))
}

func (h *Handlers) sendWelcome(inviter string, c store.Contact) {
	text := welcomeMessageV1(inviter, c.DisplayName, AppName)
	if err := h.notifier.Send(context.Background(), c.Phone, text); err != nil {
		slog.Warn("welcome delivery failed", slog.String("contact_id", c.ID.String()))
	}
}

func (h *Handlers) remove(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpapi.WriteProblem(w, r, http.StatusNotFound, httpapi.CodeContactNotFound,
			"No contact with this id in your circle.")
		return
	}
	rows, err := h.contacts.DeleteContact(r.Context(), store.DeleteContactParams{ID: id, UserID: uuid.MustParse(userID)})
	if err != nil {
		internal(w, r, err)
		return
	}
	if rows == 0 {
		httpapi.WriteProblem(w, r, http.StatusNotFound, httpapi.CodeContactNotFound,
			"No contact with this id in your circle.")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// contactJSON maps the store row to the contract's Contact schema.
func contactJSON(c store.Contact) map[string]any {
	return map[string]any{
		"id":        c.ID,
		"name":      c.DisplayName,
		"phone":     c.Phone,
		"createdAt": c.CreatedAt.Time,
	}
}

// newUUID generates the app-side UUID v7 (Postgres 16 has no uuidv7).
func newUUID() uuid.UUID {
	id, err := uuid.NewV7()
	if err != nil {
		panic(fmt.Sprintf("circle: uuid v7: %v", err)) // crypto/rand exhaustion is not recoverable
	}
	return id
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func internal(w http.ResponseWriter, r *http.Request, _ error) {
	httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
}
