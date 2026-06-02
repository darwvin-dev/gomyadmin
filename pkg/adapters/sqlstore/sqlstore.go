// Package sqlstore provides a database/sql-backed AdminStore adapter.
package sqlstore

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
)

type Dialect string

const (
	DialectMySQL  Dialect = "mysql"
	DialectSQLite Dialect = "sqlite"
)

type Store struct {
	server.ResourceMetadataStore
	db        *sql.DB
	dialect   Dialect
	resources map[string]server.ResourceMeta
}

func New(db *sql.DB, app *admin.App, dialect Dialect) *Store {
	meta := server.NewResourceMetadataStore(app)
	resources := map[string]server.ResourceMeta{}
	for _, resource := range meta.Resources() {
		resources[resource.Table] = resource
	}
	return &Store{ResourceMetadataStore: meta, db: db, dialect: dialect, resources: resources}
}

func MySQL(db *sql.DB, app *admin.App) *Store {
	return New(db, app, DialectMySQL)
}

func SQLite(db *sql.DB, app *admin.App) *Store {
	return New(db, app, DialectSQLite)
}

func (s *Store) List(ctx context.Context, table, tenantID, role, search, sortBy string, filters map[string]string, page, perPage int) ([]server.Record, int, error) {
	resource, ok := s.resources[table]
	if !ok {
		return nil, 0, server.ErrNotFound
	}
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 || perPage > 100 {
		perPage = 25
	}
	where, args, err := s.where(resource, tenantID, role, search, filters)
	if err != nil {
		return nil, 0, err
	}
	var total int
	if err := s.db.QueryRowContext(ctx, "select count(*) from "+s.quote(resource.Table)+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	order := s.order(resource, sortBy)
	args = append(args, perPage, (page-1)*perPage)
	query := "select " + s.selectCols(resource) + " from " + s.quote(resource.Table) + where + order + " limit " + s.placeholder(len(args)-1) + " offset " + s.placeholder(len(args))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	records, err := scanRows(rows)
	return records, total, err
}

func (s *Store) Get(ctx context.Context, table, id, tenantID, role string) (server.Record, error) {
	resource, ok := s.resources[table]
	if !ok {
		return nil, server.ErrNotFound
	}
	args := []any{id}
	where := " where " + s.quote(resource.PrimaryKey) + " = " + s.placeholder(1)
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += " and " + s.quote(resource.TenantKey) + " = " + s.placeholder(len(args))
	}
	rows, err := s.db.QueryContext(ctx, "select "+s.selectCols(resource)+" from "+s.quote(resource.Table)+where+" limit 1", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records, err := scanRows(rows)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, server.ErrNotFound
	}
	return records[0], nil
}

func (s *Store) Create(ctx context.Context, table, tenantID string, input server.Record) (server.Record, error) {
	resource, ok := s.resources[table]
	if !ok {
		return nil, server.ErrNotFound
	}
	input = clone(input)
	now := time.Now().UTC()
	if input[resource.PrimaryKey] == nil {
		input[resource.PrimaryKey] = newID()
	}
	if resource.TenantKey != "" {
		input[resource.TenantKey] = tenantID
	}
	if hasField(resource, "created_at") && input["created_at"] == nil {
		input["created_at"] = now
	}
	if hasField(resource, "updated_at") && input["updated_at"] == nil {
		input["updated_at"] = now
	}
	cols, marks, args, err := s.insertValues(resource, input)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("no writable fields in input")
	}
	_, err = s.db.ExecContext(ctx, "insert into "+s.quote(resource.Table)+" ("+strings.Join(cols, ", ")+") values ("+strings.Join(marks, ", ")+")", args...)
	if err != nil {
		return nil, err
	}
	return s.Get(ctx, table, fmt.Sprint(input[resource.PrimaryKey]), tenantID, "super_admin")
}

