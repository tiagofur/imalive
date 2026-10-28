package httpmw

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

// contextWithRequestID injects an id as the middleware would.
func contextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey{}, id)
}

func TestRequestLogging(t *testing.T) {
	t.Run("logs one record with the canonical fields", func(t *testing.T) {
		var buf bytes.Buffer
		handler := RequestLogging(testLogger(&buf))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
		}))
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/v1/checkins", nil)
		req = req.WithContext(contextWithRequestID(req.Context(), "req-123"))

		rec := serve(handler, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("downstream status = %d, want %d", rec.Code, http.StatusCreated)
		}
		records := decodeLogLines(t, &buf)
		if len(records) != 1 {
			t.Fatalf("logged %d records, want exactly 1: %v", len(records), records)
		}
		record := records[0]
		want := map[string]any{
			"msg":        "request",
			"method":     http.MethodPost,
			"path":       "/v1/checkins",
			"status":     float64(http.StatusCreated),
			"request_id": "req-123",
		}
		for field, value := range want {
			if record[field] != value {
				t.Errorf("%s = %v, want %v", field, record[field], value)
			}
		}
		if _, ok := record["duration"].(float64); !ok {
			t.Errorf("duration = %v (%T), want a numeric duration", record["duration"], record["duration"])
		}
	})

	t.Run("logs the implicit 200 of handlers that skip WriteHeader", func(t *testing.T) {
		var buf bytes.Buffer
		handler := RequestLogging(testLogger(&buf))(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte("ok"))
		}))

		_ = serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil))

		records := decodeLogLines(t, &buf)
		if len(records) != 1 || records[0]["status"] != float64(http.StatusOK) {
			t.Fatalf("records = %v, want one record with status 200", records)
		}
	})

	t.Run("never logs the query string", func(t *testing.T) {
		var buf bytes.Buffer
		handler := RequestLogging(testLogger(&buf))(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

		_ = serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x?phone=%2B5215512345678", nil))

		records := decodeLogLines(t, &buf)
		if len(records) != 1 || records[0]["path"] != "/x" {
			t.Fatalf("records = %v, want one record with path %q only", records, "/x")
		}
		if bytes.Contains(buf.Bytes(), []byte("phone")) {
			t.Errorf("log output leaks query data: %s", buf.String())
		}
	})
}

// Integration through the full chain (docs/stack/backend-go.md): even a panic
// must produce exactly one request record, with the true status the client
// received, next to the panic recovered record — both with the request_id.
func TestChainRequestLogging(t *testing.T) {
	chain := func(log *slog.Logger, downstream http.Handler) http.Handler {
		return RequestID(RequestLogging(log)(Recovery(log)(downstream)))
	}

	t.Run("panicking handler logs one request record with the true 500 and request_id", func(t *testing.T) {
		var buf bytes.Buffer
		handler := chain(testLogger(&buf), http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("boom")
		}))

		rec := serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/v1/x", nil))

		if rec.Code != http.StatusInternalServerError {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
		}
		wantID := rec.Header().Get("X-Request-ID")
		if wantID == "" {
			t.Fatal("X-Request-ID header is empty, want the generated id")
		}
		var requestRecords []map[string]any
		for _, record := range decodeLogLines(t, &buf) {
			switch record["msg"] {
			case "request":
				requestRecords = append(requestRecords, record)
			case "panic recovered":
				if record["request_id"] != wantID {
					t.Errorf("panic recovered request_id = %v, want %q", record["request_id"], wantID)
				}
			default:
				t.Errorf("unexpected log record %v", record)
			}
		}
		if len(requestRecords) != 1 {
			t.Fatalf("logged %d request records, want exactly 1: %v", len(requestRecords), requestRecords)
		}
		if requestRecords[0]["status"] != float64(http.StatusInternalServerError) {
			t.Errorf("request status = %v, want 500, the status the client actually received", requestRecords[0]["status"])
		}
		if requestRecords[0]["request_id"] != wantID {
			t.Errorf("request request_id = %v, want %q", requestRecords[0]["request_id"], wantID)
		}
	})

	t.Run("healthy handler logs one request record with the real status", func(t *testing.T) {
		var buf bytes.Buffer
		handler := chain(testLogger(&buf), http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTeapot)
		}))

		rec := serve(handler, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/x", nil))

		if rec.Code != http.StatusTeapot {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusTeapot)
		}
		records := decodeLogLines(t, &buf)
		if len(records) != 1 || records[0]["msg"] != "request" {
			t.Fatalf("records = %v, want exactly one request record", records)
		}
		if records[0]["status"] != float64(http.StatusTeapot) {
			t.Errorf("status = %v, want %d", records[0]["status"], http.StatusTeapot)
		}
		if records[0]["request_id"] == "" {
			t.Error("request_id is empty, want the generated id")
		}
	})
}
