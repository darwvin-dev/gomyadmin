package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

var ErrAPIKeyNotFound = errors.New("api key not found")

type APIKey struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Prefix     string      `json:"prefix"`
	Scopes     []string    `json:"scopes"`
	Actor      admin.Actor `json:"actor"`
	ExpiresAt  *time.Time  `json:"expires_at,omitempty"`
	LastUsedAt *time.Time  `json:"last_used_at,omitempty"`
	RevokedAt  *time.Time  `json:"revoked_at,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
}

type CreateAPIKeyInput struct {
	Name      string
	Actor     admin.Actor
	Scopes    []string
	ExpiresIn time.Duration
}

type APIKeyManager interface {
	Authenticate(ctx context.Context, rawKey string) (admin.Actor, APIKey, bool, error)
	Create(ctx context.Context, input CreateAPIKeyInput) (APIKey, string, error)
	List(ctx context.Context, actor admin.Actor) ([]APIKey, error)
	Revoke(ctx context.Context, id string, actor admin.Actor) error
}

const apiKeySchema = `
CREATE TABLE IF NOT EXISTS gomyadmin_api_keys (
    id            TEXT PRIMARY KEY,
    name          TEXT NOT NULL,
    prefix        TEXT NOT NULL UNIQUE,
    secret_hash   TEXT NOT NULL,
    actor_id      TEXT NOT NULL,
    actor_email   TEXT NOT NULL,
    actor_name    TEXT NOT NULL DEFAULT '',
    tenant_id     TEXT,
    roles         JSONB NOT NULL DEFAULT '[]'::jsonb,
    permissions   JSONB NOT NULL DEFAULT '[]'::jsonb,
    scopes        JSONB NOT NULL DEFAULT '[]'::jsonb,
    expires_at    TIMESTAMPTZ,
    last_used_at  TIMESTAMPTZ,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS gomyadmin_api_keys_actor_tenant_idx
    ON gomyadmin_api_keys (actor_id, tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS gomyadmin_api_keys_prefix_idx
    ON gomyadmin_api_keys (prefix);
`

type PGAPIKeys struct {
	pool *pgxpool.Pool
}

func NewPGAPIKeys(pool *pgxpool.Pool) *PGAPIKeys {
	return &PGAPIKeys{pool: pool}
}

func (s *PGAPIKeys) Migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, apiKeySchema)
	return err
}

func (s *PGAPIKeys) Create(ctx context.Context, input CreateAPIKeyInput) (APIKey, string, error) {
	if strings.TrimSpace(input.Name) == "" {
		return APIKey{}, "", errors.New("api key name is required")
	}
	if input.Actor.ID == "" {
		return APIKey{}, "", errors.New("actor id is required")
	}
	now := time.Now().UTC()
	id := "key_" + randomHex(8)
	prefix := randomHex(6)
	secret := randomHex(18)
	raw := "gma_" + prefix + "_" + secret
	hash := hashAPIKey(raw)
	scopes := cloneStrings(input.Scopes)
	if len(scopes) == 0 {
		scopes = cloneStrings(input.Actor.Permissions)
	}
	permissionsJSON, err := json.Marshal(scopes)
	if err != nil {
		return APIKey{}, "", err
	}
	rolesJSON, err := json.Marshal(input.Actor.Roles)
	if err != nil {
		return APIKey{}, "", err
	}
	scopesJSON, err := json.Marshal(scopes)
	if err != nil {
		return APIKey{}, "", err
	}
	var expiresAt *time.Time
	if input.ExpiresIn > 0 {
		exp := now.Add(input.ExpiresIn)
		expiresAt = &exp
	}
	_, err = s.pool.Exec(ctx, `
insert into gomyadmin_api_keys
    (id, name, prefix, secret_hash, actor_id, actor_email, actor_name, tenant_id, roles, permissions, scopes, expires_at, created_at)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		id, input.Name, prefix, hash,
		input.Actor.ID, input.Actor.Email, input.Actor.Name, nullableString(input.Actor.TenantID),
		rolesJSON, permissionsJSON, scopesJSON, expiresAt, now,
	)
	if err != nil {
		return APIKey{}, "", err
	}
	key := APIKey{
		ID:        id,
		Name:      input.Name,
		Prefix:    prefix,
		Scopes:    scopes,
		Actor:     actorWithPermissions(input.Actor, scopes),
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}
	return key, raw, nil
}

func (s *PGAPIKeys) Authenticate(ctx context.Context, rawKey string) (admin.Actor, APIKey, bool, error) {
	prefix, ok := parseAPIKeyPrefix(rawKey)
	if !ok {
		return admin.Actor{}, APIKey{}, false, nil
	}
	row, err := s.loadByPrefix(ctx, prefix)
	if errors.Is(err, ErrAPIKeyNotFound) {
		return admin.Actor{}, APIKey{}, false, nil
	}
	if err != nil {
		return admin.Actor{}, APIKey{}, false, err
	}
	if row.RevokedAt != nil || (row.ExpiresAt != nil && time.Now().UTC().After(*row.ExpiresAt)) {
		return admin.Actor{}, APIKey{}, false, nil
	}
	if subtle.ConstantTimeCompare([]byte(row.SecretHash), []byte(hashAPIKey(rawKey))) != 1 {
		return admin.Actor{}, APIKey{}, false, nil
	}
	now := time.Now().UTC()
	_, err = s.pool.Exec(ctx, `update gomyadmin_api_keys set last_used_at = $2 where id = $1`, row.ID, now)
	if err != nil {
		return admin.Actor{}, APIKey{}, false, err
	}
	row.LastUsedAt = &now
	return row.Actor, apiKeyFromRow(row), true, nil
}

