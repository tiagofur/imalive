package msg

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// SandboxNotifier appends each message as a JSON line to a gitignored
// outbox outside the repo (same pattern as otp.SandboxSender): local
// development reads the exact texts that production would deliver.
type SandboxNotifier struct {
	mu   sync.Mutex
	path string
}

// NewSandboxNotifier writes to path; an empty path uses the OS temp dir.
func NewSandboxNotifier(path string) *SandboxNotifier {
	if path == "" {
		path = filepath.Join(os.TempDir(), "imalive-sandbox-outbox.jsonl")
	}
	return &SandboxNotifier{path: path}
}

// Path exposes where the sandbox writes (for tests and the README).
func (s *SandboxNotifier) Path() string { return s.path }

type sandboxRecord struct {
	Phone     string    `json:"phone"`
	Text      string    `json:"text"`
	Kind      string    `json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

// Send appends the message to the outbox; delivery problems fail the call
// (parity with a real provider).
func (s *SandboxNotifier) Send(_ context.Context, phone, text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec := sandboxRecord{Phone: phone, Text: text, Kind: "message", CreatedAt: time.Now().UTC()}
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("msg: sandbox open: %w", err)
	}
	defer func() { _ = f.Close() }()
	if err := json.NewEncoder(f).Encode(rec); err != nil {
		return fmt.Errorf("msg: sandbox write: %w", err)
	}
	return nil
}

// NoopNotifier accepts every message and delivers nothing (tests).
type NoopNotifier struct{}

// Send accepts every message and delivers nothing.
func (NoopNotifier) Send(context.Context, string, string) error { return nil }
