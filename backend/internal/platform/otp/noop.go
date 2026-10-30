package otp

import "context"

// NoopSender accepts every code and delivers nothing: the default for unit
// tests and the fail-safe default when no sender is configured.
type NoopSender struct{}

// Send accepts every code and delivers nothing.
func (NoopSender) Send(context.Context, string, string) error { return nil }
