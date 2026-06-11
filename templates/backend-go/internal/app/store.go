package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

var errNotFound = errors.New("record not found")

type Store struct {
	pool      *pgxpool.Pool
	resources map[string]ResourceMeta
}

func NewPostgresStore(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	store := &Store{pool: pool, resources: demoResources()}
	if err := store.migrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	if err := store.seed(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) Resources() []ResourceMeta {
	names := make([]string, 0, len(s.resources))
	for name := range s.resources {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]ResourceMeta, 0, len(names))
	for _, name := range names {
		out = append(out, s.resources[name])
	}
	return out
}

func (s *Store) Resource(name string) (ResourceMeta, bool) {
	resource, ok := s.resources[name]
	return resource, ok
}

func (s *Store) Authenticate(ctx context.Context, email, password string) (adminActor Actor, ok bool, err error) {
	var actor Actor
	var passwordHash string
	var role string
	err = s.pool.QueryRow(ctx, `
select id, email, name, tenant_id, role, password_hash
from users
where email = $1 and status = 'active'
limit 1`, email).Scan(&actor.ID, &actor.Email, &actor.Name, &actor.TenantID, &role, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, false, nil
	}
	if err != nil {
		return Actor{}, false, err
	}
	matches, err := auth.VerifyPassword(password, passwordHash)
	if err != nil || !matches {
		return Actor{}, false, err
	}
	actor.Roles = []string{role}
	actor.Permissions = []string{"*"}
	return actor, true, nil
}

func (s *Store) ActiveUserByEmail(ctx context.Context, email string) (Actor, error) {
	var actor Actor
	var role string
	err := s.pool.QueryRow(ctx, `
select id, email, name, tenant_id, role
from users
where email = $1 and status = 'active'
limit 1`, email).Scan(&actor.ID, &actor.Email, &actor.Name, &actor.TenantID, &role)
	if errors.Is(err, pgx.ErrNoRows) {
		return Actor{}, errNotFound
	}
	if err != nil {
		return Actor{}, err
	}
	actor.Roles = []string{role}
	actor.Permissions = []string{"*"}
	return actor, nil
}

func (s *Store) Tenants(ctx context.Context) ([]map[string]string, error) {
	rows, err := s.pool.Query(ctx, "select id, name from tenants order by name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	tenants := []map[string]string{}
	for rows.Next() {
		var id, name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		tenants = append(tenants, map[string]string{"id": id, "name": name})
	}
	return tenants, rows.Err()
}

func (s *Store) List(ctx context.Context, resourceName, tenantID, role, search, sortBy string, filters map[string]string, page, perPage int) ([]Record, int, error) {
	resource, ok := s.Resource(resourceName)
	if !ok {
		return nil, 0, errNotFound
	}
	if page <= 0 {
		page = 1
	}
	if perPage <= 0 {
		perPage = 25
	}
	if perPage > 100 {
		perPage = 100
	}

	where, args, err := s.where(resource, tenantID, role, search, filters)
	if err != nil {
		return nil, 0, err
	}

	countSQL := "select count(*) from " + quoteIdent(resource.Table) + where
	var total int
	if err := s.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	orderBy, err := orderClause(resource, sortBy)
	if err != nil {
		return nil, 0, err
	}
	limitArg := len(args) + 1
	offsetArg := len(args) + 2
	args = append(args, perPage, (page-1)*perPage)
	sql := "select " + selectColumns(resource) + " from " + quoteIdent(resource.Table) + where + orderBy + fmt.Sprintf(" limit $%d offset $%d", limitArg, offsetArg)
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	records, err := scanRecords(rows, resource)
	if err != nil {
		return nil, 0, err
	}
	return records, total, nil
}

func (s *Store) Get(ctx context.Context, resourceName, id, tenantID, role string) (Record, error) {
	resource, ok := s.Resource(resourceName)
	if !ok {
		return nil, errNotFound
	}
	args := []any{id}
	where := " where " + quoteIdent(resource.PrimaryKey) + " = $1"
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" and %s = $%d", quoteIdent(resource.TenantKey), len(args))
	}
	sql := "select " + selectColumns(resource) + " from " + quoteIdent(resource.Table) + where + " limit 1"
	rows, err := s.pool.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records, err := scanRecords(rows, resource)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, errNotFound
	}
	return records[0], nil
}

