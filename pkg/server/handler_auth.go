package server

import (
	"encoding/json"
	"net/http"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

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
