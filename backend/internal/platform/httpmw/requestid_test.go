package httpmw

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestID(t *testing.T) {
	t.Run("generates a distinct id per request and exposes it in the header and context", func(t *testing.T) {
		seen := map[string]string{}
		handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			seen[w.Header().Get("X-Request-ID")] = RequestIDFromContext(r.Context())
		}))

		for range 2 {
			_ = serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil))
		}

		if len(seen) != 2 {
			t.Fatalf("got ids %v, want 2 distinct generated ids", seen)
		}
		for id, ctxID := range seen {
			if id == "" {
				t.Error("X-Request-ID header is empty")
			}
			if ctxID != id {
				t.Errorf("context id %q does not match header id %q", ctxID, id)
			}
		}
	})

	t.Run("never trusts an incoming X-Request-ID header", func(t *testing.T) {
		var ctxID string
		handler := RequestID(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
			ctxID = RequestIDFromContext(r.Context())
		}))
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)
		req.Header.Set("X-Request-ID", "attacker-controlled")

		_ = serve(handler, req)

		if ctxID == "attacker-controlled" {
			t.Fatal("incoming X-Request-ID was echoed, want a freshly generated id")
		}
	})
}

func TestRequestIDFromContext(t *testing.T) {
	t.Run("returns empty outside the middleware chain", func(t *testing.T) {
		if got := RequestIDFromContext(context.Background()); got != "" {
			t.Fatalf("RequestIDFromContext(background) = %q, want empty", got)
		}
	})
}
