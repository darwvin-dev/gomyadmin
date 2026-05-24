package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/cache"
)

// CacheSessionStore stores sessions in any cache.Cache implementation.
// Use it with Redis, Memcached, Dragonfly, KeyDB, or an existing app cache.
type CacheSessionStore struct {
	Cache  cache.Cache
	Prefix string
}

func NewCacheSessionStore(c cache.Cache) *CacheSessionStore {
	return &CacheSessionStore{Cache: c, Prefix: "gomyadmin:session:"}
}

func (s *CacheSessionStore) Create(ctx context.Context, actor admin.Actor, ttl time.Duration) (Session, error) {
	if s.Cache == nil {
		return Session{}, errors.New("auth.CacheSessionStore: Cache is required")
	}
	id, err := secureToken(32)
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	session := Session{ID: id, Actor: actor, CreatedAt: now, ExpiresAt: now.Add(ttl)}
	data, err := json.Marshal(session)
	if err != nil {
		return Session{}, err
	}
	return session, s.Cache.Set(ctx, s.key(id), data, ttl)
}

func (s *CacheSessionStore) Get(ctx context.Context, id string) (Session, error) {
	if s.Cache == nil {
		return Session{}, errors.New("auth.CacheSessionStore: Cache is required")
	}
	data, err := s.Cache.Get(ctx, s.key(id))
	if errors.Is(err, cache.ErrNotFound) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}
	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return Session{}, err
	}
	if time.Now().UTC().After(session.ExpiresAt) {
		_ = s.Delete(ctx, id)
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (s *CacheSessionStore) Delete(ctx context.Context, id string) error {
	if s.Cache == nil {
		return errors.New("auth.CacheSessionStore: Cache is required")
	}
	return s.Cache.Delete(ctx, s.key(id))
}

func (s *CacheSessionStore) key(id string) string {
	prefix := s.Prefix
	if prefix == "" {
		prefix = "gomyadmin:session:"
	}
	return prefix + id
}

var _ SessionStore = (*CacheSessionStore)(nil)
