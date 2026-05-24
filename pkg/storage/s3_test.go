package storage

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func newTestS3Config(endpoint string) S3Config {
	return S3Config{
		Endpoint:        endpoint,
		Region:          "us-east-1",
		Bucket:          "test-bucket",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		ForcePathStyle:  true,
	}
}

func TestS3NewRequiresBucket(t *testing.T) {
	_, err := NewS3(S3Config{AccessKeyID: "k", SecretAccessKey: "s"})
	if err == nil || !strings.Contains(err.Error(), "Bucket") {
		t.Fatalf("expected bucket error, got %v", err)
	}
}

func TestS3NewRequiresCredentials(t *testing.T) {
	_, err := NewS3(S3Config{Bucket: "b"})
	if err == nil || !strings.Contains(err.Error(), "AccessKeyID") {
		t.Fatalf("expected credentials error, got %v", err)
	}
}

func TestS3NewDefaultRegion(t *testing.T) {
	store, err := NewS3(S3Config{Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s"})
	if err != nil {
		t.Fatal(err)
	}
	if store.config.Region != "us-east-1" {
		t.Errorf("region = %q, want us-east-1", store.config.Region)
	}
}

func TestS3Put(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	err := store.Put(context.Background(), Object{
		Key:         "uploads/photo.jpg",
		Reader:      strings.NewReader("image-data"),
		ContentType: "image/jpeg",
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	if gotPath != "/test-bucket/uploads/photo.jpg" {
		t.Errorf("path = %q", gotPath)
	}
	if gotBody != "image-data" {
		t.Errorf("body = %q", gotBody)
	}
}

func TestS3PutSetsAuthorizationHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	_ = store.Put(context.Background(), Object{Key: "file.txt", Reader: strings.NewReader("x")})

	if !strings.HasPrefix(gotAuth, "AWS4-HMAC-SHA256") {
		t.Errorf("Authorization header = %q, want AWS4-HMAC-SHA256 prefix", gotAuth)
	}
}

func TestS3Get(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello from s3"))
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	obj, err := store.Get(context.Background(), "docs/readme.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if c, ok := obj.Reader.(io.Closer); ok {
			_ = c.Close()
		}
	}()

	body, _ := io.ReadAll(obj.Reader)
	if string(body) != "hello from s3" {
		t.Errorf("body = %q", string(body))
	}
	if obj.ContentType != "text/plain" {
		t.Errorf("content type = %q", obj.ContentType)
	}
}

func TestS3GetNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	_, err := store.Get(context.Background(), "missing.txt")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestS3Delete(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	if err := store.Delete(context.Background(), "old/file.txt"); err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
}

func TestS3DeleteNotFoundIsOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	if err := store.Delete(context.Background(), "gone.txt"); err != nil {
		t.Fatalf("delete of missing key should not error, got %v", err)
	}
}

func TestS3Stat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", "1024")
		w.Header().Set("Last-Modified", "Mon, 01 Jan 2024 00:00:00 GMT")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	stored, err := store.Stat(context.Background(), "img/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if stored.ContentType != "image/png" {
		t.Errorf("content type = %q", stored.ContentType)
	}
	if stored.CreatedAt.IsZero() {
		t.Error("CreatedAt should be populated from Last-Modified")
	}
}

func TestS3StatNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	_, err := store.Stat(context.Background(), "missing.png")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestS3StatSetsPublicURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := newTestS3Config(srv.URL)
	cfg.PublicBaseURL = "https://cdn.example.com"
	store, _ := NewS3(cfg)
	stored, err := store.Stat(context.Background(), "assets/logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if stored.URL != "https://cdn.example.com/assets/logo.png" {
		t.Errorf("URL = %q", stored.URL)
	}
}

func TestS3SignedURL(t *testing.T) {
	store, _ := NewS3(S3Config{
		Bucket:          "my-bucket",
		Region:          "us-east-1",
		AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
		SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		ForcePathStyle:  true,
		Endpoint:        "https://s3.us-east-1.amazonaws.com",
	})

	u, err := store.SignedURL(context.Background(), "private/report.pdf", 15*time.Minute)
	if err != nil {
		t.Fatal(err)
	}

	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	for _, param := range []string{"X-Amz-Algorithm", "X-Amz-Credential", "X-Amz-Date", "X-Amz-Expires", "X-Amz-SignedHeaders", "X-Amz-Signature"} {
		if q.Get(param) == "" {
			t.Errorf("missing query param %q in signed URL", param)
		}
	}
	if exp := q.Get("X-Amz-Expires"); exp != "900" {
		t.Errorf("X-Amz-Expires = %q, want 900", exp)
	}
}

func TestS3ErrorResponseXML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`<Error><Code>AccessDenied</Code><Message>Access Denied</Message></Error>`))
	}))
	defer srv.Close()

	store, _ := NewS3(newTestS3Config(srv.URL))
	err := store.Put(context.Background(), Object{Key: "x", Reader: strings.NewReader("")})
	if err == nil || !strings.Contains(err.Error(), "AccessDenied") {
		t.Fatalf("expected AccessDenied error, got %v", err)
	}
}

func TestS3ObjectURLPathStyle(t *testing.T) {
	store, _ := NewS3(S3Config{
		Bucket:          "my-bucket",
		Region:          "eu-west-1",
		AccessKeyID:     "k",
		SecretAccessKey: "s",
		Endpoint:        "https://custom.storage.example.com",
	})
	u := store.objectURL("folder/file.txt")
	if u != "https://custom.storage.example.com/my-bucket/folder/file.txt" {
		t.Errorf("objectURL = %q", u)
	}
}

func TestS3ObjectURLVirtualHosted(t *testing.T) {
	store, _ := NewS3(S3Config{
		Bucket:          "my-bucket",
		Region:          "us-west-2",
		AccessKeyID:     "k",
		SecretAccessKey: "s",
	})
	u := store.objectURL("folder/file.txt")
	if u != "https://my-bucket.s3.us-west-2.amazonaws.com/folder/file.txt" {
		t.Errorf("objectURL = %q", u)
	}
}

func TestSortedQueryString(t *testing.T) {
	q := url.Values{
		"Z-Param": {"z"},
		"A-Param": {"a"},
		"M-Param": {"m"},
	}
	result := sortedQueryString(q)
	parts := strings.Split(result, "&")
	if len(parts) != 3 {
		t.Fatalf("expected 3 parts, got %d: %s", len(parts), result)
	}
	if !strings.HasPrefix(parts[0], "A-Param") {
		t.Errorf("first param should be A-Param, got %q", parts[0])
	}
	if !strings.HasPrefix(parts[1], "M-Param") {
		t.Errorf("second param should be M-Param, got %q", parts[1])
	}
	if !strings.HasPrefix(parts[2], "Z-Param") {
		t.Errorf("third param should be Z-Param, got %q", parts[2])
	}
}
