// Package escalation is the heart of the product (R-07, ADR-0004): given a
// user with a chosen check-in time, it opens the daily cycle, escalates the
// ladder on silence (soft → urgent → contact 1 → full circle) and closes the
// cycle when a life signal arrives. Every state change that must notify
// writes an outbox_event in the same transaction; every transition leaves an
// append-only audit_events row. Notifications themselves are sent by the
// outbox dispatcher (#57), never by this package.
//
// The engine spec is docs/arquitectura/motor-escalada.md — if this package
// fails, a real person may not get help.
package escalation

import "time"

// Clock is the engine's notion of "now": injectable so the whole motor runs
// under simulated time in tests (motor-escalada.md §Reloj e idempotencia).
// Scheduling queries take the clock value as a parameter — never now() from
// Postgres — so the fake clock controls the entire test universe.
type Clock interface {
	Now() time.Time
}

// SystemClock reads the host wall clock in UTC. Production wiring; tests
// supply their own fake.
type SystemClock struct{}

// Now implements Clock.
func (SystemClock) Now() time.Time { return time.Now().UTC() }