func (s *PGAPIKeys) List(ctx context.Context, actor admin.Actor) ([]APIKey, error) {
	args := []any{actor.ID}
	where := ` where actor_id = $1`
	if actor.TenantID != "" && !actor.HasRole("super_admin") {
		args = append(args, actor.TenantID)
		where += ` and coalesce(tenant_id, '') = $2`
	}
	rows, err := s.pool.Query(ctx, `
select id, name, prefix, actor_id, actor_email, actor_name, coalesce(tenant_id, ''),
       roles, permissions, scopes, expires_at, last_used_at, revoked_at, created_at
from gomyadmin_api_keys`+where+` order by created_at desc`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var keys []APIKey
	for rows.Next() {
		row, err := scanAPIKeyRow(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, apiKeyFromRow(row))
	}
	return keys, rows.Err()
}

func (s *PGAPIKeys) Revoke(ctx context.Context, id string, actor admin.Actor) error {
	args := []any{id, actor.ID}
	sql := `update gomyadmin_api_keys set revoked_at = now() where id = $1 and actor_id = $2`
	if actor.TenantID != "" && !actor.HasRole("super_admin") {
		args = append(args, actor.TenantID)
		sql += ` and coalesce(tenant_id, '') = $3`
	}
	tag, err := s.pool.Exec(ctx, sql, args...)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrAPIKeyNotFound
	}
	return nil
}

type apiKeyRow struct {
	ID         string
	Name       string
	Prefix     string
	SecretHash string
	Actor      admin.Actor
	Scopes     []string
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	RevokedAt  *time.Time
	CreatedAt  time.Time
}

func (s *PGAPIKeys) loadByPrefix(ctx context.Context, prefix string) (apiKeyRow, error) {
	rows, err := s.pool.Query(ctx, `
select id, name, prefix, secret_hash, actor_id, actor_email, actor_name, coalesce(tenant_id, ''),
       roles, permissions, scopes, expires_at, last_used_at, revoked_at, created_at
from gomyadmin_api_keys
where prefix = $1
limit 1`, prefix)
	if err != nil {
		return apiKeyRow{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return apiKeyRow{}, ErrAPIKeyNotFound
	}
	var (
		row             apiKeyRow
		tenantID        string
		rolesJSON       []byte
		permissionsJSON []byte
		scopesJSON      []byte
	)
	if err := rows.Scan(
		&row.ID, &row.Name, &row.Prefix, &row.SecretHash, &row.Actor.ID, &row.Actor.Email, &row.Actor.Name, &tenantID,
		&rolesJSON, &permissionsJSON, &scopesJSON, &row.ExpiresAt, &row.LastUsedAt, &row.RevokedAt, &row.CreatedAt,
	); err != nil {
		return apiKeyRow{}, err
	}
	row.Actor.TenantID = tenantID
	if err := json.Unmarshal(rolesJSON, &row.Actor.Roles); err != nil {
		return apiKeyRow{}, err
	}
	if err := json.Unmarshal(permissionsJSON, &row.Actor.Permissions); err != nil {
		return apiKeyRow{}, err
	}
	if err := json.Unmarshal(scopesJSON, &row.Scopes); err != nil {
		return apiKeyRow{}, err
	}
	return row, nil
}

func scanAPIKeyRow(rows pgx.Rows) (apiKeyRow, error) {
	var (
		row             apiKeyRow
		tenantID        string
		rolesJSON       []byte
		permissionsJSON []byte
		scopesJSON      []byte
	)
	if err := rows.Scan(
		&row.ID, &row.Name, &row.Prefix, &row.Actor.ID, &row.Actor.Email, &row.Actor.Name, &tenantID,
		&rolesJSON, &permissionsJSON, &scopesJSON, &row.ExpiresAt, &row.LastUsedAt, &row.RevokedAt, &row.CreatedAt,
	); err != nil {
		return apiKeyRow{}, err
	}
	row.Actor.TenantID = tenantID
	if err := json.Unmarshal(rolesJSON, &row.Actor.Roles); err != nil {
		return apiKeyRow{}, err
	}
	if err := json.Unmarshal(permissionsJSON, &row.Actor.Permissions); err != nil {
		return apiKeyRow{}, err
	}
	if err := json.Unmarshal(scopesJSON, &row.Scopes); err != nil {
		return apiKeyRow{}, err
	}
	return row, nil
}

func apiKeyFromRow(row apiKeyRow) APIKey {
	return APIKey{
		ID:         row.ID,
		Name:       row.Name,
		Prefix:     row.Prefix,
		Scopes:     cloneStrings(row.Scopes),
		Actor:      row.Actor,
		ExpiresAt:  row.ExpiresAt,
		LastUsedAt: row.LastUsedAt,
		RevokedAt:  row.RevokedAt,
		CreatedAt:  row.CreatedAt,
	}
}

func actorWithPermissions(actor admin.Actor, permissions []string) admin.Actor {
	actor.Permissions = cloneStrings(permissions)
	return actor
}

func parseAPIKeyPrefix(rawKey string) (string, bool) {
	parts := strings.Split(rawKey, "_")
	if len(parts) != 3 || parts[0] != "gma" || parts[1] == "" || parts[2] == "" {
		return "", false
	}
	return parts[1], true
}

func hashAPIKey(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}

func nullableString(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var _ APIKeyManager = (*PGAPIKeys)(nil)
