package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

// recordStore is a fake AdminStore that stores one record in memory.
type recordStore struct {
	ResourceMetadataStore
	record Record
}

type fakeAPIKeys struct {
	actor admin.Actor
	keys  []auth.APIKey
}

func (f *fakeAPIKeys) Authenticate(_ context.Context, rawKey string) (admin.Actor, auth.APIKey, bool, error) {
	if rawKey != "gma_test_secret" {
		return admin.Actor{}, auth.APIKey{}, false, nil
	}
	return f.actor, auth.APIKey{ID: "key1", Name: "Test", Prefix: "test", Actor: f.actor}, true, nil
}

func (f *fakeAPIKeys) Create(_ context.Context, input auth.CreateAPIKeyInput) (auth.APIKey, string, error) {
	key := auth.APIKey{ID: "key1", Name: input.Name, Prefix: "test", Scopes: input.Scopes, Actor: input.Actor, CreatedAt: time.Now().UTC()}
	f.keys = append(f.keys, key)
	return key, "gma_test_secret", nil
}

func (f *fakeAPIKeys) List(_ context.Context, _ admin.Actor) ([]auth.APIKey, error) {
	return f.keys, nil
}

func (f *fakeAPIKeys) Revoke(_ context.Context, id string, _ admin.Actor) error {
	for i := range f.keys {
		if f.keys[i].ID == id {
			now := time.Now().UTC()
			f.keys[i].RevokedAt = &now
			return nil
		}
	}
	return auth.ErrAPIKeyNotFound
}

func (s *recordStore) List(_ context.Context, _ string, _, _, _, _ string, _ map[string]string, _, _ int) ([]Record, int, error) {
	if s.record == nil {
		return []Record{}, 0, nil
	}
	return []Record{s.record}, 1, nil
}
func (s *recordStore) Get(_ context.Context, _, id, _, _ string) (Record, error) {
	if s.record == nil || s.record["id"] != id {
		return nil, errNotFound
	}
	return s.record, nil
}
func (s *recordStore) Create(_ context.Context, _, _ string, input Record) (Record, error) {
	if input["id"] == nil {
		input["id"] = "created-id"
	}
	s.record = input
	return s.record, nil
}
func (s *recordStore) Update(_ context.Context, _, _, _, _ string, input Record) (Record, Record, error) {
	old := s.record
	for k, v := range input {
		s.record[k] = v
	}
	return old, s.record, nil
}
func (s *recordStore) Delete(_ context.Context, _, id, _, _ string) (Record, error) {
	if s.record == nil || s.record["id"] != id {
		return nil, errNotFound
	}
	old := s.record
	s.record = nil
	return old, nil
}
func (s *recordStore) DeleteMany(_ context.Context, _ string, ids []string, _, _ string) ([]Record, error) {
	out := make([]Record, 0, len(ids))
	for _, id := range ids {
		out = append(out, Record{"id": id})
	}
	return out, nil
}
func (s *recordStore) RecordAudit(context.Context, AuditEvent) {}
func (s *recordStore) Audit(context.Context, string, string) ([]AuditEvent, error) {
	return []AuditEvent{{ID: "ev1", Action: "create", Resource: "users"}}, nil
}
func (s *recordStore) AddFile(context.Context, Record) error { return nil }
func (s *recordStore) Files(context.Context, string, string) ([]Record, error) {
	return []Record{}, nil
}
func (s *recordStore) FileKey(context.Context, string, string, string) (string, error) {
	return "", errNotFound
}

// newTestServer builds an AdminServer with a fresh fake store and a session
// for the given actor. Returns the server and the session cookie value.
func newTestServer(t *testing.T, actor admin.Actor, record Record) (*AdminServer, string) {
	t.Helper()
	app := admin.New("Test")
	app.Resource(testUser{}).
		TableName("users").
		Field("ID").String().Primary().
		Field("Name").String().Searchable()

	sessionStore := auth.NewMemorySessionStore()
	session, err := sessionStore.Create(context.Background(), actor, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	st := &recordStore{
		ResourceMetadataStore: NewResourceMetadataStore(app),
		record:                record,
	}
	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        st,
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatal(err)
	}
	return srv, session.ID
}

