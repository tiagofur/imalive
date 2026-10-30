// Package msg carries the outbound messaging abstraction for user-facing
// messages that are NOT one-time codes (those live in internal/platform/otp):
// the contact welcome (R-03) today, the escalation notices of week 2 later.
// Delivery is delegated to a Notifier so the local sandbox and the future
// Twilio WhatsApp/SMS share the same call path.
package msg

import "context"

// Notifier delivers a text message to a phone. Implementations must never
// log the message body (it carries names).
type Notifier interface {
	Send(ctx context.Context, phone, text string) error
}
