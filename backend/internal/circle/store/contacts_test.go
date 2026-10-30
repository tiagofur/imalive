package store_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/tiagofur/imalive/backend/internal/circle"
	"github.com/tiagofur/imalive/backend/internal/circle/store"
	"github.com/tiagofur/imalive/backend/internal/platform/db/dbtest"
)

func TestContactsStore(t *testing.T) {
	ctx := context.Background()
	p := dbtest.New(t)
	q := store.New(p.Pool)

	t.Run("creates, lists and deletes a contact", func(t *testing.T) {
		userID := p.SeedUser("+5215511111111")

		created, err := q.CreateContact(ctx, store.CreateContactParams{
			ID:          mustUUID(t),
			UserID:      userID,
			DisplayName: "María",
			Phone:       "+5215522222222",
			Position:    1,
		})
		if err != nil {
			t.Fatalf("CreateContact: %v", err)
		}
		if created.DisplayName != "María" || created.Position != 1 {
			t.Fatalf("created = %+v, want María at position 1", created)
		}

		list, err := q.ListContactsByUser(ctx, userID)
		if err != nil || len(list) != 1 {
			t.Fatalf("ListContactsByUser = %v, %v; want 1 contact", list, err)
		}

		rows, err := q.DeleteContact(ctx, store.DeleteContactParams{ID: created.ID, UserID: userID})
		if err != nil || rows != 1 {
			t.Fatalf("DeleteContact = %d rows, %v; want 1", rows, err)
		}
		if _, err := q.GetContact(ctx, store.GetContactParams{ID: created.ID, UserID: userID}); !errors.Is(err, pgx.ErrNoRows) {
			t.Fatalf("GetContact after delete = %v, want pgx.ErrNoRows", err)
		}
	})

	t.Run("rejects a duplicate phone for the same user", func(t *testing.T) {
		userID := p.SeedUser("+5215511111112")
		params := store.CreateContactParams{UserID: userID, DisplayName: "Luis", Phone: "+5215533333333", Position: 1}
		params.ID = mustUUID(t)
		if _, err := q.CreateContact(ctx, params); err != nil {
			t.Fatalf("first CreateContact: %v", err)
		}
		params.ID = mustUUID(t)
		params.DisplayName = "Luis Duplicado"
		_, err := q.CreateContact(ctx, params)
		assertPgCode(t, err, "23505", "duplicate (user_id, phone)")
	})

	t.Run("rejects a sixth contact via the position ceiling", func(t *testing.T) {
		userID := p.SeedUser("+5215511111113")
		for i := 1; i <= circle.MaxContacts; i++ {
			params := store.CreateContactParams{
				ID:          mustUUID(t),
				UserID:      userID,
				DisplayName: fmt.Sprintf("Contacto %d", i),
				Phone:       fmt.Sprintf("+52155666660%02d", i),
				Position:    int16(i),
			}
			if _, err := q.CreateContact(ctx, params); err != nil {
				t.Fatalf("CreateContact #%d: %v", i, err)
			}
		}
		_, err := q.CreateContact(ctx, store.CreateContactParams{
			ID:          mustUUID(t),
			UserID:      userID,
			DisplayName: "Extra",
			Phone:       "+5215599999999",
			Position:    6,
		})
		assertPgCode(t, err, "23514", "position CHECK 1..5")
	})

	t.Run("rejects the same position for two contacts", func(t *testing.T) {
		userID := p.SeedUser("+5215511111114")
		first := store.CreateContactParams{ID: mustUUID(t), UserID: userID, DisplayName: "A", Phone: "+5215577770001", Position: 1}
		if _, err := q.CreateContact(ctx, first); err != nil {
			t.Fatalf("first CreateContact: %v", err)
		}
		second := store.CreateContactParams{ID: mustUUID(t), UserID: userID, DisplayName: "B", Phone: "+5215577770002", Position: 1}
		_, err := q.CreateContact(ctx, second)
		assertPgCode(t, err, "23505", "duplicate (user_id, position)")
	})
}

func mustUUID(t *testing.T) uuid.UUID {
	t.Helper()
	id, err := uuid.NewV7()
	if err != nil {
		t.Fatalf("uuid v7: %v", err)
	}
	return id
}

func assertPgCode(t *testing.T, err error, code, what string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a constraint violation (%s: %s), got nil", code, what)
	}
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != code {
		t.Fatalf("error = %v, want pg code %s (%s)", err, code, what)
	}
}
