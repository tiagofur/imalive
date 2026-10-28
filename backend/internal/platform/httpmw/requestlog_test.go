package httpmw

import (
	"bytes"
	"context"
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
