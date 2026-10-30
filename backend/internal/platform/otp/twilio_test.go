package otp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tiagofur/imalive/backend/internal/platform/config"
)

func TestTwilioSenderPostsVerificationWithBasicAuth(t *testing.T) {
	var gotPath, gotAuth, gotForm string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotForm = string(buf)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"status":"pending"}`))
	}))
	defer srv.Close()

	sender := NewTwilioSender("ACtest", "authtest", "VA_service", srv.URL)
	if err := sender.Send(context.Background(), "+5215511111111", "123456"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if gotPath != "/v2/Services/VA_service/Verifications" {
		t.Fatalf("path = %q, want the Verify v2 endpoint with the service sid", gotPath)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") || gotAuth == "Basic " {
		t.Fatalf("Authorization = %q, want Basic credentials", gotAuth)
	}
	if gotForm != "To=+5215511111111&Channel=sms" {
		t.Fatalf("form = %q, want To+Channel", gotForm)
	}
}

func TestTwilioSenderClassifiesProviderErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":60200,"message":"Invalid parameter"}`))
	}))
	defer srv.Close()

	sender := NewTwilioSender("ACtest", "authtest", "VA_service", srv.URL)
	err := sender.Send(context.Background(), "+5215511111111", "123456")
	if err == nil {
		t.Fatal("Send = nil error, want the provider rejection classified")
	}
	if !strings.Contains(err.Error(), "60200") || strings.Contains(err.Error(), "authtest") {
		t.Fatalf("error = %v, want the twilio code surfaced and no credentials leaked", err)
	}
}

func TestNewFromConfig(t *testing.T) {
	t.Run("selects noop, sandbox and twilio by config", func(t *testing.T) {
		cfg := config.Config{OtpSender: "noop"}
		if _, err := NewFromConfig(cfg, SenderConfig{}); err != nil {
			t.Fatalf("noop: %v", err)
		}
		cfg.OtpSender = "sandbox"
		if _, err := NewFromConfig(cfg, SenderConfig{}); err != nil {
			t.Fatalf("sandbox: %v", err)
		}
		cfg.OtpSender = "twilio"
		if _, err := NewFromConfig(cfg, SenderConfig{}); err == nil {
			t.Fatal("twilio without credentials = nil error, want fail-fast")
		}
		tw := SenderConfig{TwilioAccountSid: "AC", TwilioAuthToken: "tk", TwilioVerifyServiceSid: "VA"}
		if s, err := NewFromConfig(cfg, tw); err != nil {
			t.Fatalf("twilio with credentials: %v", err)
		} else if _, ok := s.(*TwilioSender); !ok {
			t.Fatalf("sender = %T, want *TwilioSender", s)
		}
	})
}
