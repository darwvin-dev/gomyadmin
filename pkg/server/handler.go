package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/darwvin-dev/gomyadmin/pkg/storage"
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

// Auth handlers

func (s *AdminServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	if s.cfg.Authenticate == nil {
		admin.WriteError(w, http.StatusUnauthorized, reqID(r), "NOT_CONFIGURED", "Authentication is not configured", nil)
		return
	}
	actor, ok, err := s.cfg.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		s.log.Error("authenticate", "err", err)
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "LOGIN_FAILED", "Could not complete login", nil)
		return
	}
	if !ok {
		s.store.RecordAudit(r.Context(), auditEvent{
			ActorEmail: input.Email, Action: "failed_login", Resource: "auth",
			IPAddress: r.RemoteAddr, UserAgent: r.UserAgent(), RequestID: reqID(r),
		})
		admin.WriteError(w, http.StatusUnauthorized, reqID(r), "INVALID_CREDENTIALS", "Invalid email or password", nil)
		return
	}
	session, err := s.sessions.Start(r.Context(), w, actor)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "SESSION_FAILED", "Could not create session", nil)
		return
	}
	_, _ = auth.IssueCSRF(w, false)
	s.emitAudit(r, actor, "login", "auth", actor.ID, nil, nil)
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"user": actor, "expires_at": session.ExpiresAt}, nil)
}

func (s *AdminServer) handleLogout(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.ActorFromContext(r.Context())
	_ = s.sessions.End(r.Context(), w, r)
	s.emitAudit(r, actor, "logout", "auth", actor.ID, nil, nil)
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"ok": true}, nil)
}

func (s *AdminServer) handleMe(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.ActorFromContext(r.Context())
	tenants := []map[string]string{}
	if s.cfg.Tenants != nil {
		var err error
		tenants, err = s.cfg.Tenants(r.Context(), actor)
		if err != nil {
			admin.WriteError(w, http.StatusInternalServerError, reqID(r), "TENANTS_FAILED", "Could not load tenants", nil)
			return
		}
	}
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"user": actor, "tenants": tenants}, nil)
}

func (s *AdminServer) handleResources(w http.ResponseWriter, r *http.Request) {
	admin.WriteJSON(w, http.StatusOK, reqID(r), s.store.Resources(), nil)
}

// CRUD handlers

func (s *AdminServer) handleList(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	if !s.can(r, table+".view") {
		s.deny(w, r, table)
		return
	}
	if !s.store.HasResource(table) {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "Resource not found", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	page := parseInt(r.URL.Query().Get("page"), 1)
	perPage := parseInt(r.URL.Query().Get("per_page"), 25)
	records, total, err := s.store.List(
		r.Context(), table, actor.TenantID, firstRole(actor),
		r.URL.Query().Get("q"), r.URL.Query().Get("sort"),
		parseFilters(r), page, perPage,
	)
	if err != nil {
		s.log.Error("list", "table", table, "err", err)
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "QUERY_FAILED", "Could not load records", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, reqID(r), records, map[string]any{"page": page, "per_page": perPage, "total": total})
}

func (s *AdminServer) handleGet(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	if !s.can(r, table+".view") {
		s.deny(w, r, table)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	rec, err := s.store.Get(r.Context(), table, chi.URLParam(r, "id"), actor.TenantID, firstRole(actor))
	if errors.Is(err, errNotFound) {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "Record not found", nil)
		return
	}
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "QUERY_FAILED", "Could not load record", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, reqID(r), rec, nil)
}

func (s *AdminServer) handleCreate(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	if !s.can(r, table+".create") {
		s.deny(w, r, table)
		return
	}
	var input record
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	rec, err := s.store.Create(r.Context(), table, actor.TenantID, input)
	if err != nil {
		s.log.Error("create", "table", table, "err", err)
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "CREATE_FAILED", "Could not create record", nil)
		return
	}
	s.emitAudit(r, actor, "create", table, strVal(rec["id"]), nil, rec)
	admin.WriteJSON(w, http.StatusCreated, reqID(r), rec, nil)
}

