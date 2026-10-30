// Command worker runs the imalive escalation engine (R-07, ADR-0004): every
// tick it opens due daily cycles and claims/executes due ladder jobs, writing
// notifications to the transactional outbox. The outbox dispatcher (#57) is
// the one that talks to Twilio/FCM — this process never sends anything.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/platform/config"
	"github.com/tiagofur/imalive/backend/internal/platform/db"
	"github.com/tiagofur/imalive/backend/internal/platform/lifecycle"
	"github.com/tiagofur/imalive/backend/internal/platform/logging"
	"github.com/tiagofur/imalive/backend/internal/platform/msg"
	"github.com/tiagofur/imalive/backend/internal/platform/push"
)

// version is injected at build time: go build -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

// cycleInterval is the engine tick (ADR-0004); the job delay SLO (p95 < 60 s)
// assumes this granularity.
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

	if cfg.DatabaseURL == "" {
		return fmt.Errorf("worker: DATABASE_URL is required (the engine is its state)")
	}

	ctx, stop := lifecycle.SignalContext()
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	// The signer stamps the web alert links (R-08) into incident payloads;
	// the secret must match the api process (config, fail-fast).
	engine := escalation.New(pool, escalation.SystemClock{}, log).WithAlertSigner(
		escalation.NewAlertTokenSigner([]byte(cfg.AlertLinkSecret), cfg.PublicBaseURL,
			escalation.SystemClock{}))

	// The outbox dispatcher (issue #57): the same tick drains the outbox.
	// WhatsApp/SMS go through the sandbox until #6 delivers Twilio
	// credentials; push goes through the sandbox stub (FCM HTTP v1 lands
	// behind the same interface once credentials exist).
	var messages msg.Notifier = msg.NoopNotifier{}
	if cfg.OtpSender == "sandbox" {
		messages = msg.NewSandboxNotifier("")
	}
	dispatcher := escalation.NewDispatcher(pool, escalation.SystemClock{}, log,
		messages, push.NewSandboxSender())

	ticker := time.NewTicker(cycleInterval)
	defer ticker.Stop()

	log.Info("worker started",
		slog.String("env", string(cfg.Env)),
		slog.String("version", version),
		slog.Duration("cycle_interval", cycleInterval),
	)
	if err := lifecycle.Run(ctx, ticker.C, func(ctx context.Context) {
		engine.ProcessTick(ctx)
		dispatcher.DispatchTick(ctx)
	}); err != nil {
		return err
	}
	log.Info("worker stopped")
	return nil
}
