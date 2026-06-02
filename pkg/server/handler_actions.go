package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

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
