package server

import (
	"bytes"
	"context"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

func TestNewRequiresStoreOrDatabaseURL(t *testing.T) {
	_, err := New(context.Background(), Config{})
	if err == nil {
		t.Fatal("expected error when Store, DatabaseURL, and Pool are all missing")
	}
}

func TestHandleUploadFileMissingField(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.WriteField("other", "value")
	w.Close()

	req := httptest.NewRequest(http.MethodPost, "/admin/api/files", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.AddCookie(cookie(sessionID))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for missing file field, body = %s", rec.Code, rec.Body)
	}
}

func TestHandleFilesListReturnsEmpty(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/files", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
}

func TestHandleFileNotFoundReturns404(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/files/no-such-file", nil, sessionID)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestHandleBulkActionReturns200(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodPost, "/admin/api/users/bulk-actions/archive",
		map[string][]string{"ids": {"u1", "u2"}}, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("queued")) {
		t.Fatalf("expected 'queued' in response, body = %s", rec.Body)
	}
}

func TestHandleActionNoHandlerReturns200(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), Record{"id": "u1", "name": "Bob"})
	rec := do(t, srv, http.MethodPost, "/admin/api/users/u1/actions/send-email", nil, sessionID)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte("dispatched")) {
		t.Fatalf("expected 'dispatched' in response, body = %s", rec.Body)
	}
}

func TestHandleAuditPermissionDenied(t *testing.T) {
	actor := admin.Actor{
		ID:          "viewer-1",
		Email:       "viewer@example.com",
		Roles:       []string{"viewer"},
		Permissions: []string{},
	}
	srv, sessionID := newTestServer(t, actor, nil)
	rec := do(t, srv, http.MethodGet, "/admin/api/audit", nil, sessionID)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestHandleBulkDeleteEmptyIDsReturns400(t *testing.T) {
	srv, sessionID := newTestServer(t, superActor(), nil)
	rec := do(t, srv, http.MethodPost, "/admin/api/users/bulk-delete",
		map[string][]string{"ids": {}}, sessionID)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for empty ids", rec.Code)
	}
}

func TestAppReturnsAdminApp(t *testing.T) {
	app := admin.New("Test")
	sessionStore := auth.NewMemorySessionStore()
	st := &recordStore{ResourceMetadataStore: NewResourceMetadataStore(app)}
	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        st,
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatal(err)
	}
	if srv.App() != app {
		t.Fatal("App() should return the configured admin.App")
	}
}

func TestCloseDoesNotPanicWithCustomStore(t *testing.T) {
	app := admin.New("Test")
	sessionStore := auth.NewMemorySessionStore()
	st := &recordStore{ResourceMetadataStore: NewResourceMetadataStore(app)}
	srv, err := New(context.Background(), Config{
		App:          app,
		Store:        st,
		SessionStore: sessionStore,
	})
	if err != nil {
		t.Fatal(err)
	}
	srv.Close() // should not panic when no pool owned
}

func TestHandleLoginInvalidJSONReturns400(t *testing.T) {
	app := admin.New("Test")
	sessionStore := auth.NewMemorySessionStore()
	_, _ = sessionStore.Create(context.Background(), superActor(), time.Hour)
	st := &recordStore{ResourceMetadataStore: NewResourceMetadataStore(app)}
	srv, _ := New(context.Background(), Config{
		App:          app,
		Store:        st,
		SessionStore: sessionStore,
		Authenticate: func(_ context.Context, _, _ string) (admin.Actor, bool, error) {
			return admin.Actor{}, false, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/admin/api/auth/login",
		bytes.NewBufferString("not-json"))
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for invalid JSON", rec.Code)
	}
}
