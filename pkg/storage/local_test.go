package storage

import (
	"context"
	"strings"
	"testing"
)

func TestLocalRejectsPathTraversal(t *testing.T) {
	store := NewLocal(t.TempDir(), "")
	err := store.Put(context.Background(), Object{Key: "../secret", Reader: strings.NewReader("x")})
	if err == nil {
		t.Fatal("expected path traversal error")
	}
}

func TestLocalPutGetDelete(t *testing.T) {
	store := NewLocal(t.TempDir(), "")
	if err := store.Put(context.Background(), Object{Key: "tenant/file.txt", Reader: strings.NewReader("hello")}); err != nil {
		t.Fatal(err)
	}
	object, err := store.Get(context.Background(), "tenant/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if object.Size != 5 {
		t.Fatalf("size = %d", object.Size)
	}
	if closer, ok := object.Reader.(interface{ Close() error }); ok {
		if err := closer.Close(); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.Delete(context.Background(), "tenant/file.txt"); err != nil {
		t.Fatal(err)
	}
}
