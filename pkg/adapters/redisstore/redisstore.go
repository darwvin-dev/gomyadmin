// Package redisstore provides Redis-backed sessions for GoMyAdmin.
package redisstore

import (
	"context"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/redis/go-redis/v9"
)

type Cache struct {
	Client redis.UniversalClient
}

func New(client redis.UniversalClient) *auth.CacheSessionStore {
	return auth.NewCacheSessionStore(Cache{Client: client})
}

func (c Cache) Get(ctx context.Context, key string) ([]byte, error) {
	value, err := c.Client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, auth.ErrSessionNotFound
	}
	return value, err
}

func (c Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return c.Client.Set(ctx, key, value, ttl).Err()
}

func (c Cache) Delete(ctx context.Context, key string) error {
	return c.Client.Del(ctx, key).Err()
}
