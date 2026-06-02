package server

import (
	"net/http"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

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
