package audit

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// AuditTableDDL is the CREATE TABLE statement for the audit events table.
// Call PGStore.Migrate to execute it.
const AuditTableDDL = `
CREATE TABLE IF NOT EXISTS gomyadmin_audit_events (
    id          TEXT        PRIMARY KEY,
    actor_id    TEXT,
    actor_email TEXT,
    tenant_id   TEXT,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    resource_id TEXT,
    old_values  JSONB,
    new_values  JSONB,
    metadata    JSONB,
    ip_address  TEXT,
    user_agent  TEXT,
    request_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS gomyadmin_audit_events_tenant  ON gomyadmin_audit_events (tenant_id);
CREATE INDEX IF NOT EXISTS gomyadmin_audit_events_actor   ON gomyadmin_audit_events (actor_id);
CREATE INDEX IF NOT EXISTS gomyadmin_audit_events_resource ON gomyadmin_audit_events (resource);
CREATE INDEX IF NOT EXISTS gomyadmin_audit_events_created ON gomyadmin_audit_events (created_at DESC);`

// PGStore is a PostgreSQL-backed audit Store.
// Use NewPGStore to construct it, then call Migrate once at startup.
type PGStore struct {
	pool *pgxpool.Pool
}

// NewPGStore creates a PGStore backed by the supplied connection pool.
func NewPGStore(pool *pgxpool.Pool) *PGStore {
	return &PGStore{pool: pool}
}

// Migrate creates the audit events table and indexes if they do not exist.
// Call this once at application startup.
func (s *PGStore) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, AuditTableDDL)
	return err
}

// Record inserts a new audit event. A random ID and current timestamp are
// generated automatically when the event does not provide them.
func (s *PGStore) Record(ctx context.Context, event Event) error {
	if event.ID == "" {
		event.ID = newID()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	oldJSON, err := json.Marshal(event.OldValues)
	if err != nil {
		return err
	}
	newJSON, err := json.Marshal(event.NewValues)
	if err != nil {
		return err
	}
	metaJSON, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}

	_, err = s.pool.Exec(ctx, `
		INSERT INTO gomyadmin_audit_events
		    (id, actor_id, actor_email, tenant_id, action, resource, resource_id,
		     old_values, new_values, metadata, ip_address, user_agent, request_id, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		event.ID, event.ActorID, event.ActorEmail, event.TenantID,
		event.Action, event.Resource, event.ResourceID,
		oldJSON, newJSON, metaJSON,
		event.IPAddress, event.UserAgent, event.RequestID, event.CreatedAt,
	)
	return err
}

// List returns audit events matching the query in descending creation order.
// Limit defaults to 100 and is capped at 200.
func (s *PGStore) List(ctx context.Context, query Query) ([]Event, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	if limit > 200 {
		limit = 200
	}

	clauses := []string{}
	args := []any{}

	if query.ActorID != "" {
		args = append(args, query.ActorID)
		clauses = append(clauses, fmt.Sprintf("actor_id = $%d", len(args)))
	}
	if query.TenantID != "" {
		args = append(args, query.TenantID)
		clauses = append(clauses, fmt.Sprintf("tenant_id = $%d", len(args)))
	}
	if query.Resource != "" {
		args = append(args, query.Resource)
		clauses = append(clauses, fmt.Sprintf("resource = $%d", len(args)))
	}
	if query.Action != "" {
		args = append(args, query.Action)
		clauses = append(clauses, fmt.Sprintf("action = $%d", len(args)))
	}
	if !query.From.IsZero() {
		args = append(args, query.From.UTC())
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if !query.To.IsZero() {
		args = append(args, query.To.UTC())
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", len(args)))
	}

	sql := "SELECT id, actor_id, actor_email, tenant_id, action, resource, resource_id, old_values, new_values, metadata, ip_address, user_agent, request_id, created_at FROM gomyadmin_audit_events"
	if len(clauses) > 0 {
		sql += " WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)
	sql += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var oldJSON, newJSON, metaJSON []byte
		if err := rows.Scan(
			&e.ID, &e.ActorID, &e.ActorEmail, &e.TenantID,
			&e.Action, &e.Resource, &e.ResourceID,
			&oldJSON, &newJSON, &metaJSON,
			&e.IPAddress, &e.UserAgent, &e.RequestID, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		if len(oldJSON) > 0 && string(oldJSON) != "null" {
			_ = json.Unmarshal(oldJSON, &e.OldValues)
		}
		if len(newJSON) > 0 && string(newJSON) != "null" {
			_ = json.Unmarshal(newJSON, &e.NewValues)
		}
		if len(metaJSON) > 0 && string(metaJSON) != "null" {
			_ = json.Unmarshal(metaJSON, &e.Metadata)
		}
		events = append(events, e)
	}
	return events, rows.Err()
}

func newID() string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
