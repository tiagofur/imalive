package httpmw

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"time"
)

type requestIDKey struct{}

// RequestID ensures every request carries a unique identifier: generated with
// crypto/rand, set as the X-Request-ID response header (for clients to quote
// in support conversations) and injected into the request context for
// downstream handlers and logs. Incoming X-Request-ID headers are never
// trusted.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := NewRequestID()
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey{}, id)))
	})
}

// RequestIDFromContext returns the id injected by RequestID, or "" when the
// middleware is not in the chain.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey{}).(string)
	return id
}

// NewRequestID returns a random 128-bit identifier, hex-encoded.
func NewRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand never fails on Linux/macOS; fall back to a timestamp so
		// requests stay correlatable even then.
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