func (s *Store) Update(ctx context.Context, table, id, tenantID, role string, input server.Record) (server.Record, server.Record, error) {
	oldRecord, err := s.Get(ctx, table, id, tenantID, role)
	if err != nil {
		return nil, nil, err
	}
	resource := s.resources[table]
	assignments := []string{}
	args := []any{}
	for _, field := range resource.Fields {
		if field.Readonly || field.Hidden || field.SQLName == resource.PrimaryKey || field.SQLName == resource.TenantKey {
			continue
		}
		value, ok := input[field.SQLName]
		if !ok {
			continue
		}
		if field.Type == admin.FieldPassword {
			hashed, err := auth.HashPassword(fmt.Sprint(value), auth.DefaultPasswordConfig())
			if err != nil {
				return nil, nil, err
			}
			value = hashed
		}
		args = append(args, normalize(value))
		assignments = append(assignments, s.quote(field.SQLName)+" = "+s.placeholder(len(args)))
	}
	if hasField(resource, "updated_at") {
		args = append(args, time.Now().UTC())
		assignments = append(assignments, s.quote("updated_at")+" = "+s.placeholder(len(args)))
	}
	if len(assignments) == 0 {
		return oldRecord, oldRecord, nil
	}
	args = append(args, id)
	where := " where " + s.quote(resource.PrimaryKey) + " = " + s.placeholder(len(args))
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += " and " + s.quote(resource.TenantKey) + " = " + s.placeholder(len(args))
	}
	if _, err := s.db.ExecContext(ctx, "update "+s.quote(resource.Table)+" set "+strings.Join(assignments, ", ")+where, args...); err != nil {
		return nil, nil, err
	}
	newRecord, err := s.Get(ctx, table, id, tenantID, role)
	return oldRecord, newRecord, err
}

func (s *Store) Delete(ctx context.Context, table, id, tenantID, role string) (server.Record, error) {
	records, err := s.DeleteMany(ctx, table, []string{id}, tenantID, role)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, server.ErrNotFound
	}
	return records[0], nil
}

func (s *Store) DeleteMany(ctx context.Context, table string, ids []string, tenantID, role string) ([]server.Record, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()
	deleted := []server.Record{}
	for _, id := range ids {
		old, getErr := s.Get(ctx, table, id, tenantID, role)
		if getErr != nil {
			err = getErr
			return nil, err
		}
		resource := s.resources[table]
		args := []any{id}
		where := " where " + s.quote(resource.PrimaryKey) + " = " + s.placeholder(1)
		if resource.TenantKey != "" && role != "super_admin" {
			args = append(args, tenantID)
			where += " and " + s.quote(resource.TenantKey) + " = " + s.placeholder(len(args))
		}
		if _, execErr := tx.ExecContext(ctx, "delete from "+s.quote(resource.Table)+where, args...); execErr != nil {
			err = execErr
			return nil, err
		}
		deleted = append(deleted, old)
	}
	err = tx.Commit()
	return deleted, err
}

func (s *Store) RecordAudit(ctx context.Context, event server.AuditEvent) {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	oldValues, _ := json.Marshal(event.OldValues)
	newValues, _ := json.Marshal(event.NewValues)
	_, _ = s.db.ExecContext(ctx, `insert into gomyadmin_audit_logs
(id, actor_id, actor_email, tenant_id, action, resource, resource_id, old_values, new_values, ip_address, user_agent, request_id, created_at)
values (`+s.placeholders(13)+`)`,
		first(event.ID, "audit_"+newID()), event.ActorID, event.ActorEmail, event.TenantID, event.Action, event.Resource, event.ResourceID,
		string(oldValues), string(newValues), event.IPAddress, event.UserAgent, event.RequestID, event.CreatedAt)
}

