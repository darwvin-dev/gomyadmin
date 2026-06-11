package audit

import (
	"context"
	"testing"
	"time"
)

func TestMemoryStoreFiltersTenant(t *testing.T) {
	store := NewMemoryStore()
	_ = store.Record(context.Background(), Event{TenantID: "a", Action: "login"})
	_ = store.Record(context.Background(), Event{TenantID: "b", Action: "login"})
	events, err := store.List(context.Background(), Query{TenantID: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0].TenantID != "a" {
		t.Fatalf("events = %#v", events)
	}
}

func TestMemoryStoreDefaultsCreatedAt(t *testing.T) {
	store := NewMemoryStore()
	if err := store.Record(context.Background(), Event{ID: "event-1"}); err != nil {
		t.Fatal(err)
	}
	events, err := store.List(context.Background(), Query{})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("events len = %d", len(events))
	}
	if events[0].CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be defaulted")
	}
}

func TestMemoryStoreListsNewestFirstAndAppliesLimit(t *testing.T) {
	store := NewMemoryStore()
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	for _, event := range []Event{
		{ID: "old", CreatedAt: base},
		{ID: "middle", CreatedAt: base.Add(time.Minute)},
		{ID: "new", CreatedAt: base.Add(2 * time.Minute)},
	} {
		if err := store.Record(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.List(context.Background(), Query{Limit: 2})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 2 {
		t.Fatalf("events len = %d", len(events))
	}
	if events[0].ID != "new" || events[1].ID != "middle" {
		t.Fatalf("events order = %#v", events)
	}
}

func TestMemoryStoreFiltersAllQueryFields(t *testing.T) {
	store := NewMemoryStore()
	base := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	events := []Event{
		{ID: "skip-actor", ActorID: "other", TenantID: "t1", Resource: "users", Action: "create", CreatedAt: base.Add(time.Minute)},
		{ID: "skip-tenant", ActorID: "a1", TenantID: "other", Resource: "users", Action: "create", CreatedAt: base.Add(2 * time.Minute)},
		{ID: "skip-resource", ActorID: "a1", TenantID: "t1", Resource: "orders", Action: "create", CreatedAt: base.Add(3 * time.Minute)},
		{ID: "skip-action", ActorID: "a1", TenantID: "t1", Resource: "users", Action: "delete", CreatedAt: base.Add(4 * time.Minute)},
		{ID: "skip-before", ActorID: "a1", TenantID: "t1", Resource: "users", Action: "create", CreatedAt: base.Add(-time.Minute)},
		{ID: "skip-after", ActorID: "a1", TenantID: "t1", Resource: "users", Action: "create", CreatedAt: base.Add(10 * time.Minute)},
		{ID: "match", ActorID: "a1", TenantID: "t1", Resource: "users", Action: "create", CreatedAt: base.Add(5 * time.Minute)},
	}
	for _, event := range events {
		if err := store.Record(context.Background(), event); err != nil {
			t.Fatal(err)
		}
	}
	got, err := store.List(context.Background(), Query{
		ActorID:  "a1",
		TenantID: "t1",
		Resource: "users",
		Action:   "create",
		From:     base,
		To:       base.Add(6 * time.Minute),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "match" {
		t.Fatalf("events = %#v", got)
	}
}

func TestMemoryStoreCapsInvalidLimit(t *testing.T) {
	store := NewMemoryStore()
	for i := 0; i < 205; i++ {
		if err := store.Record(context.Background(), Event{}); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.List(context.Background(), Query{Limit: 500})
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 100 {
		t.Fatalf("events len = %d", len(events))
	}
}
