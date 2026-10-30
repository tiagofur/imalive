// Alert link tokens (R-08): the web capability that lets a contact act
// without app or login. Signature and expiry are the whole gate; the
// incident state machine makes every action idempotent.
package escalation_test

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/tiagofur/imalive/backend/internal/escalation"
)

func TestAlertTokens(t *testing.T) {
	secret := []byte("test-alert-secret")
	clock := &fakeClock{now: time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)}
	signer := escalation.NewAlertTokenSigner(secret, "https://tranqui.example", clock)
	incident := uuid.Must(uuid.NewV7())
	contact := uuid.Must(uuid.NewV7())

	t.Run("roundtrip signs and verifies", func(t *testing.T) {
		token, err := signer.Sign(incident, contact)
		if err != nil {
			t.Fatalf("Sign: %v", err)
		}
		gotIncident, gotContact, exp, err := escalation.VerifyAlertToken(secret, token, clock.Now())
		if err != nil {
			t.Fatalf("Verify: %v", err)
		}
		if gotIncident != incident || gotContact != contact {
			t.Fatalf("verified (%v, %v), want (%v, %v)", gotIncident, gotContact, incident, contact)
		}
		// The link lives 7 days: the alert is actionable for days, not hours.
		if !exp.Equal(clock.Now().Add(7 * 24 * time.Hour)) {
			t.Fatalf("expiry = %v, want now+7d", exp)
		}
		if signer.BaseURL() != "https://tranqui.example" {
			t.Fatalf("base URL = %q", signer.BaseURL())
		}
	})

	t.Run("a tampered token is rejected", func(t *testing.T) {
		token, _ := signer.Sign(incident, contact)
		other := uuid.Must(uuid.NewV7())
		forged, _ := escalation.NewAlertTokenSigner([]byte("attacker-secret"), "", clock).Sign(incident, other)
		for name, bad := range map[string]string{
			"flipped char": token[:len(token)-3] + "xyz",
			"other secret": forged,
			"no signature": token[:len(token)/2],
			"empty":        "",
		} {
			t.Run(name, func(t *testing.T) {
				if _, _, _, err := escalation.VerifyAlertToken(secret, bad, clock.Now()); err == nil {
					t.Fatal("tampered token verified, want rejection")
				}
			})
		}
	})

	t.Run("an expired token is rejected but recognizable", func(t *testing.T) {
		token, _ := signer.Sign(incident, contact)
		_, _, exp, err := escalation.VerifyAlertToken(secret, token, clock.Now().Add(8*24*time.Hour))
		if err == nil {
			t.Fatal("expired token verified, want rejection")
		}
		if exp.IsZero() {
			t.Fatalf("error %v without the parsed expiry, want it (the page can say 'expired')", err)
		}
	})
}
