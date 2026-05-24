# Drop-In Adapters

GoMyAdmin can be mounted on an existing Go backend without changing your router, ORM, database, or cache.

## Minimal Mount

```go
app := admin.New("Acme Admin")
app.Resource(User{}).
	Label("Users").
	TableName("users").
	Field("ID").String().Primary().Readonly().
	Field("Email").Email().Searchable().Sortable()

adminServer, err := server.New(ctx, server.Config{
	App:          app,
	Store:        myStore,
	SessionStore: mySessionStore,
	Authenticate: authenticateAdmin,
})
if err != nil {
	return err
}

mux.Handle("/admin/", adminServer.Handler())
```

`myStore` can use any persistence layer: `database/sql`, pgx, GORM, Ent, sqlc, Bun, MongoDB, SQLite, MySQL, Postgres, DynamoDB, or an internal service API.

`mySessionStore` can use any cache/session backend: Redis, Memcached, PostgreSQL, MySQL, SQLite, in-memory, or your existing auth service.

For generic cache-backed adapters, use the small `pkg/cache.Cache` contract or adapt your existing cache client directly.

## Database Adapter Boundary

Implement `server.AdminStore`:

```go
type AdminStore interface {
	HasResource(table string) bool
	Resources() []server.ResourceMeta
	List(ctx context.Context, table, tenantID, role, search, sortBy string, filters map[string]string, page, perPage int) ([]server.Record, int, error)
	Get(ctx context.Context, table, id, tenantID, role string) (server.Record, error)
	Create(ctx context.Context, table, tenantID string, input server.Record) (server.Record, error)
	Update(ctx context.Context, table, id, tenantID, role string, input server.Record) (server.Record, server.Record, error)
	Delete(ctx context.Context, table, id, tenantID, role string) (server.Record, error)
	RecordAudit(ctx context.Context, event server.AuditEvent)
	Audit(ctx context.Context, tenantID, role string) ([]server.AuditEvent, error)
	AddFile(ctx context.Context, record server.Record) error
	Files(ctx context.Context, tenantID, role string) ([]server.Record, error)
	FileKey(ctx context.Context, id, tenantID, role string) (string, error)
}
```

Use `server.NewResourceMetadataStore(app)` to reuse GoMyAdmin's resource metadata conversion:

```go
type GormAdminStore struct {
	server.ResourceMetadataStore
	db *gorm.DB
}

func NewGormAdminStore(app *admin.App, db *gorm.DB) *GormAdminStore {
	return &GormAdminStore{
		ResourceMetadataStore: server.NewResourceMetadataStore(app),
		db: db,
	}
}
```

Then implement CRUD using your ORM's query API.

## Session and Cache Boundary

Implement `auth.SessionStore`:

```go
type SessionStore interface {
	Create(ctx context.Context, actor admin.Actor, ttl time.Duration) (auth.Session, error)
	Get(ctx context.Context, id string) (auth.Session, error)
	Delete(ctx context.Context, id string) error
}
```

Redis example shape:

```go
type RedisSessionStore struct {
	client *redis.Client
}

func (s *RedisSessionStore) Create(ctx context.Context, actor admin.Actor, ttl time.Duration) (auth.Session, error) {
	session := auth.Session{
		ID:        randomToken(),
		Actor:     actor,
		CreatedAt: time.Now().UTC(),
		ExpiresAt: time.Now().UTC().Add(ttl),
	}
	data, err := json.Marshal(session)
	if err != nil {
		return auth.Session{}, err
	}
	return session, s.client.Set(ctx, "gomyadmin:session:"+session.ID, data, ttl).Err()
}
```

## Generic Cache Boundary

For adapter code that needs simple cache semantics, `pkg/cache` exposes:

```go
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}
```

`cache.NewMemory()` is included for tests and single-process development. Production systems should wrap Redis, Memcached, Dragonfly, KeyDB, or an existing internal cache client.

## Built-In Defaults

If you pass `DatabaseURL` or `Pool`, GoMyAdmin uses the built-in PostgreSQL adapter.

If you pass `Store` without `SessionStore`, GoMyAdmin uses in-memory sessions. That is useful for local development, but production deployments should pass Redis, SQL, or another shared session store.

## Compatibility Target

The public adapter boundary is intentionally small:

- HTTP framework independent: mount on `net/http`, chi, gorilla/mux, echo, fiber adapters, or existing reverse proxies.
- Database independent: implement `AdminStore`.
- Cache/session independent: implement `auth.SessionStore`.
- Generic cache independent: implement `cache.Cache`.
- File storage independent: pass `storage.Storage`.
- Auth independent: pass `Config.Authenticate`.
