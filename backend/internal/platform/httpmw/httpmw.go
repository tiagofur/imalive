// Package httpmw provides the base middleware chain of the API
// (docs/stack/backend-go.md): recovery and request-id; the per-request log
// record joins in the next PR of the chain. Logs carry structured fields
// only — never headers, query strings or bodies (privacy by default).
package httpmw

import (
	"fmt"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery converts panics from downstream handlers into 500 responses and
// logs the panic with its stack, so a crashing handler never kills the
// process (fail-safe for the user).
func Recovery(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error("panic recovered",
						slog.String("panic", fmt.Sprint(rec)),
						slog.String("stack", string(debug.Stack())),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
