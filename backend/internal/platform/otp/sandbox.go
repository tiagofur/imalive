package otp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SandboxSender "delivers" codes without any network: each code is appended
// as a JSON line to a gitignored outbox file the developer can read to log
// in locally. The file lives outside the repo by default (TMPDIR) precisely
// so real-looking codes never land in a tracked path.
type SandboxSender struct {
	mu   sync.Mutex
	path string
}

// NewSandboxSender writes to path; an empty path uses the OS temp dir.
func NewSandboxSender(path string) *SandboxSender {
	if path == "" {
		path = filepath.Join(os.TempDir(), "imalive-sandbox-outbox.jsonl")
	}
	return &SandboxSender{path: path}
}

// Path exposes where the sandbox writes (for tests and the README).
func (s *SandboxSender) Path() string { return s.path }

type sandboxRecord struct {
	Phone     string    `json:"phone"`
	Code      string    `json:"code"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

// Send appends the code to the outbox. It never fails the request path:
// sandbox delivery problems surface as errors to keep parity with real
// senders (a real SMTP/WhatsApp failure must fail the request).
func (s *SandboxSender) Send(_ context.Context, phone, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := sandboxRecord{Phone: phone, Code: code, Kind: "otp", CreatedAt: time.Now().UTC()}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("otp: sandbox open: %w", err)
	}
	defer func() { _ = f.Close() }() // body already encoded; close errors are irrelevant here
	if err := json.NewEncoder(f).Encode(rec); err != nil {
		return fmt.Errorf("otp: sandbox write: %w", err)
	}
	return nil
}
