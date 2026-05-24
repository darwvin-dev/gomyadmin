package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/darwvin-dev/gomyadmin/pkg/storage"
)

type Server struct {
	log      *slog.Logger
	store    *Store
	sessions *auth.SessionManager
	uploads  storage.Storage
}

func NewServer(ctx context.Context, log *slog.Logger) (*Server, error) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}
	store, err := NewPostgresStore(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	return &Server{
		log:      log,
		store:    store,
		sessions: auth.NewSessionManager(auth.NewMemorySessionStore()),
		uploads:  storage.NewLocal("tmp/uploads", "http://localhost:8080/admin/api/files"),
	}, nil
}

func (s *Server) Close() {
	s.store.Close()
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(cors)
	r.Use(requestTimeout(10 * time.Second))

	loginLimiter := auth.NewRateLimiter(8, time.Minute)
	r.Post("/admin/api/auth/login", loginLimiter.Middleware(http.HandlerFunc(s.login)).ServeHTTP)
	r.Post("/admin/api/auth/logout", s.sessions.Middleware(http.HandlerFunc(s.logout)).ServeHTTP)
	r.Post("/admin/api/auth/forgot-password", s.notImplemented("PASSWORD_RESET_REQUESTED"))
	r.Post("/admin/api/auth/reset-password", s.notImplemented("PASSWORD_RESET_COMPLETED"))

	r.Group(func(r chi.Router) {
		r.Use(s.sessions.Middleware)
		r.Get("/admin/api/me", s.me)
		r.Get("/admin/api/resources", s.resources)
		r.Get("/admin/api/audit", s.audit)
		r.Post("/admin/api/files", s.uploadFile)
		r.Get("/admin/api/files", s.files)
		r.Get("/admin/api/files/{id}", s.file)

		r.Route("/admin/api/{resource}", func(r chi.Router) {
			r.Get("/", s.list)
			r.Post("/", s.create)
			r.Get("/export", s.export)
			r.Post("/import", s.importRecords)
			r.Post("/bulk-actions/{action}", s.bulkAction)
			r.Get("/{id}", s.get)
			r.Patch("/{id}", s.update)
			r.Delete("/{id}", s.delete)
			r.Post("/{id}/actions/{action}", s.action)
		})
	})
	return r
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, requestID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	actor, ok, err := s.store.Authenticate(r.Context(), input.Email, input.Password)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "LOGIN_FAILED", "Could not complete login", nil)
		return
	}
	if !ok {
		s.auditEvent(r, Actor{Email: input.Email}, "failed_login", "auth", "", nil, nil)
		admin.WriteError(w, http.StatusUnauthorized, requestID(r), "INVALID_CREDENTIALS", "Invalid email or password", nil)
		return
	}
	session, err := s.sessions.Start(r.Context(), w, actor)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "SESSION_FAILED", "Could not create session", nil)
		return
	}
	_, _ = auth.IssueCSRF(w, false)
	s.auditEvent(r, Actor(actor), "login", "auth", actor.ID, nil, nil)
	admin.WriteJSON(w, http.StatusOK, requestID(r), map[string]any{"user": actor, "expires_at": session.ExpiresAt}, nil)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.ActorFromContext(r.Context())
	_ = s.sessions.End(r.Context(), w, r)
	s.auditEvent(r, Actor(actor), "logout", "auth", actor.ID, nil, nil)
	admin.WriteJSON(w, http.StatusOK, requestID(r), map[string]any{"ok": true}, nil)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.ActorFromContext(r.Context())
	tenants, err := s.store.Tenants(r.Context())
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "TENANTS_FAILED", "Could not load tenants", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, requestID(r), map[string]any{
		"user":    actor,
		"tenants": tenants,
	}, nil)
}

