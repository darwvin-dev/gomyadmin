package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/cache"
)

func TestMemorySessionStoreCreateAndGet(t *testing.T) {
	store := NewMemorySessionStore()
	actor := admin.Actor{ID: "user1", Email: "admin@example.com"}

	session, err := store.Create(context.Background(), actor, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if session.ID == "" {
		t.Fatal("session ID should not be empty")
	}
	if session.Actor.ID != "user1" {
		t.Fatalf("actor ID = %q", session.Actor.ID)
	}

	got, err := store.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != session.ID {
		t.Fatalf("got ID = %q", got.ID)
	}
}

func TestMemorySessionStoreExpiredSessionNotFound(t *testing.T) {
	store := NewMemorySessionStore()
	actor := admin.Actor{ID: "user2"}

	session, err := store.Create(context.Background(), actor, -time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.Get(context.Background(), session.ID)
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestMemorySessionStoreDelete(t *testing.T) {
	store := NewMemorySessionStore()
	session, err := store.Create(context.Background(), admin.Actor{ID: "user3"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	_, err = store.Get(context.Background(), session.ID)
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound after delete, got %v", err)
	}
}

func TestMemorySessionStoreMissingID(t *testing.T) {
	store := NewMemorySessionStore()
	_, err := store.Get(context.Background(), "does-not-exist")
	if err != ErrSessionNotFound {
		t.Fatalf("expected ErrSessionNotFound, got %v", err)
	}
}

func TestSessionManagerStart(t *testing.T) {
	store := NewMemorySessionStore()
	manager := NewSessionManager(store)

	w := httptest.NewRecorder()
	actor := admin.Actor{ID: "user4", Email: "test@example.com"}

	session, err := manager.Start(context.Background(), w, actor)
	if err != nil {
		t.Fatal(err)
	}
	if session.ID == "" {
		t.Fatal("session ID should be set")
	}

	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "gomyadmin_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("session cookie not set")
	}
	if sessionCookie.Value != session.ID {
		t.Fatalf("cookie value = %q, want %q", sessionCookie.Value, session.ID)
	}
}

func TestSessionManagerStartHonorsCookieSettings(t *testing.T) {
	store := NewMemorySessionStore()
	manager := NewSessionManager(store)
	manager.CookieName = "custom_session"
	manager.Secure = true
	manager.SameSite = http.SameSiteStrictMode

	w := httptest.NewRecorder()
	session, err := manager.Start(context.Background(), w, admin.Actor{ID: "user-settings"})
	if err != nil {
		t.Fatal(err)
	}
	cookie := w.Result().Cookies()[0]
	if cookie.Name != "custom_session" || cookie.Value != session.ID {
		t.Fatalf("cookie = %+v, session = %+v", cookie, session)
	}
	if !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteStrictMode {
		t.Fatalf("cookie security settings = %+v", cookie)
	}
}

func TestSessionManagerEndClearsCookieAndDeletesSession(t *testing.T) {
	store := NewMemorySessionStore()
	manager := NewSessionManager(store)
	start := httptest.NewRecorder()
	session, err := manager.Start(context.Background(), start, admin.Actor{ID: "logout-user"})
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	req.AddCookie(&http.Cookie{Name: "gomyadmin_session", Value: session.ID})
	rec := httptest.NewRecorder()
	if err := manager.End(context.Background(), rec, req); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("session should be deleted, err = %v", err)
	}
	cleared := rec.Result().Cookies()[0]
	if cleared.Value != "" || cleared.MaxAge != -1 {
		t.Fatalf("clear cookie = %+v", cleared)
	}
}

func TestSessionManagerEndWithoutCookieStillClearsCookie(t *testing.T) {
	manager := NewSessionManager(NewMemorySessionStore())
	req := httptest.NewRequest(http.MethodPost, "/logout", nil)
	rec := httptest.NewRecorder()
	if err := manager.End(context.Background(), rec, req); err != nil {
		t.Fatal(err)
	}
	if cookies := rec.Result().Cookies(); len(cookies) != 1 || cookies[0].MaxAge != -1 {
		t.Fatalf("cookies = %+v", cookies)
	}
}

func TestSessionManagerMiddlewareRejectsNoSession(t *testing.T) {
	store := NewMemorySessionStore()
	manager := NewSessionManager(store)

	called := false
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.Header.Set("X-Request-ID", "req-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("handler should not be called without a session")
	}
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"request_id":"req-123"`) {
		t.Fatalf("response should include request id in meta, body = %s", w.Body.String())
	}
}

func TestSessionManagerMiddlewarePassesValidSession(t *testing.T) {
	store := NewMemorySessionStore()
	manager := NewSessionManager(store)

	w := httptest.NewRecorder()
	actor := admin.Actor{ID: "user5", Email: "valid@example.com"}
	session, _ := manager.Start(context.Background(), w, actor)

	called := false
	var gotActor admin.Actor
	handler := manager.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		gotActor, _ = ActorFromContext(r.Context())
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "gomyadmin_session", Value: session.ID})
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if !called {
		t.Error("handler should be called")
	}
	if gotActor.ID != "user5" {
		t.Errorf("actor ID = %q", gotActor.ID)
	}
}

func TestActorFromContextMissing(t *testing.T) {
	_, ok := ActorFromContext(context.Background())
	if ok {
		t.Error("should not find actor in empty context")
	}
}

func TestContextWithActor(t *testing.T) {
	actor := admin.Actor{ID: "ctx-user", Email: "ctx@example.com"}
	got, ok := ActorFromContext(ContextWithActor(context.Background(), actor))
	if !ok {
		t.Fatal("expected actor in context")
	}
	if got.ID != actor.ID {
		t.Fatalf("actor id = %q", got.ID)
	}
}

func TestCacheSessionStoreRoundTrip(t *testing.T) {
	store := NewCacheSessionStore(cache.NewMemory())
	session, err := store.Create(context.Background(), admin.Actor{ID: "a1", Email: "a@example.com"}, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(context.Background(), session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Actor.Email != "a@example.com" {
		t.Fatalf("email = %q", got.Actor.Email)
	}
	if err := store.Delete(context.Background(), session.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Get(context.Background(), session.ID); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("err = %v", err)
	}
}
