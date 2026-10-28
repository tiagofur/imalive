// Command api runs the imalive HTTP API: liveness at /healthz behind the
// base middleware chain (request-id, recovery). Product endpoints arrive
// with the OpenAPI contract (issue #2).
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/tiagofur/imalive/backend/internal/platform/config"
	"github.com/tiagofur/imalive/backend/internal/platform/healthz"
	"github.com/tiagofur/imalive/backend/internal/platform/httpmw"
	"github.com/tiagofur/imalive/backend/internal/platform/httpserver"
	"github.com/tiagofur/imalive/backend/internal/platform/logging"
)

// version is injected at build time: go build -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("api exited with error", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(version)
	if err != nil {
		// config errors already carry their "config:" context.
		return err
	}
	log := logging.New(os.Stdout, cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("listen on :%s: %w", cfg.Port, err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.New(cfg.Version))
	// Chain order per docs/stack/backend-go.md, outermost first:
	// request-id -> recovery (request logging slots between them in the next
	// PR of the chain). Request-id runs outside so panic logs carry it.
	handler := httpmw.RequestID(httpmw.Recovery(log)(mux))

	log.Info("api listening",
		slog.String("addr", ln.Addr().String()),
		slog.String("env", string(cfg.Env)),
		slog.String("version", version),
	)
	if err := httpserver.Serve(ctx, ln, httpserver.New(handler), log); err != nil {
		return fmt.Errorf("serve: %w", err)
	}
	log.Info("api stopped")
	return nil
}
