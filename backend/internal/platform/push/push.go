// Package push carries the outbound push abstraction (R-05): the daily
// reminder and escalation notices reach the device through FCM HTTP v1 in
// production. The sandbox stub records deliveries and deduplicates by event
// id — the idempotency the real provider path must keep (issue #57); the
// real FCM sender lands behind the same interface once service-account
// credentials exist (F0 block: #6-adjacent, never unverifiable network code).
package push

import (
	"context"
	"sync"
	"time"
)

// Sender delivers one push to one device token. eventID is the outbox
// identity: implementations must use it to avoid double delivery on the
// at-least-once re-sends.
type Sender interface {
	Push(ctx context.Context, token, title, body, eventID string) error
}

// Delivery is one recorded sandbox delivery (for tests and local reading).
type Delivery struct {
	Token     string    `json:"token"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	EventID   string    `json:"event_id"`
	CreatedAt time.Time `json:"created_at"`
}

// SandboxSender records deliveries in memory and deduplicates by eventID.
type SandboxSender struct {
	mu   sync.Mutex
	sent map[string]Delivery
}

// NewSandboxSender builds the sandbox stub.
func NewSandboxSender() *SandboxSender {
	return &SandboxSender{sent: map[string]Delivery{}}
}

// Push records the delivery once per eventID; a re-send of the same event is
// a no-op (the provider-side dedupe, proven in tests).
func (s *SandboxSender) Push(_ context.Context, token, title, body, eventID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.sent[eventID]; ok {
		return nil
	}
	s.sent[eventID] = Delivery{
		Token: token, Title: title, Body: body, EventID: eventID,
		CreatedAt: time.Now().UTC(),
	}
	return nil
}

// Deliveries returns the recorded deliveries (tests).
func (s *SandboxSender) Deliveries() []Delivery {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Delivery, 0, len(s.sent))
	for _, d := range s.sent {
		out = append(out, d)
	}
	return out
}
