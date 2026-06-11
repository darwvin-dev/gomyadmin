package server

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

func (s *AdminServer) handleAuthProviders(w http.ResponseWriter, r *http.Request) {
	providers := make([]map[string]string, 0, len(s.cfg.OAuthProviders))
	for key, provider := range s.cfg.OAuthProviders {
		providers = append(providers, map[string]string{
			"name":      key,
			"label":     firstNonEmpty(provider.Name, key),
			"start_url": "/admin/api/auth/oauth/" + key + "/start",
		})
	}
	admin.WriteJSON(w, http.StatusOK, reqID(r), providers, nil)
}

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

func (s *AdminServer) handleOAuthStart(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := s.cfg.OAuthProviders[providerName]
	if !ok {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "OAuth provider not found", nil)
		return
	}
	state := auth.OAuthState{
		Provider:   providerName,
		SuccessURL: firstNonEmpty(r.URL.Query().Get("success"), s.cfg.OAuthSuccessURL),
		FailureURL: firstNonEmpty(r.URL.Query().Get("failure"), s.cfg.OAuthFailureURL),
	}
	nonce, err := s.oauth.Begin(w, state)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "OAUTH_FAILED", "Could not start OAuth flow", nil)
		return
	}
	redirectURI := s.cfg.PublicURL + "/admin/api/auth/oauth/" + providerName + "/callback"
	http.Redirect(w, r, provider.AuthorizationURL(redirectURI, nonce), http.StatusFound)
}

func (s *AdminServer) handleOAuthCallback(w http.ResponseWriter, r *http.Request) {
	providerName := chi.URLParam(r, "provider")
	provider, ok := s.cfg.OAuthProviders[providerName]
	if !ok {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "OAuth provider not found", nil)
		return
	}
	flowState, err := s.oauth.Verify(r, r.URL.Query().Get("state"))
	if err != nil {
		s.oauth.Clear(w)
		s.redirectOAuthFailure(w, r, s.cfg.OAuthFailureURL, "state")
		return
	}
	s.oauth.Clear(w)
	code := r.URL.Query().Get("code")
	if code == "" {
		s.redirectOAuthFailure(w, r, flowState.FailureURL, "code")
		return
	}
	if s.cfg.ResolveOAuthActor == nil {
		s.redirectOAuthFailure(w, r, flowState.FailureURL, "not_configured")
		return
	}
	redirectURI := s.cfg.PublicURL + "/admin/api/auth/oauth/" + providerName + "/callback"
	identity, err := provider.Exchange(r.Context(), code, redirectURI)
	if err != nil {
		s.redirectOAuthFailure(w, r, flowState.FailureURL, "exchange")
		return
	}
	actor, allowed, err := s.cfg.ResolveOAuthActor(r.Context(), providerName, identity)
	if err != nil || !allowed {
		s.redirectOAuthFailure(w, r, flowState.FailureURL, "denied")
		return
	}
	_, err = s.sessions.Start(r.Context(), w, actor)
	if err != nil {
		s.redirectOAuthFailure(w, r, flowState.FailureURL, "session")
		return
	}
	_, _ = auth.IssueCSRF(w, false)
	s.emitAudit(r, actor, "oauth_login", "auth", actor.ID, nil, map[string]any{
		"provider": providerName,
		"email":    identity.Email,
		"subject":  identity.Subject,
	})
	http.Redirect(w, r, flowState.SuccessURL, http.StatusFound)
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

func (s *AdminServer) redirectOAuthFailure(w http.ResponseWriter, r *http.Request, baseURL, code string) {
	target := firstNonEmpty(baseURL, s.cfg.OAuthFailureURL)
	parsed, err := url.Parse(target)
	if err != nil {
		admin.WriteError(w, http.StatusUnauthorized, reqID(r), "OAUTH_FAILED", "OAuth login failed", nil)
		return
	}
	values := parsed.Query()
	values.Set("oauth_error", code)
	parsed.RawQuery = values.Encode()
	http.Redirect(w, r, parsed.String(), http.StatusFound)
}
