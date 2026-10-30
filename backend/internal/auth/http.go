package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"regexp"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/tiagofur/imalive/backend/internal/platform/httpapi"
)

// phonePattern is the E.164 shape the schema and the users table enforce.
var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// ValidPhone reports whether the phone is E.164.
func ValidPhone(phone string) bool { return phonePattern.MatchString(phone) }

// Handlers serve the three auth routes of the contract.
type Handlers struct {
	service *Service
	limiter *RateLimiter
}

// NewHandlers wires the handlers with the service and the rate limiter.
func NewHandlers(service *Service, limiter *RateLimiter) *Handlers {
	return &Handlers{service: service, limiter: limiter}
}

// Register mounts the routes on the mux with the exact contract paths.
func (h *Handlers) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/auth/otp/request", h.requestOtp)
	mux.HandleFunc("POST /v1/auth/otp/verify", h.verifyOtp)
	mux.HandleFunc("POST /v1/auth/refresh", h.refresh)
}

type otpRequest struct {
	Phone string `json:"phone"`
}

type otpVerify struct {
	Phone string `json:"phone"`
	Code  string `json:"code"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *Handlers) requestOtp(w http.ResponseWriter, r *http.Request) {
	var body otpRequest
	if err := httpapi.DecodeJSON(r, &body); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed, err.Error())
		return
	}
	if !ValidPhone(body.Phone) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodePhoneInvalid,
			"Phone must be E.164, e.g. +5215512345678.")
		return
	}
	if !h.limiter.Allow(r.Context(), phoneKey(body.Phone), clientIP(r)) {
		retryAfter(w, r, httpapi.CodeRateLimited)
		return
	}
	if err := h.service.Request(r.Context(), body.Phone); err != nil {
		internal(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) verifyOtp(w http.ResponseWriter, r *http.Request) {
	var body otpVerify
	if err := httpapi.DecodeJSON(r, &body); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed, err.Error())
		return
	}
	if !ValidPhone(body.Phone) {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodePhoneInvalid,
			"Phone must be E.164, e.g. +5215512345678.")
		return
	}
	if !h.limiter.Allow(r.Context(), phoneKey(body.Phone), clientIP(r)) {
		retryAfter(w, r, httpapi.CodeRateLimited)
		return
	}

	tokens, err := h.service.Verify(r.Context(), body.Phone, body.Code)
	if err != nil {
		switch {
		case errors.Is(err, ErrOtpInvalid):
			httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeOtpInvalid,
				"The code does not match.")
		case errors.Is(err, ErrOtpExpired):
			httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeOtpExpired,
				"The code expired or was already used; request a new one.")
		default:
			internal(w, r, err)
		}
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (h *Handlers) refresh(w http.ResponseWriter, r *http.Request) {
	var body refreshRequest
	if err := httpapi.DecodeJSON(r, &body); err != nil {
		httpapi.WriteProblem(w, r, http.StatusBadRequest, httpapi.CodeValidationFailed, err.Error())
		return
	}
	tokens, err := h.service.Refresh(r.Context(), body.RefreshToken)
	if err != nil {
		if errors.Is(err, ErrSessionRevoked) {
			httpapi.WriteProblem(w, r, http.StatusUnauthorized, httpapi.CodeUnauthorized,
				"Missing, expired or invalid credentials.")
			return
		}
		internal(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

// RateLimiter is a Redis fixed-window limiter for the auth routes: 5
// requests per phone per hour and 20 per client IP per hour (plan of
// issue #39). Exceeding yields 429 with Retry-After per the contract.
type RateLimiter struct {
	rdb        redis.Cmdable
	phoneLimit int
	ipLimit    int
	windowTTL  time.Duration
	retryAfter int
}

// NewRateLimiter builds the limiter with the plan's windows (5/phone/hour,
// 20/IP/hour, Retry-After 60s).
func NewRateLimiter(rdb redis.Cmdable) *RateLimiter {
	return &RateLimiter{
		rdb:        rdb,
		phoneLimit: 5,
		ipLimit:    20,
		windowTTL:  time.Hour,
		retryAfter: 60,
	}
}

// Allow consumes one slot for the phone and one for the IP; both must fit.
func (l *RateLimiter) Allow(ctx context.Context, phone, ip string) bool {
	return l.below(ctx, "rl:otp:phone:"+phone, l.phoneLimit) &&
		l.below(ctx, "rl:otp:ip:"+ip, l.ipLimit)
}

func (l *RateLimiter) below(ctx context.Context, key string, limit int) bool {
	pipe := l.rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.ExpireNX(ctx, key, l.windowTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		// Fail open on infrastructure errors: availability beats the limiter
		// (the OTP itself still has the single-attempt guarantee).
		return true
	}
	return incr.Val() <= int64(limit)
}

func phoneKey(phone string) string { return phone } // already E.164, unique per user

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func retryAfter(w http.ResponseWriter, r *http.Request, code string) {
	httpapi.RetryAfter(w, 60)
	httpapi.WriteProblem(w, r, http.StatusTooManyRequests, code, "Rate limit exceeded; wait 60 seconds.")
}

func internal(w http.ResponseWriter, r *http.Request, _ error) {
	// The error detail never reaches the client (it may carry internals).
	httpapi.WriteProblem(w, r, http.StatusInternalServerError, httpapi.CodeInternal, "Internal server error.")
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
