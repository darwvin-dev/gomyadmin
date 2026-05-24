package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryCacheCopiesValues(t *testing.T) {
	c := NewMemory()
	value := []byte("secret")
	if err := c.Set(context.Background(), "k", value, time.Minute); err != nil {
		t.Fatal(err)
	}
	value[0] = 'x'

	got, err := c.Get(context.Background(), "k")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "secret" {
		t.Fatalf("got %q", got)
	}
	got[0] = 'x'
	again, _ := c.Get(context.Background(), "k")
	if string(again) != "secret" {
		t.Fatalf("cache exposed internal slice: %q", again)
	}
}

func TestMemoryCacheExpires(t *testing.T) {
	c := NewMemory()
	if err := c.Set(context.Background(), "k", []byte("v"), time.Nanosecond); err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)

	_, err := c.Get(context.Background(), "k")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v", err)
	}
}
