package audit

import (
	"context"
	"testing"
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
