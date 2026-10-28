// Package httpserver builds the API HTTP server with production timeouts and
// graceful shutdown (docs/stack/backend-go.md).
package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Timeouts protect the process from slow clients and leaked connections.
const (
	ReadTimeout     = 5 * time.Second
	WriteTimeout    = 10 * time.Second
	IdleTimeout     = 60 * time.Second
	ShutdownTimeout = 10 * time.Second
)

// New returns an http.Server with production timeouts and the given handler.
func New(handler http.Handler) *http.Server {
	return &http.Server{
		Handler:      handler,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
		IdleTimeout:  IdleTimeout,
	}
}

// Serve accepts connections on ln until ctx is cancelled (e.g. by a SIGTERM
// forwarded as context cancellation), then shuts down gracefully, waiting up
// to ShutdownTimeout for in-flight requests to finish.
func Serve(ctx context.Context, ln net.Listener, srv *http.Server, log *slog.Logger) error {
	serveErr := make(chan error, 1)
	go func() { serveErr <- srv.Serve(ln) }()

	select {
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("httpserver: serve: %w", err)
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()
		log.Info("http server shutting down")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("httpserver: shutdown: %w", err)
		}
		return nil
	}
}
