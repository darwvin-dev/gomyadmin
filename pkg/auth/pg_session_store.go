package auth

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// SessionTableDDL is the CREATE TABLE statement for the sessions table.
// Call PGSessionStore.Migrate to execute it.
const SessionTableDDL = `
CREATE TABLE IF NOT EXISTS gomyadmin_sessions (
    id         TEXT        PRIMARY KEY,
    actor      JSONB       NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS gomyadmin_sessions_expires_at ON gomyadmin_sessions (expires_at);`

// PGSessionStore is a PostgreSQL-backed SessionStore.
// Use NewPGSessionStore to construct it, then call Migrate once at startup.
type PGSessionStore struct {
	pool *pgxpool.Pool
}

// NewPGSessionStore creates a PGSessionStore backed by the supplied pool.
func NewPGSessionStore(pool *pgxpool.Pool) *PGSessionStore {
	return &PGSessionStore{pool: pool}
}

// Migrate creates the sessions table and index if they do not exist.
// Call this once at application startup before handling any requests.
func (s *PGSessionStore) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, SessionTableDDL)
	return err
}

// Create inserts a new session for actor with the given TTL and returns it.
func (s *PGSessionStore) Create(ctx context.Context, actor admin.Actor, ttl time.Duration) (Session, error) {
	id, err := secureToken(32)
	if err != nil {
		return Session{}, err
	}
	actorJSON, err := json.Marshal(actor)
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)

	_, err = s.pool.Exec(ctx,
		`INSERT INTO gomyadmin_sessions (id, actor, expires_at, created_at)
		 VALUES ($1, $2, $3, $4)`,
		id, actorJSON, expiresAt, now,
	)
	if err != nil {
		return Session{}, err
	}
	return Session{
		ID:        id,
		Actor:     actor,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// Get retrieves a non-expired session by ID. Returns ErrSessionNotFound when
// the session does not exist or has expired.
func (s *PGSessionStore) Get(ctx context.Context, id string) (Session, error) {
	var actorJSON []byte
	var session Session
	session.ID = id

	err := s.pool.QueryRow(ctx,
		`SELECT actor, expires_at, created_at
		 FROM gomyadmin_sessions
		 WHERE id = $1 AND expires_at > NOW()`,
		id,
	).Scan(&actorJSON, &session.ExpiresAt, &session.CreatedAt)

	if errors.Is(err, pgx.ErrNoRows) {
		return Session{}, ErrSessionNotFound
	}
	if err != nil {
		return Session{}, err
	}
	if err := json.Unmarshal(actorJSON, &session.Actor); err != nil {
		return Session{}, err
	}
	return session, nil
}

// Delete removes a session. It is not an error if the session does not exist.
func (s *PGSessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx,
		`DELETE FROM gomyadmin_sessions WHERE id = $1`, id)
	return err
}

// Cleanup deletes all expired sessions and returns the number of rows removed.
// Schedule this periodically (e.g. every hour) to keep the table small.
func (s *PGSessionStore) Cleanup(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM gomyadmin_sessions WHERE expires_at <= NOW()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