func (s *Store) Create(ctx context.Context, resourceName, tenantID string, input Record) (Record, error) {
	resource, ok := s.Resource(resourceName)
	if !ok {
		return nil, errNotFound
	}
	now := time.Now().UTC()
	input = clone(input)
	input["id"] = randomID(resourceName)
	if resource.TenantKey != "" {
		input[resource.TenantKey] = tenantID
	}
	if hasField(resource, "created_at") {
		input["created_at"] = now
	}
	if hasField(resource, "updated_at") {
		input["updated_at"] = now
	}

	columns := []string{}
	placeholders := []string{}
	args := []any{}
	for _, field := range resource.Fields {
		if (field.Hidden && field.Name != resource.TenantKey) || field.Name == "password_hash" {
			continue
		}
		value, exists := input[field.Name]
		if !exists {
			continue
		}
		columns = append(columns, quoteIdent(field.Name))
		args = append(args, normalizeValue(value))
		placeholders = append(placeholders, fmt.Sprintf("$%d", len(args)))
	}
	sql := "insert into " + quoteIdent(resource.Table) + " (" + strings.Join(columns, ", ") + ") values (" + strings.Join(placeholders, ", ") + ")"
	if _, err := s.pool.Exec(ctx, sql, args...); err != nil {
		return nil, err
	}
	return s.Get(ctx, resourceName, stringValue(input["id"]), tenantID, "super_admin")
}

func (s *Store) Update(ctx context.Context, resourceName, id, tenantID, role string, input Record) (Record, Record, error) {
	oldRecord, err := s.Get(ctx, resourceName, id, tenantID, role)
	if err != nil {
		return nil, nil, err
	}
	resource, _ := s.Resource(resourceName)
	assignments := []string{}
	args := []any{}
	for _, field := range resource.Fields {
		if field.Readonly || field.Hidden || field.Name == resource.PrimaryKey || field.Name == resource.TenantKey {
			continue
		}
		value, exists := input[field.Name]
		if !exists {
			continue
		}
		args = append(args, normalizeValue(value))
		assignments = append(assignments, fmt.Sprintf("%s = $%d", quoteIdent(field.Name), len(args)))
	}
	if hasField(resource, "updated_at") {
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
	sql := "update " + quoteIdent(resource.Table) + " set " + strings.Join(assignments, ", ") + where
	if _, err := s.pool.Exec(ctx, sql, args...); err != nil {
		return nil, nil, err
	}
	newRecord, err := s.Get(ctx, resourceName, id, tenantID, role)
	if err != nil {
		return nil, nil, err
	}
	return oldRecord, newRecord, nil
}

func (s *Store) Delete(ctx context.Context, resourceName, id, tenantID, role string) (Record, error) {
	oldRecord, err := s.Get(ctx, resourceName, id, tenantID, role)
	if err != nil {
		return nil, err
	}
	resource, _ := s.Resource(resourceName)
	args := []any{id}
	where := fmt.Sprintf(" where %s = $1", quoteIdent(resource.PrimaryKey))
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" and %s = $%d", quoteIdent(resource.TenantKey), len(args))
	}
	if _, err := s.pool.Exec(ctx, "delete from "+quoteIdent(resource.Table)+where, args...); err != nil {
		return nil, err
	}
	return oldRecord, nil
}

