// Package lifecycle holds the process lifecycle helpers shared by the api
// and worker binaries: wiring shutdown signals into a context and the tick
// loop the escalation engine will grow into (ADR-0004).
package lifecycle

import (
	"context"
	"os/signal"
	"syscall"
	"time"
)

// SignalContext returns a context cancelled on SIGINT/SIGTERM, so every
// long-running loop exits through the same graceful path.
func SignalContext() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
}

// Run executes fn for every tick received from ticks until ctx is cancelled,
// then returns nil. fn runs synchronously on the caller goroutine: cycles
// never overlap by construction. The production ticker (time.NewTicker) or a
// test channel can feed ticks, which keeps the loop deterministic to test.
func Run(ctx context.Context, ticks <-chan time.Time, fn func(ctx context.Context)) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticks:
			if fn != nil {
				fn(ctx)
			}
		}
	}
}
