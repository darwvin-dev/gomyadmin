package redisstore

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/redis/go-redis/v9"
)

func startRedis(t *testing.T) *redis.Client {
	t.Helper()
	s := miniredis.RunT(t)
	return redis.NewClient(&redis.Options{Addr: s.Addr()})
}

func TestNewReturnsSessionStore(t *testing.T) {
	client := startRedis(t)
	store := New(client)
	if store == nil {
		t.Fatal("expected non-nil SessionStore")
	}
}

func TestCacheSetAndGet(t *testing.T) {
	client := startRedis(t)
	c := Cache{Client: client}
	ctx := context.Background()

	if err := c.Set(ctx, "key1", []byte("hello"), time.Minute); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := c.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("got %q, want %q", string(got), "hello")
	}
}

func TestCacheGetMissingKeyReturnsErrSessionNotFound(t *testing.T) {
	client := startRedis(t)
	c := Cache{Client: client}

	_, err := c.Get(context.Background(), "does-not-exist")
	if err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestCacheDelete(t *testing.T) {
	client := startRedis(t)
	c := Cache{Client: client}
	ctx := context.Background()

	if err := c.Set(ctx, "k", []byte("v"), time.Minute); err != nil {
		t.Fatal(err)
	}
	if err := c.Delete(ctx, "k"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	_, err := c.Get(ctx, "k")
	if err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestCacheExpiry(t *testing.T) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	c := Cache{Client: client}
	ctx := context.Background()

	if err := c.Set(ctx, "exp", []byte("val"), time.Second); err != nil {
		t.Fatal(err)
	}
	mr.FastForward(2 * time.Second)

	_, err := c.Get(ctx, "exp")
	if err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after expiry, got %v", err)
	}
}

func TestSessionRoundTrip(t *testing.T) {
	client := startRedis(t)
	store := New(client)
	ctx := context.Background()

	session, err := store.Create(ctx, admin.Actor{ID: "u1", Email: "test@example.com"}, time.Minute)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, err := store.Get(ctx, session.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Actor.Email != "test@example.com" {
		t.Fatalf("email = %q", got.Actor.Email)
	}
}

func TestSessionDeleteRemovesSession(t *testing.T) {
	client := startRedis(t)
	store := New(client)
	ctx := context.Background()

	session, err := store.Create(ctx, admin.Actor{ID: "u2", Email: "del@example.com"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Get(ctx, session.ID); err != auth.ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}
