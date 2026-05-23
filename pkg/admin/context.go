package admin

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const requestContextKey contextKey = "gomyadmin.context"

// Context carries request-scoped admin metadata.
type Context struct {
	RequestID string
	TenantID  string
	Request   *http.Request
}

func WithContext(ctx context.Context, adminContext Context) context.Context {
	return context.WithValue(ctx, requestContextKey, adminContext)
}

func FromContext(ctx context.Context) (Context, bool) {
	value, ok := ctx.Value(requestContextKey).(Context)
	return value, ok
}

type Actor struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Name        string   `json:"name"`
	TenantID    string   `json:"tenant_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func (a Actor) HasRole(role string) bool {
	for _, candidate := range a.Roles {
		if candidate == role {
			return true
		}
	}
	return false
}

func (a Actor) Can(permission string) bool {
	for _, candidate := range a.Permissions {
		if candidate == "*" || matchPermission(candidate, permission) {
			return true
		}
	}
	return false
}

func matchPermission(granted, requested string) bool {
	if granted == requested {
		return true
	}
	if granted == "*" {
		return true
	}
	if len(granted) > 2 && granted[len(granted)-2:] == ".*" {
		return len(requested) >= len(granted)-1 && requested[:len(granted)-1] == granted[:len(granted)-1]
	}
	if strings.HasPrefix(granted, "*.") {
		suffix := granted[1:]
		return len(requested) >= len(suffix) && strings.HasSuffix(requested, suffix)
	}
	return false
}
