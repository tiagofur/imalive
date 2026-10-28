package healthz

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("answers 200 with ok status and the build version", func(t *testing.T) {
		rec := httptest.NewRecorder()

		New("v0.2.0").ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/healthz", nil))

		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
		}
		var body struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("body is not valid JSON: %v (%q)", err, rec.Body.String())
		}
		if body.Status != "ok" || body.Version != "v0.2.0" {
			t.Fatalf("body = %+v, want status %q and version %q", body, "ok", "v0.2.0")
		}
	})

	t.Run("is registered for GET only, other methods get 405", func(t *testing.T) {
		mux := http.NewServeMux()
		mux.Handle("GET /healthz", New("dev"))

		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/healthz", nil))

		if rec.Code != http.StatusMethodNotAllowed {
			t.Fatalf("POST /healthz status = %d, want %d", rec.Code, http.StatusMethodNotAllowed)
		}
	})
}
