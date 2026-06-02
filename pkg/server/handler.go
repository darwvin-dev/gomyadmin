package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

func (s *AdminServer) buildRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.corsMiddleware)
	r.Use(requestTimeout(10 * time.Second))

	loginLimiter := auth.NewRateLimiter(8, time.Minute)

	r.Post("/admin/api/auth/login", loginLimiter.Middleware(http.HandlerFunc(s.handleLogin)).ServeHTTP)
	r.Post("/admin/api/auth/logout", s.sessions.Middleware(http.HandlerFunc(s.handleLogout)).ServeHTTP)

	r.Group(func(r chi.Router) {
		r.Use(s.sessions.Middleware)
		r.Get("/admin/api/me", s.handleMe)
		r.Get("/admin/api/resources", s.handleResources)
		r.Get("/admin/api/audit", s.handleAudit)
		r.Post("/admin/api/files", s.handleUploadFile)
		r.Get("/admin/api/files", s.handleFiles)
		r.Get("/admin/api/files/{id}", s.handleFile)

		r.Route("/admin/api/{resource}", func(r chi.Router) {
			r.Get("/", s.handleList)
			r.Post("/", s.handleCreate)
			r.Get("/export", s.handleExport)
			r.Post("/bulk-delete", s.handleBulkDelete)
			r.Post("/bulk-actions/{action}", s.handleBulkAction)
			r.Get("/{id}", s.handleGet)
			r.Patch("/{id}", s.handleUpdate)
			r.Delete("/{id}", s.handleDelete)
			r.Post("/{id}/actions/{action}", s.handleAction)
		})
	})
	return r
}

func (s *AdminServer) handleResources(w http.ResponseWriter, r *http.Request) {
	admin.WriteJSON(w, http.StatusOK, reqID(r), s.store.Resources(), nil)
}

// Permission helpers

func (s *AdminServer) can(r *http.Request, permission string) bool {
	actor, ok := auth.ActorFromContext(r.Context())
	if !ok {
		return false
	}
	return actor.Can("*") || actor.Can(permission)
}

func (s *AdminServer) deny(w http.ResponseWriter, r *http.Request, resource string) {
	actor, _ := auth.ActorFromContext(r.Context())
	s.emitAudit(r, actor, "permission_denied", resource, "", nil, nil)
	admin.WriteError(w, http.StatusForbidden, reqID(r), "FORBIDDEN", "You do not have permission to perform this action", nil)
}

func (s *AdminServer) emitAudit(r *http.Request, actor admin.Actor, action, resource, resourceID string, oldValues, newValues map[string]any) {
	s.store.RecordAudit(r.Context(), auditEvent{
		ActorID:    actor.ID,
		ActorEmail: actor.Email,
		TenantID:   actor.TenantID,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		OldValues:  oldValues,
		NewValues:  newValues,
		IPAddress:  r.RemoteAddr,
		UserAgent:  r.UserAgent(),
		RequestID:  reqID(r),
		CreatedAt:  time.Now().UTC(),
	})
}

func (s *AdminServer) resourceByTable(table string) (*admin.Resource, bool) {
	for _, r := range s.app.Resources() {
		if r.Table == table {
			return r, true
		}
	}
	return nil, false
}

// Small utilities

func reqID(r *http.Request) string {
	if id := middleware.GetReqID(r.Context()); id != "" {
		return id
	}
	return r.Header.Get("X-Request-ID")
}

func parseInt(s string, fallback int) int {
	n, err := strconv.Atoi(s)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func firstRole(actor admin.Actor) string {
	if len(actor.Roles) == 0 {
		return ""
	}
	return actor.Roles[0]
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func allowedMIME(contentType string) bool {
	switch strings.Split(contentType, ";")[0] {
	case "image/png", "image/jpeg", "image/webp", "application/pdf", "text/plain", "text/csv":
		return true
	default:
		return false
	}
}

func strVal(v any) string {
	if v == nil {
		return ""
	}
	return fmt.Sprint(v)
}

func parseFilters(r *http.Request) map[string]string {
	filters := map[string]string{}
	for key, values := range r.URL.Query() {
		if !strings.HasPrefix(key, "filter[") || len(values) == 0 {
			continue
		}
		trimmed := strings.TrimPrefix(key, "filter[")
		parts := strings.Split(trimmed, "][")
		field := strings.TrimSuffix(parts[0], "]")
		if field == "" {
			continue
		}
		if len(parts) > 1 {
			field = field + "::" + strings.TrimSuffix(parts[1], "]")
		}
		filters[field] = values[0]
	}
	return filters
}

func cleanIDs(ids []string) []string {
	seen := map[string]bool{}
	cleaned := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		cleaned = append(cleaned, id)
	}
	return cleaned
}