func superActor() admin.Actor {
	return admin.Actor{
		ID:          "actor-1",
		Email:       "admin@example.com",
		Roles:       []string{"super_admin"},
		Permissions: []string{"*"},
	}
}

func cookie(sessionID string) *http.Cookie {
	return &http.Cookie{Name: "gomyadmin_session", Value: sessionID}
}

func TestNormalizeScannedFormatsBinaryUUID(t *testing.T) {
	got := normalizeScanned([16]byte{0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef})
	if got != "12345678-90ab-cdef-1234-567890abcdef" {
		t.Fatalf("normalizeScanned UUID = %#v", got)
	}
}

func do(t *testing.T, srv *AdminServer, method, path string, body any, sessionID string) *httptest.ResponseRecorder {
	t.Helper()
	var bodyReader *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewReader(b)
	} else {
		bodyReader = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, bodyReader)
	if sessionID != "" {
		req.AddCookie(cookie(sessionID))
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

// --- unauthenticated ---

func TestHandlerUnauthenticatedReturns401(t *testing.T) {
	srv, _ := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/resources", nil, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// --- /admin/api/resources ---

func TestHandleResourcesReturnsRegisteredResources(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/resources", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("users")) {
		t.Fatalf("expected 'users' in resources, body = %s", rec.Body)
	}
}

// --- /admin/api/me ---

func TestHandleMeReturnsActor(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/me", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("admin@example.com")) {
		t.Fatalf("expected actor email, body = %s", rec.Body)
	}
}

func TestHandleResourcesWithAPIKeyReturnsRegisteredResources(t *testing.T) {
	srv, _ := newTestServer(t, superActor(), nil)
	srv.apiKeys = &fakeAPIKeys{actor: superActor()}
	req := httptest.NewRequest(http.MethodGet, "/admin/api/resources", nil)
	req.Header.Set("Authorization", "Bearer gma_test_secret")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCreateAndListAPIKeys(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	srv.apiKeys = &fakeAPIKeys{actor: superActor()}

	createRec := do(t, srv, http.MethodPost, "/admin/api/auth/api-keys", map[string]any{
		"name":       "Automation",
		"scopes":     []string{"users.view"},
		"expires_in": "24h",
	}, sessionID)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", createRec.Code, createRec.Body.String())
	}
	if !bytes.Contains(createRec.Body.Bytes(), []byte("gma_test_secret")) {
		t.Fatalf("expected secret in response, body = %s", createRec.Body.String())
	}

	listRec := do(t, srv, http.MethodGet, "/admin/api/auth/api-keys", nil, sessionID)
	if listRec.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRec.Code, listRec.Body.String())
	}
	if !bytes.Contains(listRec.Body.Bytes(), []byte("Automation")) {
		t.Fatalf("expected created api key in list, body = %s", listRec.Body.String())
	}
}

func TestHandleAuthProvidersReturnsConfiguredProviders(t *testing.T) {
	srv, _ := newTestServer(t, superActor(), nil)
	srv.cfg.OAuthProviders = map[string]auth.OAuthProvider{
		"google": auth.GoogleOAuthProvider("client", "secret"),
	}
	rec := do(t, srv, http.MethodGet, "/admin/api/auth/providers", nil, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("google")) {
		t.Fatalf("expected provider in response, body = %s", rec.Body.String())
	}
}

