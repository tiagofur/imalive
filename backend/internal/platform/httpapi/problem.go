// Package httpapi carries the HTTP plumbing shared by every handler: the
// RFC 9457 Problem writer with the contract's stable business codes and the
// JSON body decoding rules (unknown fields rejected, exact shapes).
package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Business codes of the contract (contracts/openapi.yaml Problem examples).
// The problem type URI derives from the code in kebab-case, e.g.
// PHONE_INVALID -> https://api.tranqui.app/problems/phone-invalid.
const (
	CodePhoneInvalid        = "PHONE_INVALID"
	CodeValidationFailed    = "VALIDATION_FAILED"
	CodeOtpInvalid          = "OTP_INVALID"
	CodeOtpExpired          = "OTP_EXPIRED"
	CodeUnauthorized        = "UNAUTHORIZED"
	CodeRateLimited         = "RATE_LIMITED"
	CodeContactLimitReached = "CONTACT_LIMIT_REACHED"
	CodeContactNotFound     = "CONTACT_NOT_FOUND"
	CodeInternal            = "INTERNAL"
)

type problem struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail,omitempty"`
	Code     string `json:"code"`
	Instance string `json:"instance,omitempty"`
}

// WriteProblem emits an RFC 9457 problem with the contract's stable fields:
// type (derived from the business code), title, status, detail, code and
// instance (the request path).
func WriteProblem(w http.ResponseWriter, r *http.Request, status int, code, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{
		Type:     "https://api.tranqui.app/problems/" + codeToSlug(code),
		Title:    titleFor(status),
		Status:   status,
		Detail:   detail,
		Code:     code,
		Instance: r.URL.Path,
	})
}

func codeToSlug(code string) string {
	parts := strings.Split(strings.ToLower(code), "_")
	return strings.Join(parts, "-")
}

func titleFor(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusForbidden:
		return "Forbidden"
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusConflict:
		return "Conflict"
	case http.StatusTooManyRequests:
		return "Too Many Requests"
	case http.StatusInternalServerError:
		return "Internal Server Error"
	default:
		return fmt.Sprintf("HTTP %d", status)
	}
}

// DecodeJSON decodes a JSON body strictly into dst: unknown fields are
// rejected and the payload must be a single JSON object.
func DecodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("invalid json body: %w", err)
	}
	return nil
}

// RetryAfter sets the contract's Retry-After header (seconds).
func RetryAfter(w http.ResponseWriter, seconds int) {
	w.Header().Set("Retry-After", fmt.Sprintf("%d", seconds))
}
