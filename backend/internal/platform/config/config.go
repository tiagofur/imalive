// Package config loads the typed process configuration from the environment
// and validates it fail-fast at startup (docs/stack/backend-go.md): the
// process refuses to run on invalid input instead of silently defaulting.
package config

import (
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

	return cfg, nil
}

func parseLevel(s string) (slog.Level, error) {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo, fmt.Errorf("config: LOG_LEVEL %q is not one of debug, info, warn or error", s)
	}
	return lvl, nil
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
