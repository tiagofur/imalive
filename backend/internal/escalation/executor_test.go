package escalation_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/tiagofur/imalive/backend/internal/escalation"
	"github.com/tiagofur/imalive/backend/internal/escalation/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

// The execution tests drive the whole motor under simulated time: a full day
// of silence walks the ladder to the circle; a pause landing mid-flight
// cancels instead of notifying (motor-escalada.md, escenarios obligatorios
// 2, 6 y 9; los 3/4/5 son del incidente #55).
func TestEngineExecutesLadder(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	local := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, loc).UTC()
	}

	t.Run("a full day of silence walks the ladder to the circle", func(t *testing.T) {
		userID := p.SeedUser("+5215530000001")
		contactPhone := "+5215530000099"
		seedContact(t, p, userID, "María", contactPhone, 1)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx) // opens the cycle at the slot

		type rung struct {
			at    time.Time
			kind  string
			phone string
		}
		rungs := []rung{
			{local(2026, 10, 30, 22, 0), escalation.KindReminderSoft, "+5215530000001"},
			{local(2026, 10, 31, 2, 0), escalation.KindReminderUrgent, "+5215530000001"},
			{local(2026, 10, 31, 6, 0), escalation.KindNotifyContact, contactPhone},
			{local(2026, 10, 31, 10, 0), escalation.KindNotifyCircle, contactPhone},
		}
		for i, r := range rungs {
			clock.now = r.at
			engine.ProcessTick(ctx)
			if n := outboxCount(t, p, userID); n != i+1 {
				t.Fatalf("after rung %d: outbox = %d events, want %d", i+1, n, i+1)
			}
			if got := lastEventDestination(t, p, userID); got != r.phone {
				t.Fatalf("rung %s notified %s, want %s", r.kind, got, r.phone)
			}
			job := jobOf(t, p, userID, r.kind)
			if job.State != "done" {
				t.Fatalf("rung %s state = %s, want done", r.kind, job.State)
			}
		}

		// Evidence per rung kind: the reminders notified, the incident rungs
		// opened and escalated the incident (spec §Auditoría).
		if n := auditKindCount(t, p, userID, escalation.AuditLadderNotified); n != 2 {
			t.Fatalf("audit ladder.notified = %d rows, want 2 (the reminders)", n)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditIncidentOpened); n != 1 {
			t.Fatalf("audit incident.opened = %d rows, want 1", n)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditIncidentCircleNotified); n != 1 {
			t.Fatalf("audit incident.circle_notified = %d rows, want 1", n)
		}
		// Re-running the tick at the same instant cannot duplicate anything.
		engine.ProcessTick(ctx)
		if n := outboxCount(t, p, userID); n != 4 {
			t.Fatalf("outbox after re-tick = %d events, want 4 (no duplicates)", n)
		}
	})

	t.Run("no contacts: the contact rungs complete without notifications", func(t *testing.T) {
		userID := p.SeedUser("+5215530000002")
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		clock.now = local(2026, 10, 31, 6, 0)
		engine.ProcessTick(ctx)
		// The two reminder rungs fired to the user's own phone; the contact
		// rung had nobody to notify and completed without crashing.
		if n := outboxCount(t, p, userID); n != 2 {
			t.Fatalf("outbox with no contacts = %d events, want 2 (only the user's reminders)", n)
		}
		if job := jobOf(t, p, userID, escalation.KindNotifyContact); job.State != "done" {
			t.Fatalf("notify_contact state = %s, want done (never crashes on an empty circle)", job.State)
		}
	})

	t.Run("a pause landing mid-flight cancels the rung instead of notifying", func(t *testing.T) {
		userID := p.SeedUser("+5215530000003")
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx) // cycle open, soft not due yet

		// The user pauses before the soft reminder is due (R-11).
		if _, err := p.Pool.Exec(ctx,
			`UPDATE users SET pause_until = $2 WHERE id = $1`,
			userID, pgTz(local(2026, 10, 31, 12, 0))); err != nil {
			t.Fatalf("seed pause: %v", err)
		}
		clock.now = local(2026, 10, 30, 22, 0)
		engine.ProcessTick(ctx)

		if n := outboxCount(t, p, userID); n != 0 {
			t.Fatalf("outbox while paused = %d events, want 0 (nothing notifies during a pause)", n)
		}
		job := jobOf(t, p, userID, escalation.KindReminderSoft)
		if job.State != "cancelled" {
			t.Fatalf("soft state = %s, want cancelled", job.State)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditLadderFrozen); n != 1 {
			t.Fatalf("audit frozen = %d rows, want 1 (the why is on the record)", n)
		}
	})
}

