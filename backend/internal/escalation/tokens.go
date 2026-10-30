package escalation

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// The alert web link (R-08) is a stateless capability: HMAC over
// "incident:contact:exp", base64url. It works without app or login, grants
// access to exactly one incident as exactly one contact, and the incident
// state machine — not token storage — makes every action idempotent (the
// "one use" of the spec). Expired links still render the current state but
// cannot act.
const (
	alertTokenTTL    = 7 * 24 * time.Hour
	alertTokenSecret = "escalation:alert-token:v1"
)

// AlertTokenSigner issues and verifies alert links. The engine signs the
// links it writes into outbox payloads; the alert web verifies what arrives.
type AlertTokenSigner struct {
	secret  []byte
	baseURL string
	clock   Clock
}

// NewAlertTokenSigner wires the signer with the shared secret (config
// ALERT_LINK_SECRET) and the public base URL used to build absolute links.
func NewAlertTokenSigner(secret []byte, baseURL string, clock Clock) *AlertTokenSigner {
	return &AlertTokenSigner{secret: secret, baseURL: baseURL, clock: clock}
}

// BaseURL returns the configured public origin ("" in local dev: the link
// travels as a path the dispatcher completes).
func (s *AlertTokenSigner) BaseURL() string { return s.baseURL }

// Sign returns the alert token for one contact over one incident.
func (s *AlertTokenSigner) Sign(incidentID, contactID uuid.UUID) (string, error) {
	exp := s.clock.Now().Add(alertTokenTTL).Unix()
	payload := strings.Join([]string{
		incidentID.String(), contactID.String(),
		strconv.FormatInt(exp, 10),
	}, ":")
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(alertTokenSecret + ":" + payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." +
		base64.RawURLEncoding.EncodeToString(sig), nil
}

// VerifyAlertToken checks the signature and expiry of an alert token and
// returns its incident, contact and expiry.
func VerifyAlertToken(secret []byte, token string, now time.Time) (incidentID, contactID uuid.UUID, exp time.Time, err error) {
	payloadB64, sigB64, found := strings.Cut(token, ".")
	if !found {
		return uuid.Nil, uuid.Nil, time.Time{}, errMalformedToken
	}
	payload, perr := base64.RawURLEncoding.DecodeString(payloadB64)
	if perr != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, fmt.Errorf("%w: payload: %v", errMalformedToken, perr)
	}
	sig, serr := base64.RawURLEncoding.DecodeString(sigB64)
	if serr != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, fmt.Errorf("%w: signature: %v", errMalformedToken, serr)
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(alertTokenSecret + ":" + string(payload)))
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return uuid.Nil, uuid.Nil, time.Time{}, errBadTokenSignature
	}
	parts := strings.Split(string(payload), ":")
	if len(parts) != 3 {
		return uuid.Nil, uuid.Nil, time.Time{}, errMalformedToken
	}
	incidentID, err = uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, fmt.Errorf("%w: incident: %v", errMalformedToken, err)
	}
	contactID, err = uuid.Parse(parts[1])
	if err != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, fmt.Errorf("%w: contact: %v", errMalformedToken, err)
	}
	expUnix, cerr := strconv.ParseInt(parts[2], 10, 64)
	if cerr != nil {
		return uuid.Nil, uuid.Nil, time.Time{}, fmt.Errorf("%w: expiry: %v", errMalformedToken, err)
	}
	exp = time.Unix(expUnix, 0).UTC()
	if now.After(exp) {
		return uuid.Nil, uuid.Nil, exp, errExpiredToken
	}
	return incidentID, contactID, exp, nil
}

// Token verification failures map to clear web responses: an expired link is
// not an attacker, a bad signature is.
var (
	errMalformedToken    = errors.New("alert: malformed token")
	errBadTokenSignature = errors.New("alert: invalid token signature")
	errExpiredToken      = errors.New("alert: expired token")
)
