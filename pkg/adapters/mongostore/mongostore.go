// Package mongostore provides a MongoDB-backed AdminStore adapter.
package mongostore

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Store struct {
	server.ResourceMetadataStore
	db        *mongo.Database
	resources map[string]server.ResourceMeta
}

func New(db *mongo.Database, app *admin.App) *Store {
	meta := server.NewResourceMetadataStore(app)
	resources := map[string]server.ResourceMeta{}
	for _, resource := range meta.Resources() {
		resources[resource.Table] = resource
	}
	return &Store{ResourceMetadataStore: meta, db: db, resources: resources}
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
	filter, err := buildFilter(resource, tenantID, role, search, filters)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.db.Collection(resource.Table).CountDocuments(ctx, filter)
	if err != nil {
		return nil, 0, err
	}
	findOptions := options.Find().SetLimit(int64(perPage)).SetSkip(int64((page - 1) * perPage))
	if sort := buildSort(resource, sortBy); sort != nil {
		findOptions.SetSort(sort)
	}
	cursor, err := s.db.Collection(resource.Table).Find(ctx, filter, findOptions)
	if err != nil {
		return nil, 0, err
	}
	defer cursor.Close(ctx)
	records := []server.Record{}
	for cursor.Next(ctx) {
		var record server.Record
		if err := cursor.Decode(&record); err != nil {
			return nil, 0, err
		}
		records = append(records, normalizeRecord(record))
	}
	return records, int(total), cursor.Err()
}

func (s *Store) Get(ctx context.Context, table, id, tenantID, role string) (server.Record, error) {
	resource, ok := s.resources[table]
	if !ok {
		return nil, server.ErrNotFound
	}
	filter := bson.M{resource.PrimaryKey: id}
	if resource.TenantKey != "" && role != "super_admin" {
		filter[resource.TenantKey] = tenantID
	}
	var record server.Record
	err := s.db.Collection(resource.Table).FindOne(ctx, filter).Decode(&record)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, server.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return normalizeRecord(record), nil
}

func (s *Store) Create(ctx context.Context, table, tenantID string, input server.Record) (server.Record, error) {
	resource, ok := s.resources[table]
	if !ok {
		return nil, server.ErrNotFound
	}
	record := clone(input)
	if record[resource.PrimaryKey] == nil {
		record[resource.PrimaryKey] = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	if resource.TenantKey != "" {
		record[resource.TenantKey] = tenantID
	}
	now := time.Now().UTC()
	if hasField(resource, "created_at") && record["created_at"] == nil {
		record["created_at"] = now
	}
	if hasField(resource, "updated_at") && record["updated_at"] == nil {
		record["updated_at"] = now
	}
	if _, err := s.db.Collection(resource.Table).InsertOne(ctx, record); err != nil {
		return nil, err
	}
	return s.Get(ctx, table, fmt.Sprint(record[resource.PrimaryKey]), tenantID, "super_admin")
}

func (s *Store) Update(ctx context.Context, table, id, tenantID, role string, input server.Record) (server.Record, server.Record, error) {
	oldRecord, err := s.Get(ctx, table, id, tenantID, role)
	if err != nil {
		return nil, nil, err
	}
	resource := s.resources[table]
	set := bson.M{}
	for _, field := range resource.Fields {
		if field.Readonly || field.Hidden || field.SQLName == resource.PrimaryKey || field.SQLName == resource.TenantKey {
			continue
		}
		if value, ok := input[field.SQLName]; ok {
			set[field.SQLName] = value
		}
	}
	if hasField(resource, "updated_at") {
		set["updated_at"] = time.Now().UTC()
	}
	if len(set) == 0 {
		return oldRecord, oldRecord, nil
	}
	filter := bson.M{resource.PrimaryKey: id}
	if resource.TenantKey != "" && role != "super_admin" {
		filter[resource.TenantKey] = tenantID
	}
	if _, err := s.db.Collection(resource.Table).UpdateOne(ctx, filter, bson.M{"$set": set}); err != nil {
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
	deleted := []server.Record{}
	for _, id := range ids {
		oldRecord, err := s.Get(ctx, table, id, tenantID, role)
		if err != nil {
			return nil, err
		}
		resource := s.resources[table]
		filter := bson.M{resource.PrimaryKey: id}
		if resource.TenantKey != "" && role != "super_admin" {
			filter[resource.TenantKey] = tenantID
		}
		if _, err := s.db.Collection(resource.Table).DeleteOne(ctx, filter); err != nil {
			return nil, err
		}
		deleted = append(deleted, oldRecord)
	}
	return deleted, nil
}

func (s *Store) RecordAudit(ctx context.Context, event server.AuditEvent) {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	if event.ID == "" {
		event.ID = fmt.Sprintf("audit_%d", time.Now().UnixNano())
	}
	_, _ = s.db.Collection("gomyadmin_audit_logs").InsertOne(ctx, event)
}

func (s *Store) Audit(ctx context.Context, tenantID, role string) ([]server.AuditEvent, error) {
	filter := bson.M{}
	if tenantID != "" && role != "super_admin" {
		filter["tenantid"] = tenantID
	}
	cursor, err := s.db.Collection("gomyadmin_audit_logs").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "createdat", Value: -1}}).SetLimit(100))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	events := []server.AuditEvent{}
	for cursor.Next(ctx) {
		var event server.AuditEvent
		if err := cursor.Decode(&event); err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	return events, cursor.Err()
}