func (s *Server) resources(w http.ResponseWriter, r *http.Request) {
	admin.WriteJSON(w, http.StatusOK, requestID(r), s.store.Resources(), nil)
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if !s.can(r, resource+".view") {
		s.forbidden(w, r, resource)
		return
	}
	if _, ok := s.store.Resource(resource); !ok {
		admin.WriteError(w, http.StatusNotFound, requestID(r), "NOT_FOUND", "Resource not found", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	page := parseInt(r.URL.Query().Get("page"), 1)
	perPage := parseInt(r.URL.Query().Get("per_page"), 25)
	filters := map[string]string{}
	for key, values := range r.URL.Query() {
		if strings.HasPrefix(key, "filter[") && len(values) > 0 {
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
	}
	records, total, err := s.store.List(r.Context(), resource, actor.TenantID, firstRole(actor), r.URL.Query().Get("q"), r.URL.Query().Get("sort"), filters, page, perPage)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "QUERY_FAILED", "Could not load records", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, requestID(r), records, map[string]any{"page": page, "per_page": perPage, "total": total})
}

func (s *Server) get(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if !s.can(r, resource+".view") {
		s.forbidden(w, r, resource)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	record, err := s.store.Get(r.Context(), resource, chi.URLParam(r, "id"), actor.TenantID, firstRole(actor))
	if errors.Is(err, errNotFound) {
		admin.WriteError(w, http.StatusNotFound, requestID(r), "NOT_FOUND", "Record not found", nil)
		return
	}
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "QUERY_FAILED", "Could not load record", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, requestID(r), record, nil)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if !s.can(r, resource+".create") {
		s.forbidden(w, r, resource)
		return
	}
	var input Record
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, requestID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	record, err := s.store.Create(r.Context(), resource, actor.TenantID, input)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "CREATE_FAILED", "Could not create record", nil)
		return
	}
	s.auditEvent(r, Actor(actor), "create", resource, stringValue(record["id"]), nil, record)
	admin.WriteJSON(w, http.StatusCreated, requestID(r), record, nil)
}

func (s *Server) update(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if !s.can(r, resource+".update") {
		s.forbidden(w, r, resource)
		return
	}
	var input Record
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, requestID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	oldRecord, record, err := s.store.Update(r.Context(), resource, chi.URLParam(r, "id"), actor.TenantID, firstRole(actor), input)
	if errors.Is(err, errNotFound) {
		admin.WriteError(w, http.StatusNotFound, requestID(r), "NOT_FOUND", "Record not found", nil)
		return
	}
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "UPDATE_FAILED", "Could not update record", nil)
		return
	}
	s.auditEvent(r, Actor(actor), "update", resource, chi.URLParam(r, "id"), oldRecord, record)
	admin.WriteJSON(w, http.StatusOK, requestID(r), record, nil)
}

func (s *Server) delete(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if !s.can(r, resource+".delete") {
		s.forbidden(w, r, resource)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	oldRecord, err := s.store.Delete(r.Context(), resource, chi.URLParam(r, "id"), actor.TenantID, firstRole(actor))
	if errors.Is(err, errNotFound) {
		admin.WriteError(w, http.StatusNotFound, requestID(r), "NOT_FOUND", "Record not found", nil)
		return
	}
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "DELETE_FAILED", "Could not delete record", nil)
		return
	}
	s.auditEvent(r, Actor(actor), "delete", resource, chi.URLParam(r, "id"), oldRecord, nil)
	admin.WriteJSON(w, http.StatusOK, requestID(r), map[string]any{"deleted": true}, nil)
}

