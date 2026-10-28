package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"
)

// client dials fresh on every request (no keep-alive) so that checks after
// shutdown are deterministic, and fails fast instead of hanging.
var client = &http.Client{
	Timeout:   2 * time.Second,
	Transport: &http.Transport{DisableKeepAlives: true},
}

func TestNew(t *testing.T) {
	srv := New(http.NewServeMux())

	if srv.ReadTimeout != ReadTimeout || srv.WriteTimeout != WriteTimeout || srv.IdleTimeout != IdleTimeout {
		t.Fatalf("New() timeouts = %v/%v/%v, want %v/%v/%v",
			srv.ReadTimeout, srv.WriteTimeout, srv.IdleTimeout, ReadTimeout, WriteTimeout, IdleTimeout)
	}
}

func TestServe(t *testing.T) {
	t.Run("serves requests and stops cleanly on context cancellation", func(t *testing.T) {
		ln := listener(t)
		srv := New(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = io.WriteString(w, "ok")
		}))

		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- Serve(ctx, ln, srv, slog.Default()) }()

		waitForServer(t, ln.Addr().String())
		resp, err := get(t, "http://"+ln.Addr().String()+"/")
		if err != nil {
			t.Fatalf("GET while serving: %v", err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK || string(body) != "ok" {
			t.Fatalf("GET = %d %q, want 200 %q", resp.StatusCode, body, "ok")
		}

		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("Serve() after cancellation = %v, want nil", err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Serve() did not return after context cancellation")
		}

		if _, err := get(t, "http://"+ln.Addr().String()+"/"); err == nil {
			t.Fatal("GET after shutdown succeeded, want connection refused")
		}
	})

	t.Run("returns the serve error of a broken listener", func(t *testing.T) {
		ln := listener(t)
		if err := ln.Close(); err != nil {
			t.Fatalf("close listener: %v", err)
		}

		err := Serve(context.Background(), ln, New(http.NewServeMux()), slog.Default())
		if err == nil {
			t.Fatal("Serve() with a closed listener = nil, want error")
		}
	})
}

func listener(t *testing.T) net.Listener {
	t.Helper()
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return ln
}

func get(t *testing.T, url string) (*http.Response, error) {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	return client.Do(req)
}

// waitForServer retries until the listener accepts connections, avoiding a
// race between Serve starting and the first request.
func waitForServer(t *testing.T, addr string) {
	t.Helper()
	for range 100 {
		if resp, err := get(t, "http://"+addr+"/"); err == nil {
			_ = resp.Body.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server never became reachable")
}
