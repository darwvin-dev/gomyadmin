package server

import (
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
	"github.com/darwvin-dev/gomyadmin/pkg/storage"
)

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
