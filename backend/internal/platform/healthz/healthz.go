// Package healthz provides the liveness endpoint of the API: the process is
// up and reports its build version. It is an operational endpoint outside
// /v1, therefore outside the OpenAPI contract (ADR-0003); readiness checks
// against Postgres/Redis arrive with the first data PR.
package healthz

import (
	"encoding/json"
	"net/http"
)

// New returns the GET /healthz handler answering 200 with the build version.
func New(version string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Status  string `json:"status"`
			Version string `json:"version"`
		}{Status: "ok", Version: version})
	})
}
