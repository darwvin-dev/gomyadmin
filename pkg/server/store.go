package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

var errNotFound = errors.New("record not found")

// internalSchema creates the audit log and file tables used internally by pkg/server.
// The sessions table is created separately via auth.PGSessionStore.Migrate.
const internalSchema = `
CREATE TABLE IF NOT EXISTS gomyadmin_audit_logs (
    id          TEXT        PRIMARY KEY,
    actor_id    TEXT,
    actor_email TEXT,
    tenant_id   TEXT,
    action      TEXT        NOT NULL,
    resource    TEXT        NOT NULL,
    resource_id TEXT,
    old_values  JSONB,
    new_values  JSONB,
    ip_address  TEXT,
    user_agent  TEXT,
    request_id  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS gomyadmin_files (
    id           TEXT        PRIMARY KEY,
    tenant_id    TEXT,
    key          TEXT        NOT NULL UNIQUE,
    name         TEXT        NOT NULL,
    content_type TEXT        NOT NULL,
    size         BIGINT      NOT NULL DEFAULT 0,
    visibility   TEXT        NOT NULL DEFAULT 'private',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS gomyadmin_audit_logs_created ON gomyadmin_audit_logs (created_at DESC);
CREATE INDEX IF NOT EXISTS gomyadmin_audit_logs_tenant  ON gomyadmin_audit_logs (tenant_id);
CREATE INDEX IF NOT EXISTS gomyadmin_files_tenant       ON gomyadmin_files (tenant_id);
`

type auditEvent struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`
	ActorEmail string         `json:"actor_email"`
	TenantID   string         `json:"tenant_id"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	OldValues  map[string]any `json:"old_values,omitempty"`
	NewValues  map[string]any `json:"new_values,omitempty"`
	IPAddress  string         `json:"ip_address"`
	UserAgent  string         `json:"user_agent"`
	RequestID  string         `json:"request_id"`
	CreatedAt  time.Time      `json:"created_at"`
}

type serverStore struct {
	pool      *pgxpool.Pool
	resources map[string]resourceMeta // keyed by table name
}

func newServerStore(pool *pgxpool.Pool, app *admin.App) *serverStore {
	return &serverStore{pool: pool, resources: appToMeta(app)}
}

func (s *serverStore) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, internalSchema)
	return err
}

func (s *serverStore) resource(table string) (resourceMeta, bool) {
	r, ok := s.resources[table]
	return r, ok
}

// Resources returns all resource metadata in stable table-name order.
func (s *serverStore) Resources() []resourceMeta {
	keys := make([]string, 0, len(s.resources))
	for k := range s.resources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]resourceMeta, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.resources[k])
	}
	return out
}

func (s *serverStore) List(ctx context.Context, table, tenantID, role, search, sortBy string, filters map[string]string, page, perPage int) ([]record, int, error) {
	resource, ok := s.resource(table)
	if !ok {
		return nil, 0, errNotFound
	}
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 25
	}
	where, args, err := s.buildWhere(resource, tenantID, role, search, filters)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := s.pool.QueryRow(ctx, "select count(*) from "+quoteIdent(resource.Table)+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order, err := buildOrderClause(resource, sortBy)
	if err != nil {
		return nil, 0, err
	}
	limitN := len(args) + 1
	offsetN := len(args) + 2
	args = append(args, perPage, (page-1)*perPage)
	sql := "select " + selectCols(resource) + " from " + quoteIdent(resource.Table) + where + order + fmt.Sprintf(" limit $%d offset $%d", limitN, offsetN)
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	records, err := scanRows(rows, resource)
	return records, total, err
}

func (s *serverStore) Get(ctx context.Context, table, id, tenantID, role string) (record, error) {
	resource, ok := s.resource(table)
	if !ok {
		return nil, errNotFound
	}
	args := []any{id}
	where := " where " + quoteIdent(resource.PrimaryKey) + " = $1"
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" and %s = $%d", quoteIdent(resource.TenantKey), len(args))
	}
	rows, err := s.pool.Query(ctx, "select "+selectCols(resource)+" from "+quoteIdent(resource.Table)+where+" limit 1", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records, err := scanRows(rows, resource)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errNotFound
	}
	return records[0], nil
}

