package server

import (
	"context"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

// AdminStore is the database boundary for the drop-in admin server.
//
// Implement this interface to mount GoMyAdmin on an existing Go backend without
// adopting the built-in PostgreSQL adapter. A Store can be backed by pgx,
// database/sql, GORM, sqlc, Ent, Bun, MongoDB, DynamoDB, SQLite, MySQL, or an
// internal service API.
type AdminStore interface {
	HasResource(table string) bool
	Resources() []ResourceMeta
	List(ctx context.Context, table, tenantID, role, search, sortBy string, filters map[string]string, page, perPage int) ([]Record, int, error)
	Get(ctx context.Context, table, id, tenantID, role string) (Record, error)
	Create(ctx context.Context, table, tenantID string, input Record) (Record, error)
	Update(ctx context.Context, table, id, tenantID, role string, input Record) (oldRecord Record, newRecord Record, err error)
	Delete(ctx context.Context, table, id, tenantID, role string) (oldRecord Record, err error)
	RecordAudit(ctx context.Context, event AuditEvent)
	Audit(ctx context.Context, tenantID, role string) ([]AuditEvent, error)
	AddFile(ctx context.Context, record Record) error
	Files(ctx context.Context, tenantID, role string) ([]Record, error)
	FileKey(ctx context.Context, id, tenantID, role string) (string, error)
}

// ResourceMetadataStore is a small helper store for adapters that want to reuse
// GoMyAdmin's resource metadata conversion while implementing persistence
// themselves.
type ResourceMetadataStore struct {
	resources map[string]resourceMeta
}

// NewResourceMetadataStore creates metadata from an admin.App. Custom stores can
// embed or compose this value to implement HasResource and Resources.
func NewResourceMetadataStore(app *admin.App) ResourceMetadataStore {
	if app == nil {
		app = admin.New("Admin")
	}
	return ResourceMetadataStore{resources: appToMeta(app)}
}

func (s ResourceMetadataStore) HasResource(table string) bool {
	_, ok := s.resources[table]
	return ok
}

func (s ResourceMetadataStore) Resources() []ResourceMeta {
	keys := sortedResourceKeys(s.resources)
	out := make([]ResourceMeta, 0, len(keys))
	for _, k := range keys {
		out = append(out, s.resources[k])
	}
	return out
}