func seedContact(t *testing.T, p *dbtest.Pool, userID uuid.UUID, name, phone string, position int) uuid.UUID {
	t.Helper()
	id := mustUUID(t)
	if _, err := p.Pool.Exec(context.Background(),
		`INSERT INTO contacts (id, user_id, display_name, phone, position)
		 VALUES ($1, $2, $3, $4, $5)`,
		id, userID, name, phone, position); err != nil {
		t.Fatalf("seed contact: %v", err)
	}
	return id
}

func outboxCount(t *testing.T, p *dbtest.Pool, userID uuid.UUID) int {
	t.Helper()
	var n int
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM outbox_event WHERE user_id = $1`, userID).Scan(&n); err != nil {
		t.Fatalf("count outbox: %v", err)
	}
	return n
}

func lastEventDestination(t *testing.T, p *dbtest.Pool, userID uuid.UUID) string {
	t.Helper()
	var dest string
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT destination FROM outbox_event WHERE user_id = $1
		 ORDER BY created_at DESC, id DESC LIMIT 1`, userID).Scan(&dest); err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	return dest
}

func auditKindCount(t *testing.T, p *dbtest.Pool, userID uuid.UUID, kind string) int {
	t.Helper()
	var n int
	if err := p.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM audit_events WHERE user_id = $1 AND kind = $2`, userID, kind).Scan(&n); err != nil {
		t.Fatalf("count audit %s: %v", kind, err)
	}
	return n
}

// TestEngineIncidents drives the incident state machine (R-08 + R-09) under
// simulated time: the contact rung opens it with the frozen last known
// signal, the circle rung escalates it, and the user's late signal closes it
// (resolved after a claim, false alarm without) with an "all clear" to the
// circle — every transition inside the same transaction as its notifications.
func TestEngineIncidents(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	local := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, loc).UTC()
	}

	seedCheckin := func(userID uuid.UUID, cycleDay time.Time, at time.Time) {
		t.Helper()
		// pgtype.Date encodes the UTC date: normalize to UTC midnight or a
		// local instant silently shifts the cycle date by the offset.
		day := time.Date(cycleDay.Year(), cycleDay.Month(), cycleDay.Day(), 0, 0, 0, 0, time.UTC)
		if _, err := p.Pool.Exec(ctx,
			`INSERT INTO checkins (id, user_id, cycle_date, checked_in_at)
			 VALUES ($1, $2, $3, $4)`, mustUUID(t), userID, pgDate(day), pgTz(at)); err != nil {
			t.Fatalf("seed checkin: %v", err)
		}
	}
	incidentOf := func(userID uuid.UUID) store.Incident {
		t.Helper()
		rows, err := p.Pool.Query(ctx,
			`SELECT id, user_id, cycle_date, state, claimed_by, last_signal_at, opened_at, closed_at
			 FROM incidents WHERE user_id = $1 ORDER BY opened_at DESC LIMIT 1`, userID)
		if err != nil {
			t.Fatalf("query incident: %v", err)
		}
		defer rows.Close()
		var inc store.Incident
		found := false
		for rows.Next() {
			if err := rows.Scan(&inc.ID, &inc.UserID, &inc.CycleDate, &inc.State,
				&inc.ClaimedBy, &inc.LastSignalAt, &inc.OpenedAt, &inc.ClosedAt); err != nil {
				t.Fatalf("scan incident: %v", err)
			}
			found = true
		}
		if !found {
			t.Fatal("incident not found")
		}
		return inc
	}

	t.Run("the contact rung opens the incident with the frozen last signal", func(t *testing.T) {
		userID := p.SeedUser("+5215570000001")
		contactPhone := "+5215570000091"
		seedContact(t, p, userID, "María", contactPhone, 1)
		signal := local(2026, 10, 29, 21, 5) // yesterday's check-in: the last known signal
		// local() returns UTC instants: the cycle day must be the civil
		// LOCAL date, passed explicitly as UTC midnight.
		seedCheckin(userID, time.Date(2026, 10, 29, 0, 0, 0, 0, time.UTC), signal)

		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx) // opens the cycle
		clock.now = local(2026, 10, 31, 6, 0)
		engine.ProcessTick(ctx) // the contact rung fires

		inc := incidentOf(userID)
		if inc.State != escalation.IncidentContact1Notified {
			t.Fatalf("incident state = %s, want contact1_notified", inc.State)
		}
		if inc.LastSignalAt.Time.UTC() != signal.UTC() {
			t.Fatalf("last_signal_at = %v, want the frozen %v", inc.LastSignalAt.Time, signal)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditIncidentOpened); n != 1 {
			t.Fatalf("incident.opened audit rows = %d, want 1", n)
		}
		ev := lastEventFields(t, p, userID)
		if ev.destination != contactPhone {
			t.Fatalf("alert went to %s, want the contact", ev.destination)
		}
		if ev.incidentID != inc.ID {
			t.Fatalf("alert incident_id = %v, want the incident", ev.incidentID)
		}
		if ev.payload["event"] != "incident.alert" || ev.payload["user_name"] == "" {
			t.Fatalf("alert payload = %v, want an incident.alert with the user name", ev.payload)
		}
	})

	t.Run("the circle rung escalates a contact1_notified incident", func(t *testing.T) {
		userID := p.SeedUser("+5215570000002")
		contact1 := "+5215570000092"
		contact2 := "+5215570000093"
		seedContact(t, p, userID, "María", contact1, 1)
		seedContact(t, p, userID, "Beto", contact2, 2)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		clock.now = local(2026, 10, 31, 6, 0)
		engine.ProcessTick(ctx) // contact rung: opens the incident
		clock.now = local(2026, 10, 31, 10, 0)
		engine.ProcessTick(ctx) // circle rung

		if inc := incidentOf(userID); inc.State != escalation.IncidentCircleNotified {
			t.Fatalf("incident state = %s, want circle_notified", inc.State)
		}
		if n := outboxCount(t, p, userID); n != 5 { // 2 reminders + contact alert + 2 circle alerts
			t.Fatalf("outbox = %d events, want 5", n)
		}
		// The circle heard both phones (dedupe by job keeps them distinct).
		if n := auditKindCount(t, p, userID, escalation.AuditIncidentCircleNotified); n != 1 {
			t.Fatalf("incident.circle_notified audit rows = %d, want 1", n)
		}
	})

	t.Run("the late signal closes claimed as resolved and fresh as false alarm", func(t *testing.T) {
		userID := p.SeedUser("+5215570000003")
		contactPhone := "+5215570000094"
		contactID := seedContact(t, p, userID, "María", contactPhone, 1)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		clock.now = local(2026, 10, 31, 6, 0)
		engine.ProcessTick(ctx) // incident opens

		// The contact claims (the web action of #55, simulated via the store).
		inc := incidentOf(userID)
		if _, err := store.New(p.Pool).ClaimIncident(ctx, store.ClaimIncidentParams{
			ID: inc.ID, ClaimedBy: pgUUID(contactID),
		}); err != nil {
			t.Fatalf("claim: %v", err)
		}

		// The user finally checks in: resolved + "all clear" to the circle.
		tx, err := p.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if err := engine.SignalReceivedTx(ctx, tx, userID, pgDate(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC))); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("SignalReceivedTx: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
		if got := incidentOf(userID); got.State != escalation.IncidentClosedResolved {
			t.Fatalf("claimed incident closed as %s, want closed_resolved", got.State)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditIncidentClosed); n != 1 {
			t.Fatalf("incident.closed audit rows = %d, want 1", n)
		}
		last := lastEventFields(t, p, userID)
		if last.payload["event"] != "incident.all_clear" {
			t.Fatalf("last event = %v, want the all_clear to the circle", last.payload)
		}

		// A user with no claimant closes as false alarm instead.
		other := p.SeedUser("+5215570000005")
		seedContact(t, p, other, "Ana", "+5215570000095", 1)
		engine2, clock2 := testEngine(p, local(2026, 10, 30, 21, 30))
		engine2.ProcessTick(ctx)
		clock2.now = local(2026, 10, 31, 6, 0)
		engine2.ProcessTick(ctx)
		tx2, err := p.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		if err := engine2.SignalReceivedTx(ctx, tx2, other, pgDate(time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC))); err != nil {
			_ = tx2.Rollback(ctx)
			t.Fatalf("SignalReceivedTx: %v", err)
		}
		if err := tx2.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
		if got := incidentOf(other); got.State != escalation.IncidentClosedFalseAlarm {
			t.Fatalf("unclaimed incident closed as %s, want closed_false_alarm", got.State)
		}
	})

	t.Run("no live incident: the circle rung completes without notifying", func(t *testing.T) {
		userID := p.SeedUser("+5215570000006")
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		// The contact rung never fired (job manipulated away): no incident.
		if _, err := p.Pool.Exec(ctx,
			`UPDATE escalation_jobs SET state = 'cancelled' WHERE user_id = $1 AND kind = 'notify_contact'`, userID); err != nil {
			t.Fatalf("cancel job: %v", err)
		}
		clock.now = local(2026, 10, 31, 10, 0)
		engine.ProcessTick(ctx) // circle rung with no live incident

		if job := jobOf(t, p, userID, escalation.KindNotifyCircle); job.State != "done" {
			t.Fatalf("circle rung state = %s, want done (skips, never crashes)", job.State)
		}
		if n := outboxCount(t, p, userID); n != 2 { // only the reminders
			t.Fatalf("outbox = %d events, want 2 (no circle alerts without an incident)", n)
		}
	})
}

// lastEventFields reads the newest outbox event with the fields the incident
// tests assert on.
func lastEventFields(t *testing.T, p *dbtest.Pool, userID uuid.UUID) struct {
	destination string
	incidentID  uuid.UUID
	payload     map[string]any
} {
	t.Helper()
	rows, err := p.Pool.Query(context.Background(),
		// Only incident events: uuid v7 ordering is not reliable within the
		// same millisecond, so the reminders could tie with the alert.
		`SELECT destination, incident_id, payload FROM outbox_event
		 WHERE user_id = $1 AND incident_id IS NOT NULL
		 ORDER BY created_at DESC, id DESC LIMIT 1`, userID)
	if err != nil {
		t.Fatalf("query outbox: %v", err)
	}
	defer rows.Close()
	var dest string
	var incident pgtype.UUID
	var payload []byte
	found := false
	for rows.Next() {
		if err := rows.Scan(&dest, &incident, &payload); err != nil {
			t.Fatalf("scan outbox: %v", err)
		}
		found = true
	}
	if !found {
		t.Fatal("no outbox event found")
	}
	var m map[string]any
	if err := json.Unmarshal(payload, &m); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	incID := uuid.Nil
	if incident.Valid {
		incID = uuid.UUID(incident.Bytes)
	}
	return struct {
		destination string
		incidentID  uuid.UUID
		payload     map[string]any
	}{dest, incID, m}
}

// pgUUID converts a domain uuid to the store type from the black-box side.
func pgUUID(u uuid.UUID) pgtype.UUID { return pgtype.UUID{Bytes: u, Valid: true} }

// TestEngineSos drives the R-10 manual emergency: arm (10 s countdown) →
// fire (straight to the circle) or cancel inside the window (nobody hears
// anything), with the incident machine reused, not reinvented.
func TestEngineSos(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	loc, err := time.LoadLocation("America/Mexico_City")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}
	local := func(y int, mo time.Month, d, h, mi int) time.Time {
		return time.Date(y, mo, d, h, mi, 0, 0, loc).UTC()
	}
	signalOf := func(userID uuid.UUID) store.Incident {
		t.Helper()
		var inc store.Incident
		found := false
		rows, err := p.Pool.Query(ctx,
			`SELECT id, user_id, cycle_date, state, claimed_by, last_signal_at, opened_at, closed_at
			 FROM incidents WHERE user_id = $1 ORDER BY opened_at DESC LIMIT 1`, userID)
		if err != nil {
			t.Fatalf("query incident: %v", err)
		}
		defer rows.Close()
		for rows.Next() {
			if err := rows.Scan(&inc.ID, &inc.UserID, &inc.CycleDate, &inc.State,
				&inc.ClaimedBy, &inc.LastSignalAt, &inc.OpenedAt, &inc.ClosedAt); err != nil {
				t.Fatalf("scan incident: %v", err)
			}
			found = true
		}
		if !found {
			t.Fatal("incident not found")
		}
		return inc
	}
	arm := func(engine *escalation.Engine, clock *fakeClock, userID uuid.UUID) time.Time {
		t.Helper()
		tx, err := p.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		runAt, err := engine.SosArmTx(ctx, tx, userID,
			pgDate(time.Date(2026, 10, 30, 0, 0, 0, 0, time.UTC)), clock.now.Add(10*time.Second))
		if err != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("SosArmTx: %v", err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}
		return runAt
	}

	t.Run("the SOS fires after the countdown straight to the circle", func(t *testing.T) {
		userID := p.SeedUser("+5215595000001")
		seedContact(t, p, userID, "María", "+5215595000011", 1)
		seedContact(t, p, userID, "Beto", "+5215595000012", 2)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx) // cycle open (the daily ladder is irrelevant here)

		runAt := arm(engine, clock, userID)
		if !runAt.Equal(clock.now.Add(10 * time.Second)) {
			t.Fatalf("run_at = %v, want now+10s", runAt)
		}
		// Inside the window the job is not due yet.
		clock.now = clock.now.Add(5 * time.Second)
		engine.ProcessTick(ctx)
		if n := outboxCount(t, p, userID); n != 0 {
			t.Fatalf("outbox inside the countdown = %d events, want 0", n)
		}

		clock.now = local(2026, 10, 30, 21, 31)
		engine.ProcessTick(ctx)

		inc := signalOf(userID)
		if inc.State != escalation.IncidentCircleNotified {
			t.Fatalf("sos incident state = %s, want circle_notified (bypasses the ladder)", inc.State)
		}
		// Both contacts alerted with claimable links.
		rows, err := p.Pool.Query(ctx,
			`SELECT destination FROM outbox_event WHERE incident_id = $1 ORDER BY destination`, inc.ID)
		if err != nil {
			t.Fatalf("query alerts: %v", err)
		}
		defer rows.Close()
		var dests []string
		for rows.Next() {
			var d string
			if err := rows.Scan(&d); err != nil {
				t.Fatalf("scan: %v", err)
			}
			dests = append(dests, d)
		}
		if len(dests) != 2 {
			t.Fatalf("sos alerts = %d, want 2 (the whole circle)", len(dests))
		}
		if n := auditKindCount(t, p, userID, escalation.AuditSosArmed); n != 1 {
			t.Fatalf("sos.armed audit rows = %d, want 1", n)
		}
	})

	t.Run("the SOS cancelled inside the window notifies nobody", func(t *testing.T) {
		userID := p.SeedUser("+5215595000002")
		seedContact(t, p, userID, "María", "+5215595000021", 1)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)

		if runAt := arm(engine, clock, userID); runAt.IsZero() {
			t.Fatal("arm returned a zero run_at")
		}
		tx, err := p.Pool.Begin(ctx)
		if err != nil {
			t.Fatalf("begin: %v", err)
		}
		n, err := engine.SosCancelTx(ctx, tx, userID)
		if err != nil || n != 1 {
			_ = tx.Rollback(ctx)
			t.Fatalf("SosCancelTx = %d (err %v), want 1/nil", n, err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatalf("commit: %v", err)
		}

		clock.now = local(2026, 10, 30, 21, 31)
		engine.ProcessTick(ctx)

		if n := outboxCount(t, p, userID); n != 0 {
			t.Fatalf("outbox after cancel = %d events, want 0 (the window held)", n)
		}
		if n := auditKindCount(t, p, userID, escalation.AuditSosCancelled); n != 1 {
			t.Fatalf("sos.cancelled audit rows = %d, want 1 (evidence either way)", n)
		}
		var incidents int
		if err := p.Pool.QueryRow(ctx,
			`SELECT count(*) FROM incidents WHERE user_id = $1`, userID).Scan(&incidents); err != nil || incidents != 0 {
			t.Fatalf("incidents after cancel = %d (err %v), want 0 (nothing fired)", incidents, err)
		}
	})

	t.Run("an SOS during a live incident reuses it and escalates", func(t *testing.T) {
		userID := p.SeedUser("+5215595000003")
		seedContact(t, p, userID, "María", "+5215595000031", 1)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		engine.ProcessTick(ctx)
		// The daily ladder already opened an incident (contact rung fired).
		clock.now = local(2026, 10, 31, 6, 0)
		engine.ProcessTick(ctx)
		before := signalOf(userID)
		if before.State != escalation.IncidentContact1Notified {
			t.Fatalf("seed incident state = %s, want contact1_notified", before.State)
		}

		arm(engine, clock, userID)
		clock.now = local(2026, 10, 31, 6, 1)
		engine.ProcessTick(ctx)

		after := signalOf(userID)
		if after.ID != before.ID {
			t.Fatalf("sos created a second incident (%s), want the same %s", after.ID, before.ID)
		}
		if after.State != escalation.IncidentCircleNotified {
			t.Fatalf("state after sos = %s, want circle_notified", after.State)
		}
	})

	t.Run("a second SOS the same day is allowed", func(t *testing.T) {
		userID := p.SeedUser("+5215595000004")
		seedContact(t, p, userID, "María", "+5215595000041", 1)
		engine, clock := testEngine(p, local(2026, 10, 30, 21, 30))
		arm(engine, clock, userID)
		clock.now = local(2026, 10, 30, 21, 31)
		engine.ProcessTick(ctx) // first SOS fires

		arm2 := arm(engine, clock, userID) // second emergency the same day
		if arm2.IsZero() {
			t.Fatal("second SOS could not be armed (the partial unique must exempt sos_alert)")
		}
	})
}