func (s *serverStore) Create(ctx context.Context, table, tenantID string, input record) (record, error) {
	resource, ok := s.resource(table)
	if !ok {
		return nil, errNotFound
	}
	now := time.Now().UTC()
	input = cloneRecord(input)

	pk := resource.PrimaryKey
	if input[pk] == nil {
		input[pk] = newID(resource)
	}
	if resource.TenantKey != "" {
		input[resource.TenantKey] = tenantID
	}
	if hasColumn(resource, "created_at") && input["created_at"] == nil {
		input["created_at"] = now
	}
	if hasColumn(resource, "updated_at") && input["updated_at"] == nil {
		input["updated_at"] = now
	}

	cols := make([]string, 0)
	placeholders := make([]string, 0)
	args := make([]any, 0)
	for _, field := range resource.Fields {
		if field.Hidden && field.SQLName != resource.TenantKey {
			continue
		}
		value, exists := input[field.SQLName]
		if !exists {
			continue
		}
		if field.Type == admin.FieldPassword {
			hashed, err := auth.HashPassword(fmt.Sprint(value), auth.DefaultPasswordConfig())
			if err != nil {
				return nil, err
			}
			value = hashed
		}
		cols = append(cols, quoteIdent(field.SQLName))
		args = append(args, normalizeValue(value))
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	if len(cols) == 0 {
		return nil, errors.New("no writable fields in input")
	}
	sql := "insert into " + quoteIdent(resource.Table) + " (" + strings.Join(cols, ", ") + ") values (" + strings.Join(placeholders, ", ") + ")"
	if _, err := s.pool.Exec(ctx, sql, args...); err != nil {
		return nil, err
	}
	return s.Get(ctx, table, fmt.Sprint(input[pk]), tenantID, "super_admin")
}

func (s *serverStore) Update(ctx context.Context, table, id, tenantID, role string, input record) (record, record, error) {
	oldRecord, err := s.Get(ctx, table, id, tenantID, role)
	if err != nil {
		return nil, nil, err
	}
	resource, _ := s.resource(table)
	assignments := make([]string, 0)
	args := make([]any, 0)
	for _, field := range resource.Fields {
		if field.Readonly || field.Hidden || field.SQLName == resource.PrimaryKey || field.SQLName == resource.TenantKey {
			continue
		}
		value, exists := input[field.SQLName]
		if !exists {
			continue
		}
		if field.Type == admin.FieldPassword {
			hashed, err := auth.HashPassword(fmt.Sprint(value), auth.DefaultPasswordConfig())
			if err != nil {
				return nil, nil, err
			}
			value = hashed
		}
		args = append(args, normalizeValue(value))
		assignments = append(assignments, fmt.Sprintf("%s = $%d", quoteIdent(field.SQLName), len(args)))
	}
	if hasColumn(resource, "updated_at") {
		args = append(args, time.Now().UTC())
		assignments = append(assignments, fmt.Sprintf("%s = $%d", quoteIdent("updated_at"), len(args)))
	}
	if len(assignments) == 0 {
		return oldRecord, oldRecord, nil
	}
	args = append(args, id)
	where := fmt.Sprintf(" where %s = $%d", quoteIdent(resource.PrimaryKey), len(args))
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" and %s = $%d", quoteIdent(resource.TenantKey), len(args))
	}
	if _, err := s.pool.Exec(ctx, "update "+quoteIdent(resource.Table)+" set "+strings.Join(assignments, ", ")+where, args...); err != nil {
		return nil, nil, err
	}
	newRecord, err := s.Get(ctx, table, id, tenantID, role)
	return oldRecord, newRecord, err
}

func (s *serverStore) Delete(ctx context.Context, table, id, tenantID, role string) (record, error) {
	oldRecord, err := s.Get(ctx, table, id, tenantID, role)
	if err != nil {
		return nil, err
	}
	resource, _ := s.resource(table)
	args := []any{id}
	where := fmt.Sprintf(" where %s = $1", quoteIdent(resource.PrimaryKey))
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" and %s = $%d", quoteIdent(resource.TenantKey), len(args))
	}
	_, err = s.pool.Exec(ctx, "delete from "+quoteIdent(resource.Table)+where, args...)
	return oldRecord, err
}

