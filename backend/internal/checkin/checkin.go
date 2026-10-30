// Package checkin owns the daily signal domain (R-04). The store underneath
// is sqlc-generated from the same DDL that ships to production.
package checkin

// CycleDate is the check-in day in the user's timezone — not the UTC date.
// In LATAM a UTC date flips at mid-afternoon and would break the per-day
// uniqueness around the 21:00 check-in (documented in migration 000003).
