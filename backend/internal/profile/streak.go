package profile

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// The streak (R-12): consecutive local check-in days, frozen — not broken —
// by the pause window. It is a pure function over facts already persisted
// (check-in dates + the pause window), so the retention rule the product
// lives on is testable without a database.
const (
	// streakLookback caps the backward walk: far beyond any plausible
	// streak, it bounds the work per request.
	streakLookback = 400
	// streakGraceDays: a streak whose latest check-in is yesterday is still
	// alive — today has not closed yet (the user's slot may be at night).
	streakGraceDays = 1
)

// CalculateStreak counts the consecutive check-in days walking backward from
// today (the user's local date). Rules:
//   - a day with a check-in adds 1 and continues;
//   - a day without a check-in INSIDE the pause window [since.date, until.date]
//     is skipped without adding or breaking (the freeze of R-11/R-12);
//   - a day without a check-in outside the window stops the count;
//   - a streak whose latest check-in is yesterday still counts (today's
//     check-in is pending, not missed).
//
// dates are the recent check-in cycle dates (any order, deduplicated here).
func CalculateStreak(dates []time.Time, pauseSince, pauseUntil pgtype.Timestamptz, today time.Time) int {
	seen := make(map[string]struct{}, len(dates))
	for _, d := range dates {
		seen[dayKey(d)] = struct{}{}
	}

	day := today
	// The walk starts today: if today is unconfirmed the grace window keeps
	// the streak alive one day (yesterday's chain still stands).
	streak := 0
	for i := 0; i < streakLookback; i++ {
		if _, ok := seen[dayKey(day)]; ok {
			streak++
			day = day.AddDate(0, 0, -1)
			continue
		}
		if i < streakGraceDays {
			// Today unconfirmed: not a miss yet — step back without breaking.
			day = day.AddDate(0, 0, -1)
			continue
		}
		if pausedOn(day, pauseSince, pauseUntil) {
			// Frozen: the pause covers this day (R-11 protects the streak).
			day = day.AddDate(0, 0, -1)
			continue
		}
		return streak
	}
	return streak
}

// pausedOn reports whether the (local) day falls inside the pause window.
func pausedOn(day time.Time, since, until pgtype.Timestamptz) bool {
	if !since.Valid || !until.Valid {
		return false
	}
	// Compare in day boundaries of the day's own location: the pause covers
	// [since, until] as instants; a day is covered when its 24h span overlaps.
	dayStart := day.Truncate(24 * time.Hour)
	dayEnd := dayStart.Add(24 * time.Hour)
	return until.Time.After(dayStart) && since.Time.Before(dayEnd)
}

func dayKey(d time.Time) string {
	return d.Format("2006-01-02")
}
