// The streak is the retention rule the product lives on (R-12): pure,
// table-driven, no database. The integration path (seeding check-ins and
// the pause window, reading it off /v1/me) lives in profile_test.go.
package profile

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

func streakDay(y int, m time.Month, d int) time.Time {
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func tz(instant time.Time) pgtype.Timestamptz { return pgtype.Timestamptz{Time: instant, Valid: true} }

func TestCalculateStreak(t *testing.T) {
	today := streakDay(2026, time.October, 30)

	t.Run("counts consecutive days including today", func(t *testing.T) {
		dates := []time.Time{
			streakDay(2026, time.October, 28), streakDay(2026, time.October, 29), today,
		}
		if got := CalculateStreak(dates, pgtype.Timestamptz{}, pgtype.Timestamptz{}, today); got != 3 {
			t.Fatalf("streak = %d, want 3", got)
		}
	})

	t.Run("today unconfirmed still shows yesterday's chain", func(t *testing.T) {
		dates := []time.Time{streakDay(2026, time.October, 28), streakDay(2026, time.October, 29)}
		if got := CalculateStreak(dates, pgtype.Timestamptz{}, pgtype.Timestamptz{}, today); got != 2 {
			t.Fatalf("streak = %d, want 2 (the day has not closed yet)", got)
		}
	})

	t.Run("a missed day outside the pause resets to zero", func(t *testing.T) {
		dates := []time.Time{streakDay(2026, time.October, 28)}
		if got := CalculateStreak(dates, pgtype.Timestamptz{}, pgtype.Timestamptz{}, today); got != 0 {
			t.Fatalf("streak = %d, want 0 (the 29th was missed)", got)
		}
	})

	t.Run("a missed day inside the pause window freezes instead", func(t *testing.T) {
		// The user left on the 27th, paused the ladder, came back on the 29th.
		dates := []time.Time{streakDay(2026, time.October, 25), streakDay(2026, time.October, 26), streakDay(2026, time.October, 29)}
		since := tz(streakDay(2026, time.October, 27).Add(12 * time.Hour))
		until := tz(streakDay(2026, time.October, 29).Add(8 * time.Hour))
		if got := CalculateStreak(dates, since, until, today); got != 3 {
			t.Fatalf("streak = %d, want 3 (the 27th and 28th are frozen: skipped, neither added nor breaking)", got)
		}
	})

	t.Run("a pause that expired before the gap does not protect it", func(t *testing.T) {
		// The pause ended on the 25th; the missed day is the 29th.
		dates := []time.Time{streakDay(2026, time.October, 28)}
		since := tz(streakDay(2026, time.October, 24).Add(12 * time.Hour))
		until := tz(streakDay(2026, time.October, 25).Add(8 * time.Hour))
		if got := CalculateStreak(dates, since, until, today); got != 0 {
			t.Fatalf("streak = %d, want 0 (the pause window is long gone)", got)
		}
	})

	t.Run("without check-ins the streak is zero", func(t *testing.T) {
		if got := CalculateStreak(nil, pgtype.Timestamptz{}, pgtype.Timestamptz{}, today); got != 0 {
			t.Fatalf("streak = %d, want 0", got)
		}
	})

	t.Run("duplicates and disorder do not fool the count", func(t *testing.T) {
		dates := []time.Time{today, streakDay(2026, time.October, 29), today, streakDay(2026, time.October, 29)}
		if got := CalculateStreak(dates, pgtype.Timestamptz{}, pgtype.Timestamptz{}, today); got != 2 {
			t.Fatalf("streak = %d, want 2", got)
		}
	})
}
