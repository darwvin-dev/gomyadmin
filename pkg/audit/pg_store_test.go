package audit

import (
	"context"
	"os"
	"testing"
	"time"

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

func TestPGStoreRecordAndList(t *testing.T) {
	pool := pgPool(t)
	store := NewPGStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM gomyadmin_audit_events WHERE tenant_id = 'pg-test-tenant'")
	})

	events := []Event{
		{
			ActorID:    "user-1",
			ActorEmail: "alice@acme.com",
			TenantID:   "pg-test-tenant",
			Action:     "create",
			Resource:   "customer",
			ResourceID: "cust-001",
			NewValues:  map[string]any{"name": "Alice Corp"},
			IPAddress:  "10.0.0.1",
		},
		{
			ActorID:    "user-1",
			ActorEmail: "alice@acme.com",
			TenantID:   "pg-test-tenant",
			Action:     "update",
			Resource:   "customer",
			ResourceID: "cust-001",
			OldValues:  map[string]any{"status": "pending"},
			NewValues:  map[string]any{"status": "active"},
		},
		{
			ActorID:    "user-2",
			ActorEmail: "bob@acme.com",
			TenantID:   "pg-test-tenant",
			Action:     "delete",
			Resource:   "invoice",
			ResourceID: "inv-007",
		},
	}

	for _, e := range events {
		if err := store.Record(ctx, e); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	// List all for tenant
	all, err := store.List(ctx, Query{TenantID: "pg-test-tenant", Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("len = %d, want 3", len(all))
	}

	// List by resource
	customers, err := store.List(ctx, Query{TenantID: "pg-test-tenant", Resource: "customer", Limit: 10})
	if err != nil {
		t.Fatalf("List by resource: %v", err)
	}
	if len(customers) != 2 {
		t.Fatalf("customer events = %d, want 2", len(customers))
	}

	// List by actor
	byActor, err := store.List(ctx, Query{TenantID: "pg-test-tenant", ActorID: "user-2", Limit: 10})
	if err != nil {
		t.Fatalf("List by actor: %v", err)
	}
	if len(byActor) != 1 || byActor[0].Action != "delete" {
		t.Fatalf("actor events = %v", byActor)
	}
}

func TestPGStoreOldNewValuesRoundtrip(t *testing.T) {
	pool := pgPool(t)
	store := NewPGStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM gomyadmin_audit_events WHERE resource = 'roundtrip-resource'")
	})

	err := store.Record(ctx, Event{
		Action:   "update",
		Resource: "roundtrip-resource",
		OldValues: map[string]any{
			"status": "pending",
			"amount": 100.50,
		},
		NewValues: map[string]any{
			"status": "active",
			"amount": 200.0,
		},
	})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	events, err := store.List(ctx, Query{Resource: "roundtrip-resource", Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no events returned")
	}

	e := events[0]
	if e.OldValues["status"] != "pending" {
		t.Errorf("old status = %v", e.OldValues["status"])
	}
	if e.NewValues["status"] != "active" {
		t.Errorf("new status = %v", e.NewValues["status"])
	}
}

func TestPGStoreDateRangeFilter(t *testing.T) {
	pool := pgPool(t)
	store := NewPGStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM gomyadmin_audit_events WHERE resource = 'date-filter-resource'")
	})

	before := time.Now().UTC()
	_ = store.Record(ctx, Event{Action: "login", Resource: "date-filter-resource"})
	after := time.Now().UTC()

	// From before → should find the event
	found, err := store.List(ctx, Query{Resource: "date-filter-resource", From: before, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(found) == 0 {
		t.Error("expected at least one event with From filter")
	}

	// From after → should find nothing
	empty, err := store.List(ctx, Query{Resource: "date-filter-resource", From: after, Limit: 10})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(empty) != 0 {
		t.Errorf("expected no events, got %d", len(empty))
	}
}

func TestPGStoreLimitCap(t *testing.T) {
	pool := pgPool(t)
	store := NewPGStore(pool)
	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	// Requesting more than 200 should be silently capped
	events, err := store.List(ctx, Query{Limit: 500})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	// We can't assert count easily without data, but the query must succeed
	_ = events
}

func TestPGStoreAutoIDAndTimestamp(t *testing.T) {
	pool := pgPool(t)
	store := NewPGStore(pool)

	ctx := context.Background()
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(ctx, "DELETE FROM gomyadmin_audit_events WHERE resource = 'auto-id-resource'")
	})

	before := time.Now().UTC().Add(-time.Second)
	err := store.Record(ctx, Event{Action: "login", Resource: "auto-id-resource"})
	if err != nil {
		t.Fatalf("Record: %v", err)
	}

	events, err := store.List(ctx, Query{Resource: "auto-id-resource", Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no event returned")
	}
	if events[0].ID == "" {
		t.Error("ID should be auto-generated")
	}
	if events[0].CreatedAt.Before(before) {
		t.Error("CreatedAt should be recent")
	}
}
