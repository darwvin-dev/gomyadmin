// Package cache defines a tiny adapter boundary for optional caching layers.
package cache

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("cache: key not found")

// Cache is the minimal contract GoMyAdmin adapters need from Redis,
// Memcached, in-memory caches, or an existing application cache.
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

type item struct {
	value     []byte
	expiresAt time.Time
}

// MemoryCache is a small concurrency-safe cache for tests and single-process
// development. Production deployments should prefer Redis, Memcached, or a
// shared application cache.
type MemoryCache struct {
	mu    sync.RWMutex
	items map[string]item
}

func NewMemory() *MemoryCache {
	return &MemoryCache{items: map[string]item{}}
}

func (c *MemoryCache) Get(_ context.Context, key string) ([]byte, error) {
	c.mu.RLock()
	it, ok := c.items[key]
	c.mu.RUnlock()
	if !ok || (!it.expiresAt.IsZero() && time.Now().UTC().After(it.expiresAt)) {
		if ok {
			_ = c.Delete(context.Background(), key)
		}
		return nil, ErrNotFound
	}
	out := make([]byte, len(it.value))
	copy(out, it.value)
	return out, nil
}

func (c *MemoryCache) Set(_ context.Context, key string, value []byte, ttl time.Duration) error {
	expiresAt := time.Time{}
	if ttl > 0 {
		expiresAt = time.Now().UTC().Add(ttl)
	}
	copied := make([]byte, len(value))
	copy(copied, value)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = item{value: copied, expiresAt: expiresAt}
	return nil
}

func (c *MemoryCache) Delete(_ context.Context, key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
	return nil
}

var _ Cache = (*MemoryCache)(nil)
