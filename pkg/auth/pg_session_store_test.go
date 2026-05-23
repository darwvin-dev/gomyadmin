package auth

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/darwvin/gomyadmin/pkg/admin"
	"github.com/jackc/pgx/v5/pgxpool"
)

func pgPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set; skipping PostgreSQL integration test")
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func TestPGSessionStoreCreateGetDelete(t *testing.T) {
	pool := pgPool(t)
	store := NewPGSessionStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// clean up after the test
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM gomyadmin_sessions WHERE actor->>'id' = 'pg-test-user'")
	})

	actor := admin.Actor{
		ID:    "pg-test-user",
		Email: "pg@example.com",
		Roles: []string{"admin"},
	}

	session, err := store.Create(ctx, actor, time.Hour)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if session.ID == "" {
		t.Fatal("session ID should not be empty")
	}

	got, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Actor.ID != "pg-test-user" {
		t.Errorf("actor ID = %q", got.Actor.ID)
	}
	if got.Actor.Email != "pg@example.com" {
		t.Errorf("actor email = %q", got.Actor.Email)
	}
	if len(got.Actor.Roles) != 1 || got.Actor.Roles[0] != "admin" {
		t.Errorf("actor roles = %v", got.Actor.Roles)
	}

	if err := store.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err = store.Get(ctx, session.ID)
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestPGSessionStoreExpiredSession(t *testing.T) {
	pool := pgPool(t)
	store := NewPGSessionStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM gomyadmin_sessions WHERE expires_at < NOW()")
	})

	session, err := store.Create(ctx, admin.Actor{ID: "expired-user"}, -time.Second)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	_, err = store.Get(ctx, session.ID)
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound for expired session, got %v", err)
	}
}

func TestPGSessionStoreCleanup(t *testing.T) {
	pool := pgPool(t)
	store := NewPGSessionStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM gomyadmin_sessions WHERE expires_at < NOW()")
	})

	// Insert two expired sessions
	_, _ = store.Create(ctx, admin.Actor{ID: "cleanup-a"}, -time.Second)
	_, _ = store.Create(ctx, admin.Actor{ID: "cleanup-b"}, -time.Second)

	deleted, err := store.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup: %v", err)
	}
	if deleted < 2 {
		t.Errorf("Cleanup deleted %d rows, want >= 2", deleted)
	}
}

func TestPGSessionStoreMissingSession(t *testing.T) {
	pool := pgPool(t)
	store := NewPGSessionStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	_, err := store.Get(ctx, "session-that-does-not-exist")
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}
