package otp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// TwilioSender delivers OTPs through the Twilio Verify API v2. Credentials
// come from config (TWILIO_* env); they are never logged and never embedded
// in wrapped errors (docs/seguridad/seguridad-privacidad.md).
type TwilioSender struct {
	accountSid string
	authToken  string
	serviceSid string
	baseURL    string // overridable in tests
	client     *http.Client
}

// NewTwilioSender builds the sender; baseURL empty means the real API.
func NewTwilioSender(accountSid, authToken, serviceSid, baseURL string) *TwilioSender {
	if baseURL == "" {
		baseURL = "https://verify.twilio.com"
	}
	return &TwilioSender{
		accountSid: accountSid,
		authToken:  authToken,
		serviceSid: serviceSid,
		baseURL:    strings.TrimRight(baseURL, "/"),
		client:     &http.Client{Timeout: 5 * time.Second},
	}
}

// Send starts a verification for the phone over SMS. Twilio generates and
// delivers the code itself; the returned value is the provider's verification
// sid for traceability (never logged with request data).
func (s *TwilioSender) Send(ctx context.Context, phone, _ string) error {
	form := strings.NewReader("To=" + phone + "&Channel=sms")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		s.baseURL+"/v2/Services/"+s.serviceSid+"/Verifications", form)
	if err != nil {
		return fmt.Errorf("otp: twilio request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.accountSid, s.authToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("otp: twilio call: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var body struct {
		Code     int    `json:"code"`
		Message  string `json:"message"`
		Status   string `json:"status"`
		MoreInfo string `json:"more_info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return fmt.Errorf("otp: twilio response (http %d): %w", resp.StatusCode, err)
	}
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("otp: twilio rejected (http %d, twilio code %d): %s", resp.StatusCode, body.Code, body.Message)
	}
	return nil
}