func (s *serverStore) RecordAudit(ctx context.Context, event auditEvent) {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = "audit_" + randomHex(8)
	}
	oldValues, _ := json.Marshal(event.OldValues)
	newValues, _ := json.Marshal(event.NewValues)
	_, _ = s.pool.Exec(ctx, `
insert into gomyadmin_audit_logs
    (id, actor_id, actor_email, tenant_id, action, resource, resource_id, old_values, new_values, ip_address, user_agent, request_id, created_at)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		event.ID, event.ActorID, event.ActorEmail, event.TenantID,
		event.Action, event.Resource, event.ResourceID,
		oldValues, newValues,
		event.IPAddress, event.UserAgent, event.RequestID, event.CreatedAt)
}

func (s *serverStore) Audit(ctx context.Context, tenantID, role string) ([]auditEvent, error) {
	args := []any{}
	where := ""
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where = " where tenant_id = $1"
	}
	rows, err := s.pool.Query(ctx, `
select id,
       coalesce(actor_id,''), coalesce(actor_email,''), coalesce(tenant_id,''),
       action, resource, coalesce(resource_id,''),
       coalesce(old_values,'{}'::jsonb), coalesce(new_values,'{}'::jsonb),
       coalesce(ip_address,''), coalesce(user_agent,''), coalesce(request_id,''),
       created_at
from gomyadmin_audit_logs`+where+` order by created_at desc limit 100`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var events []auditEvent
	for rows.Next() {
		var e auditEvent
		var oldVal, newVal []byte
		if err := rows.Scan(
			&e.ID, &e.ActorID, &e.ActorEmail, &e.TenantID,
			&e.Action, &e.Resource, &e.ResourceID,
			&oldVal, &newVal,
			&e.IPAddress, &e.UserAgent, &e.RequestID, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(oldVal, &e.OldValues)
		_ = json.Unmarshal(newVal, &e.NewValues)
		events = append(events, e)
	}
	return events, rows.Err()
}

func (s *serverStore) AddFile(ctx context.Context, r record) error {
	if r["id"] == nil {
		r["id"] = "file_" + randomHex(8)
	}
	if r["created_at"] == nil {
		r["created_at"] = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
insert into gomyadmin_files (id, tenant_id, key, name, content_type, size, visibility, created_at)
values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		r["id"], r["tenant_id"], r["key"], r["name"], r["content_type"], r["size"], "private", r["created_at"])
	return err
}

func (s *serverStore) Files(ctx context.Context, tenantID, role string) ([]record, error) {
	args := []any{}
	where := ""
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where = " where tenant_id = $1"
	}
	rows, err := s.pool.Query(ctx, `
select id, coalesce(tenant_id,''), key, name, content_type, size, visibility, created_at
from gomyadmin_files`+where+` order by created_at desc`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var records []record
	for rows.Next() {
		var id, tid, key, name, ct, vis string
		var size int64
		var createdAt time.Time
		if err := rows.Scan(&id, &tid, &key, &name, &ct, &size, &vis, &createdAt); err != nil {
			return nil, err
		}
		records = append(records, record{
			"id": id, "tenant_id": tid, "key": key, "name": name,
			"content_type": ct, "size": size, "visibility": vis, "created_at": createdAt,
		})
	}
	return records, rows.Err()
}

func (s *serverStore) FileKey(ctx context.Context, id, tenantID, role string) (string, error) {
	args := []any{id}
	where := " where id = $1"
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" and tenant_id = $%d", len(args))
	}
	var key string
	err := s.pool.QueryRow(ctx, "select key from gomyadmin_files"+where+" limit 1", args...).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errNotFound
	}
	return key, err
}

// SQL helpers

func (s *serverStore) buildWhere(resource resourceMeta, tenantID, role, search string, filters map[string]string) (string, []any, error) {
	var clauses []string
	var args []any

	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", quoteIdent(resource.TenantKey), len(args)))
	}
	if search != "" {
		var parts []string
		for _, f := range resource.Fields {
			if !f.Searchable || f.Hidden {
				continue
			}
			args = append(args, "%"+search+"%")
			parts = append(parts, fmt.Sprintf("%s::text ilike $%d", quoteIdent(f.SQLName), len(args)))
		}
		if len(parts) > 0 {
			clauses = append(clauses, "("+strings.Join(parts, " or ")+")")
		}
	}
	for key, value := range filters {
		if value == "" {
			continue
		}
		fieldName, operator := key, "eq"
		if before, after, ok := strings.Cut(key, "::"); ok {
			fieldName, operator = before, after
		}
		f, ok := fieldBySQLName(resource, fieldName)
		if !ok || !f.Filterable {
			return "", nil, fmt.Errorf("field %q is not filterable", fieldName)
		}
		switch operator {
		case "eq":
			args = append(args, value)
			clauses = append(clauses, fmt.Sprintf("%s::text = $%d", quoteIdent(fieldName), len(args)))
		case "contains":
			args = append(args, "%"+value+"%")
			clauses = append(clauses, fmt.Sprintf("%s::text ilike $%d", quoteIdent(fieldName), len(args)))
		case "starts_with":
			args = append(args, value+"%")
			clauses = append(clauses, fmt.Sprintf("%s::text ilike $%d", quoteIdent(fieldName), len(args)))
		case "ends_with":
			args = append(args, "%"+value)
			clauses = append(clauses, fmt.Sprintf("%s::text ilike $%d", quoteIdent(fieldName), len(args)))
		case "gte", "lte", "gt", "lt":
			args = append(args, value)
			op := map[string]string{"gte": ">=", "lte": "<=", "gt": ">", "lt": "<"}[operator]
			clauses = append(clauses, fmt.Sprintf("%s::text %s $%d", quoteIdent(fieldName), op, len(args)))
		default:
			return "", nil, fmt.Errorf("unsupported filter operator %q", operator)
		}
	}
	if len(clauses) == 0 {
		return "", args, nil
	}
	return " where " + strings.Join(clauses, " and "), args, nil
}

func scanRows(rows pgx.Rows, resource resourceMeta) ([]record, error) {
	columns := visibleColumns(resource)
	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	var records []record
	for rows.Next() {
		for i := range values {
			values[i] = nil
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		row := make(record, len(columns))
		for i, col := range columns {
			row[col] = normalizeScanned(values[i])
		}
		records = append(records, row)
	}
	return records, rows.Err()
}

func selectCols(resource resourceMeta) string {
	cols := visibleColumns(resource)
	quoted := make([]string, len(cols))
	for i, c := range cols {
		quoted[i] = quoteIdent(c)
	}
	return strings.Join(quoted, ", ")
}

func visibleColumns(resource resourceMeta) []string {
	cols := make([]string, 0, len(resource.Fields))
	for _, f := range resource.Fields {
		if !f.Hidden {
			cols = append(cols, f.SQLName)
		}
	}
	return cols
}

func buildOrderClause(resource resourceMeta, sortBy string) (string, error) {
	if sortBy == "" {
		sortBy = resource.DefaultSort
	}
	if sortBy == "" {
		sortBy = "-created_at"
	}
	desc := strings.HasPrefix(sortBy, "-")
	fieldName := strings.TrimPrefix(sortBy, "-")
	f, ok := fieldBySQLName(resource, fieldName)
	if !ok || !f.Sortable {
		return "", nil
	}
	dir := "asc"
	if desc {
		dir = "desc"
	}
	return " order by " + quoteIdent(fieldName) + " " + dir, nil
}

func fieldBySQLName(resource resourceMeta, sqlName string) (fieldMeta, bool) {
	for _, f := range resource.Fields {
		if f.SQLName == sqlName {
			return f, true
		}
	}
	return fieldMeta{}, false
}

func hasColumn(resource resourceMeta, col string) bool {
	_, ok := fieldBySQLName(resource, col)
	return ok
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case map[string]any, []any:
		b, _ := json.Marshal(v)
		return b
	default:
		return v
	}
}

func normalizeScanned(value any) any {
	if b, ok := value.([]byte); ok {
		return string(b)
	}
	return value
}

func cloneRecord(r record) record {
	out := make(record, len(r))
	for k, v := range r {
		out[k] = v
	}
	return out
}

func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// newID generates a primary key for the given resource.
// UUID fields get a proper UUID v4; everything else gets a random hex string.
func newID(resource resourceMeta) string {
	for _, f := range resource.Fields {
		if f.Primary && f.Type == admin.FieldUUID {
			return newUUID()
		}
	}
	return randomHex(8)
}

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
