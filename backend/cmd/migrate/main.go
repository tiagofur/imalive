// Command migrate applies the embedded SQL migrations against DATABASE_URL.
// It is the single migration path for dev (make migrate-*), CI and the
// integration tests: the SQL ships inside the binary via the migrations
// package (no external CLI).
//
// Usage:
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down
//	go run ./cmd/migrate version
package main

import (
	"fmt"
	"os"

	"github.com/tiagofur/imalive/backend/internal/platform/config"
	"github.com/tiagofur/imalive/backend/internal/platform/db"
)

func main() {
	if len(os.Args) != 2 {
		usage()
	}
	cfg, err := config.Load("dev")
	if err != nil {
		fatal(err)
	}
	if cfg.DatabaseURL == "" {
		fatal(fmt.Errorf("migrate: DATABASE_URL is required (set it in .env; make migrate-* sources it)"))
	}

	switch os.Args[1] {
	case "up":
		if err := db.Up(cfg.DatabaseURL); err != nil {
			fatal(err)
		}
	case "down":
		if err := db.DownOne(cfg.DatabaseURL); err != nil {
			fatal(err)
		}
	case "version":
		v, dirty, err := db.Version(cfg.DatabaseURL)
		if err != nil {
			fatal(err)
		}
		fmt.Printf("schema version %d (dirty=%v)\n", v, dirty)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: migrate up|down|version")
	os.Exit(2)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
