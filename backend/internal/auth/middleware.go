package auth

import (
	"context"
	"net/http"
	"strings"
)

type userIDKey struct{}

// Middleware guards protected routes with the bearer access token: it
// validates the JWT locally (signature + issuer + expiry, no roundtrip) and
// injects the user id into the request context. It mounts after
// request-id -> logging -> recovery (docs/stack/backend-go.md chain order).
type Middleware struct {
	service *Service
}

// NewMiddleware builds the bearer middleware for the service.
func NewMiddleware(service *Service) Middleware {
	return Middleware{service: service}
}

// Wrap guards next with the bearer check; a missing or invalid token yields a
// 401 Problem response (RFC 9457) consistent with the contract's errors.
func (m Middleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id, err := m.Validate(r.Header.Get("Authorization"))
		if err != nil {
			w.Header().Set("Content-Type", "application/problem+json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"type":"about:blank","title":"Unauthorized","status":401}`))
			return
		}
		next.ServeHTTP(w, r.WithContext(WithUserID(r.Context(), id)))
	})
}

// Validate returns the authenticated user id for an Authorization header, or
// an error when the token is missing or invalid.
func (m Middleware) Validate(authorization string) (string, error) {
	token, ok := bearerToken(authorization)
	if !ok {
		return "", ErrSessionRevoked
	}
	return m.service.UserID(token)
}

func bearerToken(header string) (string, bool) {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return "", false
	}
	return strings.TrimSpace(header[len(prefix):]), true
}

// WithUserID stores the authenticated user id in a context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, userIDKey{}, userID)
}

// UserIDFromContext extracts the authenticated user id (empty when absent).
func UserIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(userIDKey{}).(string)
	return id
}
