package storage

import (
	"context"
	"errors"
	"io"
	"time"
)

var (
	ErrUnsupported = errors.New("storage operation is not supported by this adapter")
	ErrNotFound    = errors.New("object not found")
)

type Object struct {
	Key         string
	Reader      io.Reader
	ContentType string
	Size        int64
	Private     bool
	Metadata    map[string]string
}

type StoredObject struct {
	Key         string    `json:"key"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Private     bool      `json:"private"`
	URL         string    `json:"url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type Storage interface {
	Put(ctx context.Context, object Object) error
	Get(ctx context.Context, key string) (Object, error)
	Delete(ctx context.Context, key string) error
	SignedURL(ctx context.Context, key string, ttl time.Duration) (string, error)
}

type Inspector interface {
	Stat(ctx context.Context, key string) (StoredObject, error)
}

type S3Config struct {
	Endpoint        string
	Region          string
	Bucket          string
	AccessKeyID     string
	SecretAccessKey string
	PublicBaseURL   string
	ForcePathStyle  bool
}