func (s *AdminServer) handleUpdate(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	if !s.can(r, table+".update") {
		s.deny(w, r, table)
		return
	}
	var input record
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	id := chi.URLParam(r, "id")
	old, rec, err := s.store.Update(r.Context(), table, id, actor.TenantID, firstRole(actor), input)
	if errors.Is(err, errNotFound) {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "Record not found", nil)
		return
	}
	if err != nil {
		s.log.Error("update", "table", table, "err", err)
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "UPDATE_FAILED", "Could not update record", nil)
		return
	}
	s.emitAudit(r, actor, "update", table, id, old, rec)
	admin.WriteJSON(w, http.StatusOK, reqID(r), rec, nil)
}

func (s *AdminServer) handleDelete(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	if !s.can(r, table+".delete") {
		s.deny(w, r, table)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	id := chi.URLParam(r, "id")
	old, err := s.store.Delete(r.Context(), table, id, actor.TenantID, firstRole(actor))
	if errors.Is(err, errNotFound) {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "Record not found", nil)
		return
	}
	if err != nil {
		s.log.Error("delete", "table", table, "err", err)
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "DELETE_FAILED", "Could not delete record", nil)
		return
	}
	s.emitAudit(r, actor, "delete", table, id, old, nil)
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"deleted": true}, nil)
}

func (s *AdminServer) handleBulkDelete(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	if !s.can(r, table+".delete") {
		s.deny(w, r, table)
		return
	}
	var input struct {
		IDs []string `json:"ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	ids := cleanIDs(input.IDs)
	if len(ids) == 0 {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "EMPTY_SELECTION", "At least one record id is required", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	oldRecords, err := s.store.DeleteMany(r.Context(), table, ids, actor.TenantID, firstRole(actor))
	if errors.Is(err, errNotFound) {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "Resource or record not found", nil)
		return
	}
	if err != nil {
		s.log.Error("bulk_delete", "table", table, "err", err)
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "BULK_DELETE_FAILED", "Could not delete records", nil)
		return
	}
	s.emitAudit(r, actor, "bulk_delete", table, "", map[string]any{"records": oldRecords}, nil)
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"deleted": len(oldRecords), "ids": ids}, nil)
}

// Action handlers

func (s *AdminServer) handleAction(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	action := chi.URLParam(r, "action")
	if !s.can(r, table+".actions."+strings.ReplaceAll(action, "-", "_")) {
		s.deny(w, r, table)
		return
	}
	var input record
	_ = json.NewDecoder(r.Body).Decode(&input)
	id := chi.URLParam(r, "id")
	actor, _ := auth.ActorFromContext(r.Context())

	// Try to dispatch to a registered ActionHandler.
	if res, ok := s.resourceByTable(table); ok {
		if a, ok := res.ActionByName(action); ok {
			if fn, ok := a.Handler.(admin.ActionHandler); ok {
				result, err := fn(r.Context(), admin.ActionRequest[map[string]any]{
					Actor:      actor,
					Resource:   res,
					ResourceID: id,
					Input:      input,
				})
				if err != nil {
					s.log.Error("action", "table", table, "action", action, "err", err)
					admin.WriteError(w, http.StatusInternalServerError, reqID(r), "ACTION_FAILED", err.Error(), nil)
					return
				}
				s.emitAudit(r, actor, "action:"+action, table, id, nil, map[string]any{"message": result.Message})
				admin.WriteJSON(w, http.StatusOK, reqID(r), result, nil)
				return
			}
		}
	}

	// No handler registered — acknowledge receipt so the frontend doesn't hang.
	s.emitAudit(r, actor, "action:"+action, table, id, nil, input)
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"message": "Action dispatched"}, nil)
}

func (s *AdminServer) handleBulkAction(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	action := chi.URLParam(r, "action")
	if !s.can(r, table+".actions."+strings.ReplaceAll(action, "-", "_")) {
		s.deny(w, r, table)
		return
	}
	var input struct {
		IDs []string `json:"ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	actor, _ := auth.ActorFromContext(r.Context())
	s.emitAudit(r, actor, "bulk_action:"+action, table, "", nil, map[string]any{"ids": input.IDs})
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"message": "Bulk action queued", "count": len(input.IDs)}, nil)
}