func (s *Store) Audit(ctx context.Context, tenantID, role string) ([]server.AuditEvent, error) {
	args := []any{}
	where := ""
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where = " where tenant_id = " + s.placeholder(1)
	}
	rows, err := s.db.QueryContext(ctx, `select id, actor_id, actor_email, tenant_id, action, resource, resource_id, old_values, new_values, ip_address, user_agent, request_id, created_at from gomyadmin_audit_logs`+where+` order by created_at desc limit 100`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []server.AuditEvent{}
	for rows.Next() {
		var event server.AuditEvent
		var oldValues, newValues string
		var createdAt any
		if err := rows.Scan(&event.ID, &event.ActorID, &event.ActorEmail, &event.TenantID, &event.Action, &event.Resource, &event.ResourceID, &oldValues, &newValues, &event.IPAddress, &event.UserAgent, &event.RequestID, &createdAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(oldValues), &event.OldValues)
		_ = json.Unmarshal([]byte(newValues), &event.NewValues)
		switch v := createdAt.(type) {
		case time.Time:
			event.CreatedAt = v
		case string:
			event.CreatedAt, _ = time.Parse(time.RFC3339Nano, v)
		}
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) AddFile(ctx context.Context, record server.Record) error {
	_, err := s.db.ExecContext(ctx, "insert into gomyadmin_files (id, tenant_id, key, name, content_type, size, visibility, created_at) values ("+s.placeholders(8)+")",
		first(fmt.Sprint(record["id"]), "file_"+newID()), record["tenant_id"], record["key"], record["name"], record["content_type"], record["size"], "private", firstTime(record["created_at"]))
	return err
}

func (s *Store) Files(ctx context.Context, tenantID, role string) ([]server.Record, error) {
	args := []any{}
	where := ""
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where = " where tenant_id = " + s.placeholder(1)
	}
	rows, err := s.db.QueryContext(ctx, "select id, tenant_id, key, name, content_type, size, visibility, created_at from gomyadmin_files"+where+" order by created_at desc", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s *Store) FileKey(ctx context.Context, id, tenantID, role string) (string, error) {
	args := []any{id}
	where := " where id = " + s.placeholder(1)
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += " and tenant_id = " + s.placeholder(len(args))
	}
	var key string
	if err := s.db.QueryRowContext(ctx, "select key from gomyadmin_files"+where+" limit 1", args...).Scan(&key); err != nil {
		if err == sql.ErrNoRows {
			return "", server.ErrNotFound
		}
		return "", err
	}
	return key, nil
}

func (s *Store) where(resource server.ResourceMeta, tenantID, role, search string, filters map[string]string) (string, []any, error) {
	clauses := []string{}
	args := []any{}
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		clauses = append(clauses, s.quote(resource.TenantKey)+" = "+s.placeholder(len(args)))
	}
	if search != "" {
		parts := []string{}
		for _, field := range resource.Fields {
			if !field.Searchable || field.Hidden {
				continue
			}
			args = append(args, "%"+search+"%")
			parts = append(parts, "lower(cast("+s.quote(field.SQLName)+" as char)) like lower("+s.placeholder(len(args))+")")
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
		field, ok := fieldBySQLName(resource, fieldName)
		if !ok || !field.Filterable {
			return "", nil, fmt.Errorf("field %q is not filterable", fieldName)
		}
		switch operator {
		case "eq":
			args = append(args, value)
			clauses = append(clauses, "cast("+s.quote(fieldName)+" as char) = "+s.placeholder(len(args)))
		case "contains":
			args = append(args, "%"+value+"%")
			clauses = append(clauses, "lower(cast("+s.quote(fieldName)+" as char)) like lower("+s.placeholder(len(args))+")")
		case "starts_with":
			args = append(args, value+"%")
			clauses = append(clauses, "lower(cast("+s.quote(fieldName)+" as char)) like lower("+s.placeholder(len(args))+")")
		case "ends_with":
			args = append(args, "%"+value)
			clauses = append(clauses, "lower(cast("+s.quote(fieldName)+" as char)) like lower("+s.placeholder(len(args))+")")
		case "gte", "lte", "gt", "lt":
			args = append(args, value)
			op := map[string]string{"gte": ">=", "lte": "<=", "gt": ">", "lt": "<"}[operator]
			clauses = append(clauses, s.quote(fieldName)+" "+op+" "+s.placeholder(len(args)))
		default:
			return "", nil, fmt.Errorf("unsupported filter operator %q", operator)
		}
	}
	if len(clauses) == 0 {
		return "", args, nil
	}
	return " where " + strings.Join(clauses, " and "), args, nil
}

