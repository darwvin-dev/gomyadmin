package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Memory is a thread-safe in-memory storage adapter, primarily for tests.
type Memory struct {
	mu        sync.RWMutex
	objects   map[string]memObject
	publicURL string
}

type memObject struct {
	data        []byte
	contentType string
	private     bool
	metadata    map[string]string
	createdAt   time.Time
}

// NewMemory creates a new in-memory storage backend.
// publicURL is an optional base URL used to build object URLs (can be empty).
func NewMemory(publicURL string) *Memory {
	return &Memory{
		objects:   map[string]memObject{},
		publicURL: strings.TrimRight(publicURL, "/"),
	}
}

func (m *Memory) Put(_ context.Context, object Object) error {
	if object.Key == "" {
		return errors.New("object key is required")
	}
	data, err := io.ReadAll(object.Reader)
	if err != nil {
		return err
	}
	meta := make(map[string]string, len(object.Metadata))
	for k, v := range object.Metadata {
		meta[k] = v
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[object.Key] = memObject{
		data:        data,
		contentType: object.ContentType,
		private:     object.Private,
		metadata:    meta,
		createdAt:   time.Now().UTC(),
	}
	return nil
}

func (m *Memory) Get(_ context.Context, key string) (Object, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[key]
	if !ok {
		return Object{}, ErrNotFound
	}
	return Object{
		Key:         key,
		Reader:      bytes.NewReader(obj.data),
		ContentType: obj.contentType,
		Size:        int64(len(obj.data)),
		Private:     obj.private,
		Metadata:    obj.metadata,
	}, nil
}

func (m *Memory) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

func (m *Memory) SignedURL(_ context.Context, key string, ttl time.Duration) (string, error) {
	m.mu.RLock()
	_, ok := m.objects[key]
	m.mu.RUnlock()
	if !ok {
		return "", ErrNotFound
	}
	if m.publicURL == "" {
		return "", ErrUnsupported
	}
	expires := time.Now().UTC().Add(ttl).Unix()
	return m.publicURL + "/" + url.PathEscape(key) + "?expires=" + url.QueryEscape(time.Unix(expires, 0).Format(time.RFC3339)), nil
}

func (m *Memory) Stat(_ context.Context, key string) (StoredObject, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	obj, ok := m.objects[key]
	if !ok {
		return StoredObject{}, ErrNotFound
	}
	stored := StoredObject{
		Key:         key,
		ContentType: obj.contentType,
		Size:        int64(len(obj.data)),
		Private:     obj.private,
		CreatedAt:   obj.createdAt,
	}
	if m.publicURL != "" && !obj.private {
		stored.URL = m.publicURL + "/" + url.PathEscape(key)
	}
	return stored, nil
}

// Len returns the number of stored objects.
func (m *Memory) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.objects)
}
