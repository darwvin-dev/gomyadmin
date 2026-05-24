package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

type fakeStore struct {
	ResourceMetadataStore
	deletedIDs []string
}

type testUser struct{}

func (s fakeStore) List(context.Context, string, string, string, string, string, map[string]string, int, int) ([]Record, int, error) {
	return []Record{}, 0, nil
}
func (s fakeStore) Get(context.Context, string, string, string, string) (Record, error) {
	return nil, errNotFound
}
func (s fakeStore) Create(context.Context, string, string, Record) (Record, error) {
	return Record{}, nil
}
func (s fakeStore) Update(context.Context, string, string, string, string, Record) (Record, Record, error) {
	return Record{}, Record{}, nil
}
func (s fakeStore) Delete(context.Context, string, string, string, string) (Record, error) {
	return Record{}, nil
}
func (s fakeStore) DeleteMany(_ context.Context, _ string, ids []string, _, _ string) ([]Record, error) {
	records := make([]Record, 0, len(ids))
	for _, id := range ids {
		records = append(records, Record{"id": id})
	}
	return records, nil
}
func (s fakeStore) RecordAudit(context.Context, AuditEvent) {}
func (s fakeStore) Audit(context.Context, string, string) ([]AuditEvent, error) {
	return []AuditEvent{}, nil
}

func TestBulkDeleteUsesCustomStore(t *testing.T) {
	app := admin.New("Acme")
	app.Resource(testUser{}).TableName("users").Field("ID").String().Primary()

	sessionStore := auth.NewMemorySessionStore()
	session, err := sessionStore.Create(context.Background(), admin.Actor{
		ID:          "actor_1",
		Email:       "admin@example.com",
		Roles:       []string{"super_admin"},
		Permissions: []string{"users.delete"},
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        fakeStore{ResourceMetadataStore: NewResourceMetadataStore(app)},
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatal(err)
	}

	body, _ := json.Marshal(map[string]any{"ids": []string{"u1", "u1", "u2", ""}})
	req := httptest.NewRequest(http.MethodPost, "/admin/api/users/bulk-delete", bytes.NewReader(body))
	req.AddCookie(&http.Cookie{Name: "gomyadmin_session", Value: session.ID})
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"deleted":2`)) {
		t.Fatalf("expected deleted count, body = %s", rec.Body.String())
	}
}
func (s fakeStore) AddFile(context.Context, Record) error { return nil }
func (s fakeStore) Files(context.Context, string, string) ([]Record, error) {
	return []Record{}, nil
}
func (s fakeStore) FileKey(context.Context, string, string, string) (string, error) {
	return "", errNotFound
}

func TestNewWithCustomStoreDoesNotRequirePostgres(t *testing.T) {
	app := admin.New("Acme")
	app.Resource(testUser{}).TableName("users").Field("ID").String().Primary()

	sessionStore := auth.NewMemorySessionStore()
	session, err := sessionStore.Create(context.Background(), admin.Actor{
		ID:          "actor_1",
		Email:       "admin@example.com",
		Roles:       []string{"super_admin"},
		Permissions: []string{"*"},
	}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}

	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        fakeStore{ResourceMetadataStore: NewResourceMetadataStore(app)},
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/admin/api/resources", nil)
	req.AddCookie(&http.Cookie{Name: "gomyadmin_session", Value: session.ID})
	rec := httptest.NewRecorder()

	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
}
