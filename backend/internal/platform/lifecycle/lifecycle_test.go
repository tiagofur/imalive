package lifecycle

import (
	"context"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	t.Run("runs one cycle per tick and exits cleanly on cancellation", func(t *testing.T) {
		ticks := make(chan time.Time)
		ctx, cancel := context.WithCancel(context.Background())
		cycled := make(chan int, 8) // cycle counter snapshot after each fn
		done := make(chan error, 1)

		go func() {
			done <- Run(ctx, ticks, func(context.Context) {
				select {
				case cycled <- 1:
				default:
				}
			})
		}()

		for range 3 {
			ticks <- time.Now() // unbuffered: blocks until Run receives
			<-cycled            // blocks until fn finished
		}

		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Run() = %v, want nil", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Run() did not return after cancellation")
		}
	})

	t.Run("returns promptly when the context starts cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		var ran bool

		err := Run(ctx, make(chan time.Time), func(context.Context) { ran = true })
		if err != nil {
			t.Fatalf("Run() = %v, want nil", err)
		}
		if ran {
			t.Fatal("a cycle ran although the context was already cancelled")
		}
	})

	t.Run("consumes ticks without a callback", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		ticks := make(chan time.Time)

		done := make(chan error, 1)
		go func() { done <- Run(ctx, ticks, nil) }()
		ticks <- time.Now()
		cancel()

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Run() = %v, want nil", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Run() did not return")
		}
	})

	t.Run("returns promptly when the ticks channel is closed", func(t *testing.T) {
		// A closed channel is always ready to receive: without the guard the
		// loop would spin at 100% CPU (JD ledger JD-4).
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ticks := make(chan time.Time)
		close(ticks)

		done := make(chan error, 1)
		go func() { done <- Run(ctx, ticks, nil) }()

		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Run() = %v, want nil", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Run() did not return on a closed ticks channel")
		}
	})
}
