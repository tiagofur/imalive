package httpmw

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLogging writes one log record per request with the canonical fields:
// method, path, status, duration and request_id. Only the path is logged —
// never the query string, headers or body (privacy by default).
func RequestLogging(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			// Emitted in a defer: one record per request means also when a
			// panic unwinds from downstream (JD-1). With recovery running
			// inside, rec.status is the true status the client received.
			defer func() {
				log.Info("request",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", rec.status),
					slog.Duration("duration", time.Since(start)),
					slog.String("request_id", RequestIDFromContext(r.Context())),
				)
			}()
			next.ServeHTTP(rec, r)
		})
	}
}

// statusRecorder captures the status code the handler wrote. Handlers that
// write body bytes without an explicit WriteHeader keep the implicit 200.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(status int) {
	if r.wroteHeader {
		return
	}
	r.status = status
	r.wroteHeader = true
	r.ResponseWriter.WriteHeader(status)
}
