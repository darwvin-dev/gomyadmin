package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/darwvin/gomyadmin/pkg/admin"
)

var ErrSessionNotFound = errors.New("session not found")

type sessionContextKey string

const actorKey sessionContextKey = "gomyadmin.actor"

type Session struct {
	ID        string
	Actor     admin.Actor
	ExpiresAt time.Time
	CreatedAt time.Time
}

type SessionStore interface {
	Create(ctx context.Context, actor admin.Actor, ttl time.Duration) (Session, error)
	Get(ctx context.Context, id string) (Session, error)
	Delete(ctx context.Context, id string) error
}

type MemorySessionStore struct {
	mu       sync.RWMutex
	sessions map[string]Session
}

func NewMemorySessionStore() *MemorySessionStore {
	return &MemorySessionStore{sessions: map[string]Session{}}
}

func (s *MemorySessionStore) Create(_ context.Context, actor admin.Actor, ttl time.Duration) (Session, error) {
	id, err := secureToken(32)
	if err != nil {
		return Session{}, err
	}
	session := Session{
		ID:        id,
		Actor:     actor,
		ExpiresAt: time.Now().UTC().Add(ttl),
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = session
	return session, nil
}

func (s *MemorySessionStore) Get(_ context.Context, id string) (Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[id]
	if !ok || time.Now().UTC().After(session.ExpiresAt) {
		return Session{}, ErrSessionNotFound
	}
	return session, nil
}

func (s *MemorySessionStore) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, id)
	return nil
}

type SessionManager struct {
	Store      SessionStore
	CookieName string
	TTL        time.Duration
	Secure     bool
	SameSite   http.SameSite
}

func NewSessionManager(store SessionStore) *SessionManager {
	return &SessionManager{
		Store:      store,
		CookieName: "gomyadmin_session",
		TTL:        12 * time.Hour,
		SameSite:   http.SameSiteLaxMode,
	}
}

func (m *SessionManager) Start(ctx context.Context, w http.ResponseWriter, actor admin.Actor) (Session, error) {
	session, err := m.Store.Create(ctx, actor, m.TTL)
	if err != nil {
		return Session{}, err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.CookieName,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: m.SameSite,
	})
	return session, nil
}

func (m *SessionManager) End(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(m.CookieName)
	if err == nil {
		_ = m.Store.Delete(ctx, cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.CookieName,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: m.SameSite,
	})
	return nil
}

func (m *SessionManager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie(m.CookieName)
		if err != nil || cookie.Value == "" {
			admin.WriteError(w, http.StatusUnauthorized, requestID(r), "UNAUTHENTICATED", "Authentication required", nil)
			return
		}
		session, err := m.Store.Get(r.Context(), cookie.Value)
		if err != nil {
			admin.WriteError(w, http.StatusUnauthorized, requestID(r), "UNAUTHENTICATED", "Authentication required", nil)
			return
		}
		ctx := context.WithValue(r.Context(), actorKey, session.Actor)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ActorFromContext(ctx context.Context) (admin.Actor, bool) {
	actor, ok := ctx.Value(actorKey).(admin.Actor)
	return actor, ok
}

func secureToken(size int) (string, error) {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func requestID(r *http.Request) string {
	return r.Header.Get("X-Request-ID")
}
