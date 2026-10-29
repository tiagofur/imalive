package lifecycle

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"runtime"
	"syscall"
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

// The signal tests are unix-only: they deliver real signals to the process.
func TestSignalContextCancelsOnFirstSignal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal delivery test requires a unix process")
	}
	ctx, stop := SignalContext()
	defer stop()

	if err := syscall.Kill(os.Getpid(), syscall.SIGTERM); err != nil {
		t.Fatalf("could not deliver SIGTERM to self: %v", err)
	}
	select {
	case <-ctx.Done():
	case <-time.After(2 * time.Second):
		t.Fatal("context was not cancelled by the first SIGTERM")
	}
}

// TestSecondSignalForceExits runs itself as a subprocess: a hung shutdown
// must be interruptible with a second signal (JD ledger JD-6). The child
// cancels on the first signal, then sleeps to simulate a hung shutdown; the
// parent sends a second signal and expects exit code 1 well before the sleep
// ends. A clean exit would mean the force-exit did not fire.
func TestSecondSignalForceExits(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("signal delivery test requires a unix process")
	}
	if os.Getenv("LIFECYCLE_SIGNAL_TEST_CHILD") == "1" {
		ctx, stop := SignalContext()
		defer stop()
		<-ctx.Done()
		time.Sleep(2 * time.Second) // simulate the hung shutdown window
		return
	}

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=TestSecondSignalForceExits", "-test.count=1")
	cmd.Env = append(os.Environ(), "LIFECYCLE_SIGNAL_TEST_CHILD=1")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}

	time.Sleep(300 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("first signal: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatalf("second signal: %v", err)
	}

	waited := make(chan error, 1)
	go func() { waited <- cmd.Wait() }()
	select {
	case err := <-waited:
		var ee *exec.ExitError
		if !errors.As(err, &ee) || ee.ExitCode() != 1 {
			t.Fatalf("child exit = %v, want exit code 1 from the force exit", err)
		}
	case <-time.After(5 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("child did not force-exit after the second signal")
	}
}
