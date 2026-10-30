// Package config loads the typed process configuration from the environment
// and validates it fail-fast at startup (docs/stack/backend-go.md): the
// process refuses to run on invalid input instead of silently defaulting.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Environment is the deployment environment (docs/arquitectura/overview.md).
type Environment string

const (
	// EnvLocal is a developer machine: docker-compose, sandbox credentials.
	EnvLocal Environment = "local"
	// EnvStaging mirrors production for the closed beta.
	EnvStaging Environment = "staging"
	// EnvProduction serves real users with minimal data.
	EnvProduction Environment = "production"
)

// Defaults for the optional variables.
const (
	DefaultPort     = "8080"
	DefaultLogLevel = "info"
)

// Config is the validated process configuration.
type Config struct {
	// Env selects environment-specific behavior.
	Env Environment
	// Port is the TCP port the HTTP API listens on.
	Port string
	// LogLevel is the minimum severity of the slog JSON logger.
	LogLevel slog.Level
	// Version is the build version, injected via -ldflags "-X main.version=..."
	// (never read from the environment).
	Version string
	// DatabaseURL points at PostgreSQL (ADR-0004). Declared but not yet
	// required: it becomes mandatory when the first data consumer lands.
	DatabaseURL string
	// RedisURL points at Redis (cache/ephemeral). Same policy as DatabaseURL.
	RedisURL string
	// OtpSender selects how one-time codes are delivered: "noop" (default,
	// accepts everything), "sandbox" (writes to a local outbox file) or
	// "twilio" (Verify API, credentials via TWILIO_* env, issue #6).
	OtpSender string
	// Twilio credentials for OtpSender=twilio (Verify API v2). Required only
	// when the twilio sender is selected (fail-fast in the sender factory).
	TwilioAccountSid       string
	TwilioAuthToken        string
	TwilioVerifyServiceSid string
	// OtpSecret keys the HMAC that hashes OTPs in Redis. Required whenever
	// OtpSender is not "noop": without it, issued codes would be worthless.
	OtpSecret string
	// AlertLinkSecret keys the HMAC of the web alert links "Voy yo / No
	// puedo" (R-08). Required in staging/production (a forgeable link would
	// let anyone answer someone else's alert); in local an ephemeral random
	// secret is generated per process.
	AlertLinkSecret string
	// PublicBaseURL is the origin the alert links point at (R-08). Empty in
	// local dev: the payloads carry a relative path.
	PublicBaseURL string
}

// Load builds the configuration from the process environment.
func Load(version string) (Config, error) {
	return FromEnv(os.Getenv, version)
}

// FromEnv builds and validates the configuration. getenv is injectable for
// tests; version comes from build flags.
func FromEnv(getenv func(string) string, version string) (Config, error) {
	cfg := Config{Version: version}

	env, err := required(getenv, "APP_ENV")
	if err != nil {
		return Config{}, err
	}
	switch e := Environment(env); e {
	case EnvLocal, EnvStaging, EnvProduction:
		cfg.Env = e
	default:
		return Config{}, fmt.Errorf("config: APP_ENV %q is not one of %q, %q or %q", env, EnvLocal, EnvStaging, EnvProduction)
	}

	port := optional(getenv, "PORT", DefaultPort)
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("config: PORT %q is not a valid TCP port", port)
	}
	cfg.Port = port

	level, err := parseLevel(optional(getenv, "LOG_LEVEL", DefaultLogLevel))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	cfg.DatabaseURL = strings.TrimSpace(getenv("DATABASE_URL"))
	cfg.RedisURL = strings.TrimSpace(getenv("REDIS_URL"))

	sender := optional(getenv, "OTP_SENDER", "noop")
	switch sender {
	case "noop", "sandbox", "twilio":
		cfg.OtpSender = sender
	default:
		return Config{}, fmt.Errorf("config: OTP_SENDER %q is not one of noop, sandbox or twilio", sender)
	}
	secret := strings.TrimSpace(getenv("OTP_SECRET"))
	if sender != "noop" && secret == "" {
		return Config{}, fmt.Errorf("config: OTP_SECRET is required when OTP_SENDER is %q (codes are stored hashed)", sender)
	}
	cfg.OtpSecret = secret

	cfg.TwilioAccountSid = strings.TrimSpace(getenv("TWILIO_ACCOUNT_SID"))
	cfg.TwilioAuthToken = strings.TrimSpace(getenv("TWILIO_AUTH_TOKEN"))
	cfg.TwilioVerifyServiceSid = strings.TrimSpace(getenv("TWILIO_VERIFY_SERVICE_SID"))

	cfg.PublicBaseURL = strings.TrimRight(strings.TrimSpace(getenv("PUBLIC_BASE_URL")), "/")
	alertSecret := strings.TrimSpace(getenv("ALERT_LINK_SECRET"))
	if alertSecret == "" {
		if cfg.Env != EnvLocal {
			return Config{}, fmt.Errorf("config: ALERT_LINK_SECRET is required in %q (the alert links are signed capabilities, R-08)", cfg.Env)
		}
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			return Config{}, fmt.Errorf("config: ephemeral alert secret: %w", err)
		}
		alertSecret = base64.RawURLEncoding.EncodeToString(buf)
	}
	cfg.AlertLinkSecret = alertSecret

	return cfg, nil
}

// parseLevel accepts exactly the documented enum (debug, info, warn, error).
// It deliberately avoids slog.Level.UnmarshalText: that accepts offset forms
// like "INFO+2" whose extremes (ERROR+50) would silently drop every record,
// including errors (fail-fast contract: no silently surprising levels).
func parseLevel(s string) (slog.Level, error) {
	switch s {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("config: LOG_LEVEL %q is not one of debug, info, warn or error", s)
	}
}

func required(getenv func(string) string, key string) (string, error) {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("config: %s is required", key)
}

func optional(getenv func(string) string, key, fallback string) string {
	if v := strings.TrimSpace(getenv(key)); v != "" {
		return v
	}
	return fallback
}