func (s *Store) AddFile(ctx context.Context, record server.Record) error {
	if record["id"] == nil {
		record["id"] = fmt.Sprintf("file_%d", time.Now().UnixNano())
	}
	if record["created_at"] == nil {
		record["created_at"] = time.Now().UTC()
	}
	_, err := s.db.Collection("gomyadmin_files").InsertOne(ctx, record)
	return err
}

func (s *Store) Files(ctx context.Context, tenantID, role string) ([]server.Record, error) {
	filter := bson.M{}
	if tenantID != "" && role != "super_admin" {
		filter["tenant_id"] = tenantID
	}
	cursor, err := s.db.Collection("gomyadmin_files").Find(ctx, filter, options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}))
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)
	records := []server.Record{}
	for cursor.Next(ctx) {
		var record server.Record
		if err := cursor.Decode(&record); err != nil {
			return nil, err
		}
		records = append(records, normalizeRecord(record))
	}
	return records, cursor.Err()
}

func (s *Store) FileKey(ctx context.Context, id, tenantID, role string) (string, error) {
	filter := bson.M{"id": id}
	if tenantID != "" && role != "super_admin" {
		filter["tenant_id"] = tenantID
	}
	var record server.Record
	err := s.db.Collection("gomyadmin_files").FindOne(ctx, filter).Decode(&record)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", server.ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprint(record["key"]), nil
}

func buildFilter(resource server.ResourceMeta, tenantID, role, search string, filters map[string]string) (bson.M, error) {
	filter := bson.M{}
	if resource.TenantKey != "" && role != "super_admin" {
		filter[resource.TenantKey] = tenantID
	}
	if search != "" {
		or := bson.A{}
		for _, field := range resource.Fields {
			if field.Searchable && !field.Hidden {
				or = append(or, bson.M{field.SQLName: bson.M{"$regex": regexp.QuoteMeta(search), "$options": "i"}})
			}
		}
		if len(or) > 0 {
			filter["$or"] = or
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
			return nil, fmt.Errorf("field %q is not filterable", fieldName)
		}
		switch operator {
		case "eq":
			filter[fieldName] = value
		case "contains":
			filter[fieldName] = bson.M{"$regex": regexp.QuoteMeta(value), "$options": "i"}
		case "starts_with":
			filter[fieldName] = bson.M{"$regex": "^" + regexp.QuoteMeta(value), "$options": "i"}
		case "ends_with":
			filter[fieldName] = bson.M{"$regex": regexp.QuoteMeta(value) + "$", "$options": "i"}
		case "gte":
			filter[fieldName] = bson.M{"$gte": value}
		case "lte":
			filter[fieldName] = bson.M{"$lte": value}
		case "gt":
			filter[fieldName] = bson.M{"$gt": value}
		case "lt":
			filter[fieldName] = bson.M{"$lt": value}
		default:
			return nil, fmt.Errorf("unsupported filter operator %q", operator)
		}
	}
	return filter, nil
}

func buildSort(resource server.ResourceMeta, sortBy string) bson.D {
	if sortBy == "" {
		sortBy = resource.DefaultSort
	}
	desc := strings.HasPrefix(sortBy, "-")
	fieldName := strings.TrimPrefix(sortBy, "-")
	field, ok := fieldBySQLName(resource, fieldName)
	if !ok || !field.Sortable {
		return nil
	}
	if desc {
		return bson.D{{Key: fieldName, Value: -1}}
	}
	return bson.D{{Key: fieldName, Value: 1}}
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

func normalizeRecord(record server.Record) server.Record {
	if id, ok := record["_id"]; ok && record["id"] == nil {
		record["id"] = fmt.Sprint(id)
	}
	delete(record, "_id")
	return record
}

func clone(record server.Record) server.Record {
	out := server.Record{}
	for k, v := range record {
		out[k] = v
	}
	return out
}

var _ server.AdminStore = (*Store)(nil)
