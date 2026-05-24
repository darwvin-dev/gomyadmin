// Package server provides a drop-in HTTP handler for a GoMyAdmin admin panel.
//
// Any existing Go HTTP server can mount an admin panel in three steps:
//
//  1. Define your resources with admin.App.
//  2. Create an AdminServer with New.
//  3. Mount the handler returned by Handler().
//
// Example:
//
//	app := admin.New("My App")
//	app.Resource(User{}).Label("Users").Field("ID").UUID().Primary().Readonly().
//	    Field("Email").Email().Required().Searchable()
//
//	srv, err := server.New(ctx, server.Config{
//	    DatabaseURL:  os.Getenv("DATABASE_URL"),
//	    App:          app,
//	    Authenticate: myAuthFunc,
//	})
//	if err != nil { log.Fatal(err) }
//	defer srv.Close()
//
//	mux.Handle("/admin/", srv.Handler())
//
// Existing applications can pass Config.Store and Config.SessionStore to use
// their own database, ORM, service layer, Redis, Memcached, or cache system.
package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/darwvin-dev/gomyadmin/pkg/storage"
)

// Config holds all options for an AdminServer.
type Config struct {
	// DatabaseURL is a PostgreSQL connection string (e.g. "postgres://user:pass@host/db").
	// Mutually exclusive with Pool.
	DatabaseURL string

	// Pool is a pre-existing *pgxpool.Pool.
	// Mutually exclusive with DatabaseURL.
	// The caller remains responsible for closing it; AdminServer.Close() will not close it.
	Pool *pgxpool.Pool

	// Store is a custom persistence adapter.
	// Supply this when your application already uses another database, ORM, or
	// service layer. When Store is set, DatabaseURL and Pool are optional and no
	// PostgreSQL migrations are run by GoMyAdmin.
	Store AdminStore

	// SessionStore overrides the built-in PostgreSQL session store.
	// Use this for Redis, Memcached, database/sql, or an existing cache layer.
	// When omitted with DatabaseURL/Pool, PostgreSQL sessions are used. When
	// omitted with Store-only configuration, an in-memory session store is used.
	SessionStore auth.SessionStore

	// App is the resource registry built with admin.New and app.Resource().
	// An empty App is created automatically when nil.
	App *admin.App

	// Authenticate is called on every login attempt.
	// Return (actor, true, nil) on success or (zero, false, nil) on bad credentials.
	// When nil, all login attempts return 401 Unauthorized.
	Authenticate func(ctx context.Context, email, password string) (admin.Actor, bool, error)

	// Tenants is an optional callback that returns the list of tenants the actor
	// may switch to. The response appears in the "me" endpoint under "tenants".
	// When nil, an empty list is returned.
	Tenants func(ctx context.Context, actor admin.Actor) ([]map[string]string, error)

	// Log is the structured logger. Defaults to JSON on stderr.
	Log *slog.Logger

	// UploadDir is the local directory used by the built-in file upload handler.
	// Defaults to "tmp/uploads". Supply a storage.Storage via the Uploads field
	// instead to use S3, R2, or MinIO.
	UploadDir string

	// Uploads overrides the default local file storage adapter.
	// Set this to use S3, R2, or MinIO for file uploads.
	Uploads storage.Storage

	// PublicURL is the externally reachable base URL of this server.
	// Used as the Access-Control-Allow-Origin header value.
	// Defaults to GOMYADMIN_PUBLIC_URL env var, then "http://localhost:8080".
	PublicURL string

	// ownPool is true when New() created the pool; Close() will close it.
	ownPool bool
}

// AdminServer is a self-contained HTTP handler that serves a GoMyAdmin admin panel.
// Use New to create one, Handler to get the http.Handler, and Close when done.
type AdminServer struct {
	cfg      Config
	log      *slog.Logger
	app      *admin.App
	store    AdminStore
	sessions *auth.SessionManager
	uploads  storage.Storage
}

// New creates an AdminServer.
//
// With DatabaseURL or Pool, it uses the built-in PostgreSQL adapter and runs
// GoMyAdmin's internal migrations. With Config.Store, it uses the caller's
// adapter and does not require PostgreSQL.
func New(ctx context.Context, cfg Config) (*AdminServer, error) {
	if cfg.Store == nil && cfg.DatabaseURL == "" && cfg.Pool == nil {
		return nil, errors.New("server.Config: Store, DatabaseURL, or Pool is required")
	}
	if cfg.DatabaseURL != "" && cfg.Pool != nil {
		return nil, errors.New("server.Config: supply DatabaseURL or Pool, not both")
	}
	if cfg.App == nil {
		cfg.App = admin.New("Admin")
	}
	if cfg.Log == nil {
		cfg.Log = slog.New(slog.NewJSONHandler(os.Stderr, nil))
	}
	if cfg.UploadDir == "" {
		cfg.UploadDir = "tmp/uploads"
	}
	if cfg.PublicURL == "" {
		cfg.PublicURL = envOr("GOMYADMIN_PUBLIC_URL", "http://localhost:8080")
	}

	var pool *pgxpool.Pool
	var st AdminStore
	if cfg.Store != nil {
		st = cfg.Store
	} else {
		pool = cfg.Pool
		if pool == nil {
			var err error
			pool, err = pgxpool.New(ctx, cfg.DatabaseURL)
			if err != nil {
				return nil, err
			}
			cfg.ownPool = true
			cfg.Pool = pool
		}

		st = newServerStore(pool, cfg.App)
		if err := st.(*serverStore).migrate(ctx); err != nil {
			if cfg.ownPool {
				pool.Close()
			}
			return nil, err
		}
	}

	uploads := cfg.Uploads
	if uploads == nil {
		uploads = storage.NewLocal(cfg.UploadDir, cfg.PublicURL+"/admin/api/files")
	}

	sessionStore := cfg.SessionStore
	if sessionStore == nil {
		if pool != nil {
			pgSessions := auth.NewPGSessionStore(pool)
			if err := pgSessions.Migrate(ctx); err != nil {
				if cfg.ownPool {
					pool.Close()
				}
				return nil, err
			}
			sessionStore = pgSessions
		} else {
			sessionStore = auth.NewMemorySessionStore()
		}
	}

	return &AdminServer{
		cfg:      cfg,
		log:      cfg.Log,
		app:      cfg.App,
		store:    st,
		sessions: auth.NewSessionManager(sessionStore),
		uploads:  uploads,
	}, nil
}

// Handler returns an http.Handler that serves all admin API routes under /admin/api/.
// Mount it at your mux root so the /admin/ prefix is routed here:
//
//	mux.Handle("/admin/", adminServer.Handler())
//
// The handler is safe to call multiple times; it returns the same router each call.
func (s *AdminServer) Handler() http.Handler {
	return s.buildRouter()
}

// Close releases the database pool when AdminServer created it (DatabaseURL was used).
// When you supplied your own Pool, close it yourself — AdminServer will not close it.
func (s *AdminServer) Close() {
	if s.cfg.ownPool && s.cfg.Pool != nil {
		s.cfg.Pool.Close()
	}
}

// App returns the admin.App resource registry so callers can inspect or add resources
// after construction (before any requests are served).
func (s *AdminServer) App() *admin.App {
	return s.app
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// compile-time assertion
var _ interface {
	Handler() http.Handler
} = (*AdminServer)(nil)
