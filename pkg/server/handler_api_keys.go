package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/auth"
)

func (s *AdminServer) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	if s.apiKeys == nil {
		admin.WriteError(w, http.StatusNotImplemented, reqID(r), "NOT_ENABLED", "API key management is not enabled", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	keys, err := s.apiKeys.List(r.Context(), actor)
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "QUERY_FAILED", "Could not load API keys", nil)
		return
	}
	admin.WriteJSON(w, http.StatusOK, reqID(r), keys, nil)
}

func (s *AdminServer) handleCreateAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.apiKeys == nil {
		admin.WriteError(w, http.StatusNotImplemented, reqID(r), "NOT_ENABLED", "API key management is not enabled", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	var input struct {
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		ExpiresIn string   `json:"expires_in"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_JSON", "Invalid JSON body", nil)
		return
	}
	expiresIn, err := parseOptionalDuration(input.ExpiresIn)
	if err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "INVALID_DURATION", "expires_in must be a valid Go duration like 24h", nil)
		return
	}
	key, secret, err := s.apiKeys.Create(r.Context(), auth.CreateAPIKeyInput{
		Name:      input.Name,
		Actor:     actor,
		Scopes:    input.Scopes,
		ExpiresIn: expiresIn,
	})
	if err != nil {
		admin.WriteError(w, http.StatusBadRequest, reqID(r), "CREATE_FAILED", err.Error(), nil)
		return
	}
	s.emitAudit(r, actor, "create_api_key", "api_keys", key.ID, nil, map[string]any{
		"name":   key.Name,
		"prefix": key.Prefix,
		"scopes": key.Scopes,
	})
	admin.WriteJSON(w, http.StatusCreated, reqID(r), map[string]any{
		"key":    key,
		"secret": secret,
	}, nil)
}

func (s *AdminServer) handleRevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	if s.apiKeys == nil {
		admin.WriteError(w, http.StatusNotImplemented, reqID(r), "NOT_ENABLED", "API key management is not enabled", nil)
		return
	}
	actor, _ := auth.ActorFromContext(r.Context())
	id := chi.URLParam(r, "id")
	err := s.apiKeys.Revoke(r.Context(), id, actor)
	if errors.Is(err, auth.ErrAPIKeyNotFound) {
		admin.WriteError(w, http.StatusNotFound, reqID(r), "NOT_FOUND", "API key not found", nil)
		return
	}
	if err != nil {
		admin.WriteError(w, http.StatusInternalServerError, reqID(r), "REVOKE_FAILED", "Could not revoke API key", nil)
		return
	}
	s.emitAudit(r, actor, "revoke_api_key", "api_keys", id, nil, nil)
	admin.WriteJSON(w, http.StatusOK, reqID(r), map[string]any{"ok": true}, nil)
}

func parseOptionalDuration(raw string) (time.Duration, error) {
	if raw == "" {
		return 0, nil
	}
	return time.ParseDuration(raw)
}