func (s *Server) action(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	action := chi.URLParam(r, "action")
	if !s.can(r, resource+".actions."+strings.ReplaceAll(action, "-", "_")) {
		s.forbidden(w, r, resource)
		return
	}
	var input Record
	_ = json.NewDecoder(r.Body).Decode(&input)
	id := chi.URLParam(r, "id")
	switch action {
	case "block-user":
		actor, _ := auth.ActorFromContext(r.Context())
		_, _, _ = s.store.Update(r.Context(), resource, id, actor.TenantID, firstRole(actor), Record{"status": "blocked"})
	case "reset-password":
	case "mark-as-paid":
		actor, _ := auth.ActorFromContext(r.Context())
		_, _, _ = s.store.Update(r.Context(), resource, id, actor.TenantID, firstRole(actor), Record{"status": "paid"})
	case "refund-invoice":
		actor, _ := auth.ActorFromContext(r.Context())
		_, _, _ = s.store.Update(r.Context(), resource, id, actor.TenantID, firstRole(actor), Record{"status": "refunded"})
	case "close-ticket":
		actor, _ := auth.ActorFromContext(r.Context())
		_, _, _ = s.store.Update(r.Context(), resource, id, actor.TenantID, firstRole(actor), Record{"status": "solved"})
	default:
		admin.WriteError(w, http.StatusNotFound, requestID(r), "ACTION_NOT_FOUND", "Action not found", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	s.auditEvent(r, Actor(actor), "custom_action:"+action, resource, id, nil, input)
	admin.WriteJSON(w, http.StatusOK, requestID(r), map[string]any{"message": "Action completed"}, nil)
}

func (s *Server) bulkAction(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	action := chi.URLParam(r, "action")
	if !s.can(r, resource+".actions."+strings.ReplaceAll(action, "-", "_")) {
		s.forbidden(w, r, resource)
		return
	}
	var input struct {
		IDs []string `json:"ids"`
	}
	_ = json.NewDecoder(r.Body).Decode(&input)
	actor, _ := auth.ActorFromContext(r.Context())
	s.auditEvent(r, Actor(actor), "bulk_action:"+action, resource, "", nil, map[string]any{"ids": input.IDs})
	admin.WriteJSON(w, http.StatusOK, requestID(r), map[string]any{"message": "Bulk action queued", "count": len(input.IDs)}, nil)
}

func (s *Server) export(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if !s.can(r, resource+".view") {
		s.forbidden(w, r, resource)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	records, _, err := s.store.List(r.Context(), resource, actor.TenantID, firstRole(actor), "", "", nil, 1, 100)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "EXPORT_FAILED", "Could not export records", nil)
		return
	}
	s.auditEvent(r, Actor(actor), "export", resource, "", nil, map[string]any{"count": len(records)})
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="`+resource+`.csv"`)
	_, _ = w.Write([]byte("id,name,status\n"))
	for _, record := range records {
		_, _ = fmt.Fprintf(w, "%s,%s,%s\n", stringValue(record["id"]), stringValue(record["name"]), stringValue(record["status"]))
	}
}

func (s *Server) importRecords(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if !s.can(r, resource+".create") {
		s.forbidden(w, r, resource)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	s.auditEvent(r, Actor(actor), "import", resource, "", nil, map[string]any{"status": "accepted"})
	admin.WriteJSON(w, http.StatusAccepted, requestID(r), map[string]any{"message": "Import accepted"}, nil)
}

func (s *Server) audit(w http.ResponseWriter, r *http.Request) {
	if !s.can(r, "audit.view") {
		s.forbidden(w, r, "audit")
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	events, err := s.store.Audit(r.Context(), actor.TenantID, firstRole(actor))
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "AUDIT_FAILED", "Could not load audit log", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, requestID(r), events, nil)
}

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	if !s.can(r, "files.create") {
		s.forbidden(w, r, "files")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		admin.WriteError(w, http.StatusBadRequest, requestID(r), "INVALID_FILE", "A file field is required", nil)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}
	if !allowedMIME(contentType) {
		admin.WriteError(w, http.StatusBadRequest, requestID(r), "INVALID_MIME", "File type is not allowed", nil)
		return
	}

	actor, _ := auth.ActorFromContext(r.Context())
	key := actor.TenantID + "/" + randomID("file") + "-" + filepath.Base(header.Filename)
	if err := s.uploads.Put(r.Context(), storage.Object{Key: key, Reader: file, ContentType: contentType, Size: header.Size}); err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "UPLOAD_FAILED", "Could not store file", nil)
		return
	}
	record := Record{"id": randomID("file"), "tenant_id": actor.TenantID, "key": key, "name": header.Filename, "content_type": contentType, "size": header.Size, "created_at": time.Now().UTC()}
	if err := s.store.AddFile(r.Context(), record); err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "FILE_METADATA_FAILED", "Could not record file metadata", nil)
		return
	}
	s.auditEvent(r, Actor(actor), "file_upload", "files", stringValue(record["id"]), nil, record)
	admin.WriteJSON(w, http.StatusCreated, requestID(r), record, nil)
}

func (s *Server) files(w http.ResponseWriter, r *http.Request) {
	if !s.can(r, "files.view") {
		s.forbidden(w, r, "files")
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	files, err := s.store.Files(r.Context(), actor.TenantID, firstRole(actor))
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, requestID(r), "FILES_FAILED", "Could not load files", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, requestID(r), files, nil)
}

func (s *Server) file(w http.ResponseWriter, r *http.Request) {
	actor, _ := auth.ActorFromContext(r.Context())
	key, err := s.store.FileKey(r.Context(), chi.URLParam(r, "id"), actor.TenantID, firstRole(actor))
	if err != nil {
		admin.WriteError(w, http.StatusNotFound, requestID(r), "FILE_NOT_FOUND", "File not found", nil)
		return
	}
	object, err := s.uploads.Get(r.Context(), key)
	if err != nil {
		admin.WriteError(w, http.StatusNotFound, requestID(r), "FILE_NOT_FOUND", "File not found", nil)
		return
	}
	defer closeReader(object.Reader)
	w.Header().Set("Content-Type", first(object.ContentType, "application/octet-stream"))
	_, _ = io.Copy(w, object.Reader)
}

func (s *Server) forbidden(w http.ResponseWriter, r *http.Request, resource string) {
	actor, _ := auth.ActorFromContext(r.Context())
	s.auditEvent(r, Actor(actor), "permission_denied", resource, "", nil, nil)
	admin.WriteError(w, http.StatusForbidden, requestID(r), "FORBIDDEN", "You do not have permission to perform this action", nil)
}

func (s *Server) can(r *http.Request, permission string) bool {
	actor, _ := auth.ActorFromContext(r.Context())
	if actor.Can("*") {
		return true
	}
	for _, p := range actor.Permissions {
		if p == permission || p == "*" {
			return true
		}
	}
	return false
}

func (s *Server) auditEvent(r *http.Request, actor Actor, action, resource, resourceID string, oldValues, newValues map[string]any) {
	_ = s.store.RecordAudit(r.Context(), AuditEvent{
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
		RequestID:  requestID(r),
		CreatedAt:  time.Now().UTC(),
	})
}

func (s *Server) notImplemented(code string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		admin.WriteJSON(w, http.StatusAccepted, requestID(r), map[string]any{"message": code}, nil)
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", first(os.Getenv("GOMYADMIN_PUBLIC_URL"), "http://localhost:3000"))
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

func requestTimeout(timeout time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func requestID(r *http.Request) string {
	if id := middleware.GetReqID(r.Context()); id != "" {
		return id
	}
	return r.Header.Get("X-Request-ID")
}

func parseInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func first(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstRole(actor admin.Actor) string {
	if len(actor.Roles) == 0 {
		return ""
	}
	return actor.Roles[0]
}

func allowedMIME(contentType string) bool {
	switch strings.Split(contentType, ";")[0] {
	case "image/png", "image/jpeg", "image/webp", "application/pdf", "text/plain", "text/csv":
		return true
	default:
		return false
	}
}

func randomID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(b[:])
}

func closeReader(reader io.Reader) {
	if closer, ok := reader.(io.Closer); ok {
		_ = closer.Close()
	}
}
