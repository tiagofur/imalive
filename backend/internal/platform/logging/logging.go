// Package logging sets up the process logger: slog with a JSON handler on
// stdout (docs/stack/backend-go.md). Records carry structured fields only —
// never PII, request headers or bodies (privacy by default).
package logging

import (
	"io"
	"log/slog"
	"os"
)

// New returns a slog logger writing JSON records at the given level to w
// (os.Stdout when w is nil).
func New(w io.Writer, level slog.Level) *slog.Logger {
	if w == nil {
		w = os.Stdout
	}
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}
