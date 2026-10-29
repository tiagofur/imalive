// Package lifecycle holds the process lifecycle helpers shared by the api
// and worker binaries: wiring shutdown signals into a context and the tick
// loop the escalation engine will grow into (ADR-0004).
package lifecycle

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// SignalContext returns a context cancelled on the first SIGINT/SIGTERM, so
// every long-running loop exits through the same graceful path. A second
// signal force-exits the process with status 1: an operator can always cut a
// hung shutdown short without reaching for SIGKILL (JD ledger JD-6). The
// returned stop unregisters the handler and tears the helper down, so a
// clean exit can never force-exit afterwards.
func SignalContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-ch: // first signal: take the graceful path
			cancel()
			select {
			case <-ch: // second signal: the operator insists, cut the shutdown
				os.Exit(1)
			case <-done:
				return
			}
		case <-done:
			return
		}
	}()
	stop := func() {
		signal.Stop(ch)
		close(done)
		cancel()
	}
	return ctx, stop
}

// Run executes fn for every tick received from ticks until ctx is cancelled
// or the ticks channel is closed, then returns nil. fn runs synchronously on
// the caller goroutine: cycles never overlap by construction. The production
// ticker (time.NewTicker) or a test channel can feed ticks, which keeps the
// loop deterministic to test. A closed channel ends the loop: without this
// guard the receive on a closed channel is always ready and the loop would
// spin at 100% CPU (JD ledger JD-4).
func Run(ctx context.Context, ticks <-chan time.Time, fn func(ctx context.Context)) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case _, ok := <-ticks:
			if !ok {
				return nil
			}
			if fn != nil {
				fn(ctx)
			}
		}
	}
}