func (s *Store) RecordAudit(ctx context.Context, event AuditEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = randomID("audit")
	}
	oldValues, _ := json.Marshal(event.OldValues)
	newValues, _ := json.Marshal(event.NewValues)
	metadata, _ := json.Marshal(event.Metadata)
	_, err := s.pool.Exec(ctx, `
insert into audit_logs (id, actor_id, actor_email, tenant_id, action, resource, resource_id, old_values, new_values, metadata, ip_address, user_agent, request_id, created_at)
values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		event.ID, event.ActorID, event.ActorEmail, event.TenantID, event.Action, event.Resource, event.ResourceID,
		oldValues, newValues, metadata, event.IPAddress, event.UserAgent, event.RequestID, event.CreatedAt)
	return err
}

func (s *Store) Audit(ctx context.Context, tenantID, role string) ([]AuditEvent, error) {
	args := []any{}
	where := ""
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where = " where tenant_id = $1"
	}
	rows, err := s.pool.Query(ctx, `
select id, coalesce(actor_id, ''), coalesce(actor_email, ''), coalesce(tenant_id, ''), action, resource, coalesce(resource_id, ''),
       coalesce(old_values, '{}'::jsonb), coalesce(new_values, '{}'::jsonb), coalesce(metadata, '{}'::jsonb),
       coalesce(ip_address, ''), coalesce(user_agent, ''), coalesce(request_id, ''), created_at
from audit_logs`+where+` order by created_at desc limit 100`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []AuditEvent{}
	for rows.Next() {
		var event AuditEvent
		var oldValues, newValues, metadata []byte
		if err := rows.Scan(&event.ID, &event.ActorID, &event.ActorEmail, &event.TenantID, &event.Action, &event.Resource, &event.ResourceID, &oldValues, &newValues, &metadata, &event.IPAddress, &event.UserAgent, &event.RequestID, &event.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(oldValues, &event.OldValues)
		_ = json.Unmarshal(newValues, &event.NewValues)
		_ = json.Unmarshal(metadata, &event.Metadata)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (s *Store) AddFile(ctx context.Context, record Record) error {
	if record["id"] == nil {
		record["id"] = randomID("file")
	}
	if record["created_at"] == nil {
		record["created_at"] = time.Now().UTC()
	}
	_, err := s.pool.Exec(ctx, `
insert into files (id, tenant_id, key, name, content_type, size, visibility, created_at)
values ($1,$2,$3,$4,$5,$6,$7,$8)`,
		record["id"], record["tenant_id"], record["key"], record["name"], record["content_type"], record["size"], "private", record["created_at"])
	return err
}

func (s *Store) Files(ctx context.Context, tenantID, role string) ([]Record, error) {
	args := []any{}
	where := ""
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where = " where tenant_id = $1"
	}
	rows, err := s.pool.Query(ctx, `select id, tenant_id, key, name, content_type, size, visibility, created_at from files`+where+` order by created_at desc`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	records := []Record{}
	for rows.Next() {
		var id, tenantID, key, name, contentType, visibility string
		var size int64
		var createdAt time.Time
		if err := rows.Scan(&id, &tenantID, &key, &name, &contentType, &size, &visibility, &createdAt); err != nil {
			return nil, err
		}
		records = append(records, Record{"id": id, "tenant_id": tenantID, "key": key, "name": name, "content_type": contentType, "size": size, "visibility": visibility, "created_at": createdAt})
	}
	return records, rows.Err()
}

func (s *Store) where(resource ResourceMeta, tenantID, role, search string, filters map[string]string) (string, []any, error) {
	clauses := []string{}
	args := []any{}
	if resource.TenantKey != "" && role != "super_admin" {
		args = append(args, tenantID)
		clauses = append(clauses, fmt.Sprintf("%s = $%d", quoteIdent(resource.TenantKey), len(args)))
	}
	if search != "" {
		parts := []string{}
		for _, field := range resource.Fields {
			if !field.Searchable || field.Hidden {
				continue
			}
			args = append(args, "%"+search+"%")
			parts = append(parts, fmt.Sprintf("%s::text ilike $%d", quoteIdent(field.Name), len(args)))
		}
		if len(parts) > 0 {
			clauses = append(clauses, "("+strings.Join(parts, " or ")+")")
		}
	}
	for key, value := range filters {
		if value == "" {
			continue
		}
		fieldName := key
		operator := "eq"
		if before, after, ok := strings.Cut(key, "::"); ok {
			fieldName = before
			operator = after
		}
		field, ok := fieldByName(resource, fieldName)
		if !ok || !field.Filterable {
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
			sqlOperator := map[string]string{"gte": ">=", "lte": "<=", "gt": ">", "lt": "<"}[operator]
			clauses = append(clauses, fmt.Sprintf("%s::text %s $%d", quoteIdent(fieldName), sqlOperator, len(args)))
		default:
			return "", nil, fmt.Errorf("operator %q is not supported for %q", operator, fieldName)
		}
	}
	if len(clauses) == 0 {
		return "", args, nil
	}
	return " where " + strings.Join(clauses, " and "), args, nil
}

func scanRecords(rows pgx.Rows, resource ResourceMeta) ([]Record, error) {
	records := []Record{}
	fieldNames := listFieldNames(resource)
	values := make([]any, len(fieldNames))
	pointers := make([]any, len(fieldNames))
	for rows.Next() {
		for i := range values {
			values[i] = nil
		}
		for i := range values {
			pointers[i] = &values[i]
		}
		if err := rows.Scan(pointers...); err != nil {
			return nil, err
		}
		record := Record{}
		for i, name := range fieldNames {
			record[name] = normalizeScanned(values[i])
		}
		records = append(records, record)
	}
	return records, rows.Err()
}

func (s *Store) FileKey(ctx context.Context, id, tenantID, role string) (string, error) {
	args := []any{id}
	where := " where id = $1"
	if tenantID != "" && role != "super_admin" {
		args = append(args, tenantID)
		where += fmt.Sprintf(" and tenant_id = $%d", len(args))
	}
	var key string
	err := s.pool.QueryRow(ctx, "select key from files"+where+" limit 1", args...).Scan(&key)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errNotFound
	}
	return key, err
}

func selectColumns(resource ResourceMeta) string {
	quoted := []string{}
	for _, name := range listFieldNames(resource) {
		quoted = append(quoted, quoteIdent(name))
	}
	return strings.Join(quoted, ", ")
}

func listFieldNames(resource ResourceMeta) []string {
	names := []string{}
	for _, field := range resource.Fields {
		if field.Hidden {
			continue
		}
		names = append(names, field.Name)
	}
	return names
}

func orderClause(resource ResourceMeta, sortBy string) (string, error) {
	if sortBy == "" {
		sortBy = "-created_at"
	}
	desc := strings.HasPrefix(sortBy, "-")
	fieldName := strings.TrimPrefix(sortBy, "-")
	field, ok := fieldByName(resource, fieldName)
	if !ok || !field.Sortable {
		return "", nil
	}
	direction := "asc"
	if desc {
		direction = "desc"
	}
	return " order by " + quoteIdent(fieldName) + " " + direction, nil
}

func fieldByName(resource ResourceMeta, name string) (FieldMeta, bool) {
	for _, field := range resource.Fields {
		if field.Name == name {
			return field, true
		}
	}
	return FieldMeta{}, false
}

func hasField(resource ResourceMeta, name string) bool {
	_, ok := fieldByName(resource, name)
	return ok
}

func normalizeValue(value any) any {
	switch typed := value.(type) {
	case float64:
		return typed
	case map[string]any, []any:
		encoded, _ := json.Marshal(typed)
		return encoded
	default:
		return typed
	}
}

func normalizeScanned(value any) any {
	switch typed := value.(type) {
	case []byte:
		return string(typed)
	default:
		return typed
	}
}

func clone(record Record) Record {
	out := Record{}
	for key, value := range record {
		out[key] = value
	}
	return out
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func quoteIdent(identifier string) string {
	return `"` + strings.ReplaceAll(identifier, `"`, `""`) + `"`
}

func (s *Store) migrate(ctx context.Context) error {
	_, err := s.pool.Exec(ctx, schemaSQL)
	return err
}

func (s *Store) seed(ctx context.Context) error {
	var count int
	if err := s.pool.QueryRow(ctx, "select count(*) from users").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	hash, err := auth.HashPassword("password", auth.DefaultPasswordConfig())
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, seedSQL, hash)
	return err
}
