package audit

import (
	"context"
	"sync"
	"time"
)

type Event struct {
	ID         string         `json:"id"`
	ActorID    string         `json:"actor_id"`
	ActorEmail string         `json:"actor_email"`
	TenantID   string         `json:"tenant_id"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	OldValues  map[string]any `json:"old_values,omitempty"`
	NewValues  map[string]any `json:"new_values,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	IPAddress  string         `json:"ip_address"`
	UserAgent  string         `json:"user_agent"`
	RequestID  string         `json:"request_id"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Query struct {
	ActorID  string
	TenantID string
	Resource string
	Action   string
	From     time.Time
	To       time.Time
	Limit    int
}

type Writer interface {
	Record(ctx context.Context, event Event) error
}

type Reader interface {
	List(ctx context.Context, query Query) ([]Event, error)
}

type Store interface {
	Writer
	Reader
}

type MemoryStore struct {
	mu     sync.RWMutex
	events []Event
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (s *MemoryStore) Record(_ context.Context, event Event) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
	return nil
}

func (s *MemoryStore) List(_ context.Context, query Query) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	limit := query.Limit
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	out := make([]Event, 0, limit)
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		event := s.events[i]
		if query.ActorID != "" && event.ActorID != query.ActorID {
			continue
		}
		if query.TenantID != "" && event.TenantID != query.TenantID {
			continue
		}
		if query.Resource != "" && event.Resource != query.Resource {
			continue
		}
		if query.Action != "" && event.Action != query.Action {
			continue
		}
		if !query.From.IsZero() && event.CreatedAt.Before(query.From) {
			continue
		}
		if !query.To.IsZero() && event.CreatedAt.After(query.To) {
			continue
		}
		out = append(out, event)
	}
	return out, nil
}
