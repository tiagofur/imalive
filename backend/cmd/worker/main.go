// Command worker runs the imalive escalation engine skeleton: a placeholder
// tick loop that only logs cycle heartbeats. No state, no jobs: the real
// engine (daily cycle + jobs + outbox on PostgreSQL, ADR-0004) lands with the
// escalation issue of the MVP.
package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/tiagofur/imalive/backend/internal/platform/config"
	"github.com/tiagofur/imalive/backend/internal/platform/lifecycle"
	"github.com/tiagofur/imalive/backend/internal/platform/logging"
)

// version is injected at build time: go build -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

// cycleInterval is the placeholder tick of the engine loop.
const cycleInterval = 30 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("worker exited with error", slog.String("error", err.Error()))
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

	ctx, stop := lifecycle.SignalContext()
	defer stop()

	ticker := time.NewTicker(cycleInterval)
	defer ticker.Stop()

	log.Info("worker started",
		slog.String("env", string(cfg.Env)),
		slog.String("version", version),
		slog.Duration("cycle_interval", cycleInterval),
	)
	if err := lifecycle.Run(ctx, ticker.C, func(context.Context) {
		log.Info("cycle heartbeat", slog.Duration("interval", cycleInterval))
	}); err != nil {
		return err
	}
	log.Info("worker stopped")
	return nil
}
