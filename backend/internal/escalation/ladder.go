package escalation

import "time"

// Ladder job kinds (escalation_jobs.kind CHECK, migration 000005). One job
// per rung: the outbox row it produces carries the job_id for the
// at-least-once dedupe.
const (
	KindReminderSoft   = "reminder_soft"   // gentle nudge to the user
	KindReminderUrgent = "reminder_urgent" // "we are about to alert your circle"
	KindNotifyContact  = "notify_contact"  // contact 1 (position 1)
	KindNotifyCircle   = "notify_circle"   // the whole circle
	// KindSosAlert is the R-10 manual emergency: it bypasses the ladder and
	// lands in the circle after the 10 s server-side countdown (job run_at).
	KindSosAlert = "sos_alert"
)

// Ladder offsets from the soft reminder, per the issue #54 ladder
// ("suave → +4h urgente → +8h contacto 1 → +12h círculo"). Per-user product
// configuration of these offsets is F2 territory; the grace window already
// lives on users.escalation_window_minutes.
const (
	timeAfterSoft = 4 * time.Hour
)

// Step is one rung of the escalation ladder of a daily cycle.
type Step struct {
	Kind  string
	RunAt time.Time
}

// Schedule builds the escalation ladder for a cycle opening at open (the
// user's local check-in time): a grace window of silence first (default 1h,
// the spec state machine), then the soft reminder, then the rest at fixed
// offsets. RunAt instants carry the opening location, so DST transitions
// resolve correctly when converted to UTC by the caller.
func Schedule(open time.Time, grace time.Duration) []Step {
	soft := open.Add(grace)
	return []Step{
		{Kind: KindReminderSoft, RunAt: soft},
		{Kind: KindReminderUrgent, RunAt: soft.Add(timeAfterSoft)},
		{Kind: KindNotifyContact, RunAt: soft.Add(2 * timeAfterSoft)},
		{Kind: KindNotifyCircle, RunAt: soft.Add(3 * timeAfterSoft)},
	}
}