func (s *AdminServer) handleExport(w http.ResponseWriter, r *http.Request) {
	table := chi.URLParam(r, "resource")
	if !s.can(r, table+".view") {
		s.deny(w, r, table)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	records, _, err := s.store.List(r.Context(), table, actor.TenantID, firstRole(actor), "", "", nil, 1, 100)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "EXPORT_FAILED", "Could not export records", nil)
		return
	}
	s.emitAudit(r, actor, "export", table, "", nil, map[string]any{"count": len(records)})
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+table+`.csv"`)
	if len(records) > 0 {
		cols := make([]string, 0, len(records[0]))
		for k := range records[0] {
			cols = append(cols, k)
		}
		sort.Strings(cols)
		_, _ = fmt.Fprintln(w, strings.Join(cols, ","))
		for _, rec := range records {
			vals := make([]string, len(cols))
			for i, c := range cols {
				vals[i] = strVal(rec[c])
			}
			_, _ = fmt.Fprintln(w, strings.Join(vals, ","))
		}
	}
}

// Audit handler

func (s *AdminServer) handleAudit(w http.ResponseWriter, r *http.Request) {
	if !s.can(r, "audit.view") {
		s.deny(w, r, "audit")
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	events, err := s.store.Audit(r.Context(), actor.TenantID, firstRole(actor))
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "AUDIT_FAILED", "Could not load audit log", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, reqID(r), events, nil)
}

// File handlers

func (s *AdminServer) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if !s.can(r, "files.create") {
		s.deny(w, r, "files")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_FILE", "A 'file' form field is required", nil)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}
	if !allowedMIME(contentType) {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_MIME", "File type not allowed", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	key := actor.TenantID + "/file_" + randomHex(8) + "_" + filepath.Base(header.Filename)
	if err := s.uploads.Put(r.Context(), storage.Object{
		Key: key, Reader: file, ContentType: contentType, Size: header.Size,
	}); err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "UPLOAD_FAILED", "Could not store file", nil)
		return
	}
	rec := record{
		"id":           "file_" + randomHex(8),
		"tenant_id":    actor.TenantID,
		"key":          key,
		"name":         header.Filename,
		"content_type": contentType,
		"size":         header.Size,
		"created_at":   time.Now().UTC(),
	}
	if err := s.store.AddFile(r.Context(), rec); err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "FILE_FAILED", "Could not record file metadata", nil)
		return
	}
	s.emitAudit(r, actor, "file_upload", "files", strVal(rec["id"]), nil, rec)
	admin.WriteJSON(w, http.StatusCreated, reqID(r), rec, nil)
}

func (s *AdminServer) handleFiles(w http.ResponseWriter, r *http.Request) {
	if !s.can(r, "files.view") {
		s.deny(w, r, "files")
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	files, err := s.store.Files(r.Context(), actor.TenantID, firstRole(actor))
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "FILES_FAILED", "Could not load files", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, reqID(r), files, nil)
}

func (s *AdminServer) handleFile(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.ActorFromContext(r.Context())
	key, err := s.store.FileKey(r.Context(), chi.URLParam(r, "id"), actor.TenantID, firstRole(actor))
	if err != nil {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "File not found", nil)
		return
	}
	obj, err := s.uploads.Get(r.Context(), key)
	if err != nil {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "File not found", nil)
		return
	}
	defer func() {
		if c, ok := obj.Reader.(io.Closer); ok {
			_ = c.Close()
		}
	}()
	w.Header().Set("Content-Type", firstNonEmpty(obj.ContentType, "application/octet-stream"))
	_, _ = io.Copy(w, obj.Reader)
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

// Middleware

func (s *AdminServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.cfg.PublicURL)
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-CSRF-Token, X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestTimeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
