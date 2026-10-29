package httpmw

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRecovery(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.Handler
		wantStatus int
		wantPanic  bool
	}{
		{
			name: "panicking handler becomes a 500 with the panic logged",
			handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				panic("boom")
			}),
			wantStatus: http.StatusInternalServerError,
			wantPanic:  true,
		},
		{
			name: "healthy handler passes through untouched",
			handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusTeapot)
			}),
			wantStatus: http.StatusTeapot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)

			rec := serve(Recovery(testLogger(&buf))(tt.handler), req)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			records := decodeLogLines(t, &buf)
			if !tt.wantPanic {
				if len(records) != 0 {
					t.Fatalf("healthy handler logged %d records, want 0", len(records))
				}
				return
			}
			if len(records) != 1 || records[0]["msg"] != "panic recovered" {
				t.Fatalf("records = %v, want one %q record", records, "panic recovered")
			}
			if records[0]["panic"] != "boom" {
				t.Errorf("panic field = %v, want %q", records[0]["panic"], "boom")
			}
			if records[0]["stack"] == "" {
				t.Error("stack field is empty, want the panic stack")
			}
		})
	}
}

// Recovery must keep the process alive: panics are absorbed, never re-raised.
func TestRecoveryAbsorbsRepeatedPanics(t *testing.T) {
	var buf bytes.Buffer
	handler := Recovery(testLogger(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("always")
	}))

	for range 3 {
		if rec := serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil)); rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want 500", rec.Code)
		}
	}
	if got := len(decodeLogLines(t, &buf)); got != 3 {
		t.Fatalf("logged %d panics, want 3", got)
	}
}

// http.ErrAbortHandler is the net/http convention for a deliberate response
// abort, not a bug: Recovery re-raises it for the server to handle silently
// instead of logging it as a crash or writing over the aborted response
// (JD ledger JD-8).
func TestRecoveryReraisesErrAbortHandler(t *testing.T) {
	var buf bytes.Buffer
	handler := Recovery(testLogger(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		rec := recover()
		if rec == nil {
			t.Fatal("Recovery absorbed http.ErrAbortHandler, want it re-raised")
		}
		if rec != http.ErrAbortHandler {
			t.Fatalf("recovered %v, want http.ErrAbortHandler", rec)
		}
		if records := decodeLogLines(t, &buf); len(records) != 0 {
			t.Fatalf("logged %v, want no panic record for a deliberate abort", records)
		}
	}()
	_ = serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil))
}

// The chain runs request-id outside recovery (docs/stack/backend-go.md): the
// panic record must carry the same id the client sees in X-Request-ID.
func TestChainPanicRecordCarriesRequestID(t *testing.T) {
	var buf bytes.Buffer
	handler := RequestID(Recovery(testLogger(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	})))

	rec := serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	wantID := rec.Header().Get("X-Request-ID")
	if wantID == "" {
		t.Fatal("X-Request-ID header is empty, want the generated id")
	}
	records := decodeLogLines(t, &buf)
	if len(records) != 1 || records[0]["msg"] != "panic recovered" {
		t.Fatalf("records = %v, want one %q record", records, "panic recovered")
	}
	if records[0]["request_id"] != wantID {
		t.Errorf("request_id = %v, want %q", records[0]["request_id"], wantID)
	}
}

// Chain control: a request that never panics logs no recovery record.
func TestChainHealthyHandlerLogsNoRecoveryRecord(t *testing.T) {
	var buf bytes.Buffer
	handler := RequestID(Recovery(testLogger(&buf))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})))

	rec := serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil))

	if rec.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
	}
	if records := decodeLogLines(t, &buf); len(records) != 0 {
		t.Fatalf("healthy handler logged %v, want no records", records)
	}
}
