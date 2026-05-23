package storage

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestMemoryPutGetDelete(t *testing.T) {
	store := NewMemory("")

	err := store.Put(context.Background(), Object{
		Key:         "folder/file.txt",
		Reader:      strings.NewReader("hello world"),
		ContentType: "text/plain",
	})
	if err != nil {
		t.Fatal(err)
	}

	obj, err := store.Get(context.Background(), "folder/file.txt")
	if err != nil {
		t.Fatal(err)
	}
	if obj.Size != 11 {
		t.Fatalf("size = %d", obj.Size)
	}
	if obj.ContentType != "text/plain" {
		t.Fatalf("content type = %q", obj.ContentType)
	}

	if err := store.Delete(context.Background(), "folder/file.txt"); err != nil {
		t.Fatal(err)
	}

	_, err = store.Get(context.Background(), "folder/file.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound after delete, got %v", err)
	}
}

func TestMemoryGetMissingReturnsNotFound(t *testing.T) {
	store := NewMemory("")
	_, err := store.Get(context.Background(), "missing.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryStat(t *testing.T) {
	store := NewMemory("http://cdn.example.com")
	_ = store.Put(context.Background(), Object{
		Key:    "images/photo.jpg",
		Reader: strings.NewReader("jpeg-data"),
	})

	stat, err := store.Stat(context.Background(), "images/photo.jpg")
	if err != nil {
		t.Fatal(err)
	}
	if stat.Size != 9 {
		t.Fatalf("size = %d", stat.Size)
	}
	if stat.URL == "" {
		t.Error("URL should be set when publicURL is configured")
	}
}

func TestMemoryStatMissing(t *testing.T) {
	store := NewMemory("")
	_, err := store.Stat(context.Background(), "nope.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemorySignedURL(t *testing.T) {
	store := NewMemory("http://cdn.example.com")
	_ = store.Put(context.Background(), Object{Key: "doc.pdf", Reader: strings.NewReader("pdf")})

	u, err := store.SignedURL(context.Background(), "doc.pdf", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if u == "" {
		t.Error("signed URL should not be empty")
	}
}

func TestMemorySignedURLNoPublicURL(t *testing.T) {
	store := NewMemory("")
	_ = store.Put(context.Background(), Object{Key: "doc.pdf", Reader: strings.NewReader("pdf")})

	_, err := store.SignedURL(context.Background(), "doc.pdf", time.Hour)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("expected ErrUnsupported, got %v", err)
	}
}

func TestMemorySignedURLMissingKey(t *testing.T) {
	store := NewMemory("http://cdn.example.com")
	_, err := store.SignedURL(context.Background(), "missing.pdf", time.Hour)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMemoryPutRequiresKey(t *testing.T) {
	store := NewMemory("")
	err := store.Put(context.Background(), Object{Key: "", Reader: strings.NewReader("x")})
	if err == nil {
		t.Error("expected error for empty key")
	}
}

func TestMemoryLen(t *testing.T) {
	store := NewMemory("")
	_ = store.Put(context.Background(), Object{Key: "a", Reader: strings.NewReader("1")})
	_ = store.Put(context.Background(), Object{Key: "b", Reader: strings.NewReader("2")})
	if store.Len() != 2 {
		t.Fatalf("len = %d", store.Len())
	}
	_ = store.Delete(context.Background(), "a")
	if store.Len() != 1 {
		t.Fatalf("len after delete = %d", store.Len())
	}
}

func TestMemoryPrivateObjectURLOmitted(t *testing.T) {
	store := NewMemory("http://cdn.example.com")
	_ = store.Put(context.Background(), Object{
		Key:     "private/secret.txt",
		Reader:  strings.NewReader("secret"),
		Private: true,
	})
	stat, err := store.Stat(context.Background(), "private/secret.txt")
	if err != nil {
		t.Fatal(err)
	}
	if stat.URL != "" {
		t.Errorf("private object should not have a public URL, got %q", stat.URL)
	}
}
