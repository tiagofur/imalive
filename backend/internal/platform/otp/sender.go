package otp

import (
	"fmt"

	"github.com/tiagofur/imalive/backend/internal/platform/config"
)

// SenderConfig carries the optional delivery settings beyond the base
// config.OtpSender selection.
type SenderConfig struct {
	// SandboxPath overrides the sandbox outbox location (tests).
	SandboxPath string
	// TwilioAccountSid / TwilioAuthToken / TwilioVerifyServiceSid are the
	// Verify API credentials (TWILIO_* env). Required only when
	// cfg.OtpSender == "twilio"; missing credentials fail fast here so a
	// production process with a real sender never boots half-configured.
	TwilioAccountSid        string
	TwilioAuthToken         string
	TwilioVerifyServiceSid  string
	TwilioBaseURLForTesting string
}

// NewFromConfig builds the Sender selected by cfg.OtpSender — the single
// factory every binary uses (cmd/api, cmd/worker, tests).
func NewFromConfig(cfg config.Config, extra SenderConfig) (Sender, error) {
	switch cfg.OtpSender {
	case "noop":
		return NoopSender{}, nil
	case "sandbox":
		return NewSandboxSender(extra.SandboxPath), nil
	case "twilio":
		if extra.TwilioAccountSid == "" || extra.TwilioAuthToken == "" || extra.TwilioVerifyServiceSid == "" {
			return nil, fmt.Errorf("otp: OTP_SENDER=twilio requires TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN and TWILIO_VERIFY_SERVICE_SID (issue #6)")
		}
		return NewTwilioSender(extra.TwilioAccountSid, extra.TwilioAuthToken, extra.TwilioVerifyServiceSid, extra.TwilioBaseURLForTesting), nil
	default:
		// config fail-fast already rejects unknown senders; this is a belt.
		return nil, fmt.Errorf("otp: unknown sender %q", cfg.OtpSender)
	}
}
