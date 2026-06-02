package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

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
