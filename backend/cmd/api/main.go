// Command api runs the imalive HTTP API: liveness at /healthz plus the
// product endpoints behind the base middleware chain (request-id, logging,
// recovery) and the auth middleware for protected routes. The auth routes
// (R-01, issue #39) arrive with the OpenAPI contract (issue #2).
package main

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/auth"
	authstore "github.com/tiagofur/imalive/backend/internal/auth/store"
	"github.com/tiagofur/imalive/backend/internal/checkin"
	checkinstore "github.com/tiagofur/imalive/backend/internal/checkin/store"
	"github.com/tiagofur/imalive/backend/internal/circle"
	circlestore "github.com/tiagofur/imalive/backend/internal/circle/store"
	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/platform/config"
	"github.com/tiagofur/imalive/backend/internal/platform/db"
	"github.com/tiagofur/imalive/backend/internal/platform/healthz"
	"github.com/tiagofur/imalive/backend/internal/platform/httpmw"
	"github.com/tiagofur/imalive/backend/internal/platform/httpserver"
	"github.com/tiagofur/imalive/backend/internal/platform/lifecycle"
	"github.com/tiagofur/imalive/backend/internal/platform/logging"
	"github.com/tiagofur/imalive/backend/internal/platform/msg"
	"github.com/tiagofur/imalive/backend/internal/platform/otp"
	"github.com/tiagofur/imalive/backend/internal/profile"
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

	// The auth routes are the first data consumers: DatabaseURL and RedisURL
	// become mandatory for this binary (the worker keeps its own policy).
	if cfg.DatabaseURL == "" {
		return fmt.Errorf("api: DATABASE_URL is required (auth persists users)")
	}
	if cfg.RedisURL == "" {
		return fmt.Errorf("api: REDIS_URL is required (otp and sessions live in Redis)")
	}

	// Shared with the worker (lifecycle): graceful cancel on the first
	// SIGINT/SIGTERM, force-exit if the operator sends a second one.
	ctx, stop := lifecycle.SignalContext()
	defer stop()

	pool, err := db.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	rdb := redis.NewClient(&redis.Options{Addr: redisAddr(cfg.RedisURL)})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("api: redis: %w", err)
	}

	sender, err := otp.NewFromConfig(cfg, otp.SenderConfig{
		TwilioAccountSid:       cfg.TwilioAccountSid,
		TwilioAuthToken:        cfg.TwilioAuthToken,
		TwilioVerifyServiceSid: cfg.TwilioVerifyServiceSid,
	})
	if err != nil {
		return err
	}
	// Messaging for user-facing texts (welcome, R-03): sandbox locally; the
	// Twilio Messaging side arrives with issue #6 credentials.
	var msgNotifier msg.Notifier = msg.NoopNotifier{}
	if cfg.OtpSender == "sandbox" {
		msgNotifier = msg.NewSandboxNotifier("")
	}

	authService := auth.NewService(otp.NewStore(rdb, cfg.OtpSecret), sender, rdb,
		authstore.New(pool), cfg.OtpSecret)
	authHandlers := auth.NewHandlers(authService, auth.NewRateLimiter(rdb))
	circleHandlers := circle.NewHandlers(circlestore.New(pool), msgNotifier)
	// The escalation engine (R-07) serves the life-signal and pause
	// transactions of the handlers below; the worker runs its tick loop.
	engine := escalation.New(pool, escalation.SystemClock{}, log)
	profileHandlers := profile.NewHandlers(pool, authstore.New(pool), engine)
	authMW := auth.NewMiddleware(authService)
	checkinHandlers := checkin.NewHandlers(pool, checkinstore.New(pool), profileHandlers, engine)

	var lc net.ListenConfig
	ln, err := lc.Listen(ctx, "tcp", ":"+cfg.Port)
	if err != nil {
		return fmt.Errorf("listen on :%s: %w", cfg.Port, err)
	}

	mux := http.NewServeMux()
	mux.Handle("GET /healthz", healthz.New(cfg.Version))
	authHandlers.Register(mux)
	circleHandlers.Register(mux, authMW)
	profileHandlers.Register(mux, authMW)
	checkinHandlers.Register(mux, authMW)
	// Chain order per docs/stack/backend-go.md, outermost first:
	// request-id -> request-logging -> recovery. Request-logging wraps
	// recovery so the per-request record carries the true 500 of a panic,
	// and request-id runs outside so every log carries it. The auth
	// middleware guards protected routes inside the handlers (per-route).
	handler := httpmw.RequestID(httpmw.RequestLogging(log)(httpmw.Recovery(log)(mux)))

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

// redisAddr extracts host:port from a redis:// URL for the client options
// (the platform keeps a single RedisURL config value).
func redisAddr(url string) string {
	const prefix = "redis://"
	if len(url) > len(prefix) && url[:len(prefix)] == prefix {
		return url[len(prefix):]
	}
	return url
}