func (s *Store) order(resource server.ResourceMeta, sortBy string) string {
	if sortBy == "" {
		sortBy = resource.DefaultSort
	}
	desc := strings.HasPrefix(sortBy, "-")
	fieldName := strings.TrimPrefix(sortBy, "-")
	field, ok := fieldBySQLName(resource, fieldName)
	if !ok || !field.Sortable {
		return ""
	}
	if desc {
		return " order by " + s.quote(fieldName) + " desc"
	}
	return " order by " + s.quote(fieldName) + " asc"
}

func (s *Store) insertValues(resource server.ResourceMeta, input server.Record) ([]string, []string, []any, error) {
	cols := []string{}
	marks := []string{}
	args := []any{}
	for _, field := range resource.Fields {
		if field.Hidden && field.SQLName != resource.TenantKey {
			continue
		}
		value, ok := input[field.SQLName]
		if !ok {
			continue
		}
		if field.Type == admin.FieldPassword {
			hashed, err := auth.HashPassword(fmt.Sprint(value), auth.DefaultPasswordConfig())
			if err != nil {
				return nil, nil, nil, err
			}
			value = hashed
		}
		args = append(args, normalize(value))
		cols = append(cols, s.quote(field.SQLName))
		marks = append(marks, s.placeholder(len(args)))
	}
	return cols, marks, args, nil
}

func (s *Store) selectCols(resource server.ResourceMeta) string {
	cols := []string{}
	for _, field := range resource.Fields {
		if !field.Hidden {
			cols = append(cols, s.quote(field.SQLName))
		}
	}
	return strings.Join(cols, ", ")
}

func (s *Store) placeholder(n int) string {
	return "?"
}

func (s *Store) placeholders(n int) string {
	out := make([]string, n)
	for i := range out {
		out[i] = s.placeholder(i + 1)
	}
	return strings.Join(out, ", ")
}

func (s *Store) quote(identifier string) string {
	switch s.dialect {
	case DialectMySQL:
		return "`" + strings.ReplaceAll(identifier, "`", "``") + "`"
	default:
		return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
	}
}

func scanRows(rows *sql.Rows) ([]server.Record, error) {
	cols, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	values := make([]any, len(cols))
	pointers := make([]any, len(cols))
	records := []server.Record{}
	for rows.Next() {
		for i := range values {
			values[i] = nil
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		record := server.Record{}
		for i, col := range cols {
			record[col] = normalizeScanned(values[i])
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func fieldBySQLName(resource server.ResourceMeta, sqlName string) (server.FieldMeta, bool) {
	for _, field := range resource.Fields {
		if field.SQLName == sqlName {
			return field, true
		}
	}
	return server.FieldMeta{}, false
}

func hasField(resource server.ResourceMeta, sqlName string) bool {
	_, ok := fieldBySQLName(resource, sqlName)
	return ok
}

func clone(record server.Record) server.Record {
	out := server.Record{}
	for k, v := range record {
		out[k] = v
	}
	return out
}

func normalize(value any) any {
	switch v := value.(type) {
	case map[string]any, []any:
		b, _ := json.Marshal(v)
		return string(b)
	default:
		return v
	}
}

func normalizeScanned(value any) any {
	switch v := value.(type) {
	case []byte:
		return string(v)
	default:
		return v
	}
}

func first(value, fallback string) string {
	if strings.TrimSpace(value) == "" || value == "<nil>" {
		return fallback
	}
	return value
}

func firstTime(value any) time.Time {
	if t, ok := value.(time.Time); ok {
		return t
	}
	return time.Now().UTC()
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

var _ server.AdminStore = (*Store)(nil)