func TestHandleOAuthStartRedirectsToProvider(t *testing.T) {
	srv, _ := newTestServer(t, superActor(), nil)
	srv.cfg.PublicURL = "http://localhost:8080"
	srv.cfg.OAuthProviders = map[string]auth.OAuthProvider{
		"google": auth.GoogleOAuthProvider("client", "secret"),
	}
	req := httptest.NewRequest(http.MethodGet, "/admin/api/auth/oauth/google/start", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	location := rec.Header().Get("Location")
	if !strings.Contains(location, "accounts.google.com") {
		t.Fatalf("expected oauth redirect, got %q", location)
	}
}

// --- /admin/api/auth/login ---

func TestHandleLoginNoAuthenticateReturns401(t *testing.T) {
	srv, _ := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodPost, "/admin/api/auth/login",
		map[string]string{"email": "a@b.com", "password": "secret"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandleLoginWithAuthenticateReturns200(t *testing.T) {
	app := admin.New("Test")
	sessionStore := auth.NewMemorySessionStore()
	session, _ := sessionStore.Create(context.Background(), superActor(), time.Hour)

	st := &recordStore{ResourceMetadataStore: NewResourceMetadataStore(app)}
	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        st,
		SessionStore: sessionStore,
		Authenticate: func(_ context.Context, email, _ string) (admin.Actor, bool, error) {
			if email == "admin@example.com" {
				return superActor(), true, nil
			}
			return admin.Actor{}, false, nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = session // session already created; just use the login endpoint
	rec := do(t, srv, http.MethodPost, "/admin/api/auth/login",
		map[string]string{"email": "admin@example.com", "password": "pass"}, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("admin@example.com")) {
		t.Fatalf("expected actor in response, body = %s", rec.Body)
	}
}

func TestHandleLoginBadCredentialsReturns401(t *testing.T) {
	app := admin.New("Test")
	sessionStore := auth.NewMemorySessionStore()
	st := &recordStore{ResourceMetadataStore: NewResourceMetadataStore(app)}
	srv, _ := New(context.Background(), Config{
		App:          app,
		Store:        st,
		SessionStore: sessionStore,
		Authenticate: func(_ context.Context, _, _ string) (admin.Actor, bool, error) {
			return admin.Actor{}, false, nil
		},
	})
	rec := do(t, srv, http.MethodPost, "/admin/api/auth/login",
		map[string]string{"email": "bad@x.com", "password": "wrong"}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

// --- CRUD: list ---

func TestHandleListReturnsRecords(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), Record{"id": "u1", "name": "Alice"})
	rec := do(t, srv, http.MethodGet, "/admin/api/users", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Alice")) {
		t.Fatalf("expected 'Alice' in response, body = %s", rec.Body)
	}
}

func TestHandleListUnknownResourceReturns404(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/nonexistent", nil, sessionID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleListPermissionDenied(t *testing.T) {
	actor := admin.Actor{
		ID:          "actor-2",
		Email:       "viewer@example.com",
		Roles:       []string{"viewer"},
		Permissions: []string{},
	}
	srv, sessionID := newTestServer(t, actor, nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/users", nil, sessionID)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

// --- CRUD: get ---

func TestHandleGetReturns404ForMissingRecord(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/users/no-such-id", nil, sessionID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleGetReturnsRecord(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), Record{"id": "u1", "name": "Bob"})
	rec := do(t, srv, http.MethodGet, "/admin/api/users/u1", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Bob")) {
		t.Fatalf("expected 'Bob', body = %s", rec.Body)
	}
}

// --- CRUD: create ---

func TestHandleCreateReturns201(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodPost, "/admin/api/users",
		map[string]string{"name": "Charlie"}, sessionID)
	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("Charlie")) {
		t.Fatalf("expected 'Charlie', body = %s", rec.Body)
	}
}

func TestHandleCreateBadJSONReturns400(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	req := httptest.NewRequest(http.MethodPost, "/admin/api/users", strings.NewReader("not-json"))
	req.AddCookie(cookie(sessionID))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

// --- CRUD: update ---

func TestHandleUpdateReturns200(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), Record{"id": "u1", "name": "Old"})
	rec := do(t, srv, http.MethodPatch, "/admin/api/users/u1",
		map[string]string{"name": "New"}, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

// --- CRUD: delete ---

func TestHandleDeleteReturns200(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), Record{"id": "u1", "name": "ToDelete"})
	rec := do(t, srv, http.MethodDelete, "/admin/api/users/u1", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"deleted":true`)) {
		t.Fatalf("expected deleted:true, body = %s", rec.Body)
	}
}

func TestHandleDeleteMissingReturns404(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodDelete, "/admin/api/users/ghost", nil, sessionID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// --- export ---

func TestHandleExportReturnsCSV(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), Record{"id": "u1", "name": "Alice"})
	rec := do(t, srv, http.MethodGet, "/admin/api/users/export", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/csv") {
		t.Fatalf("content-type = %q, want text/csv", ct)
	}
}

// --- audit ---

func TestHandleAuditReturnsEvents(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/audit", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("create")) {
		t.Fatalf("expected audit event, body = %s", rec.Body)
	}
}

// --- logout ---

func TestHandleLogoutReturns200(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodPost, "/admin/api/auth/logout", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
}
