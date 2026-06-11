package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
)

func TestOAuthStateManagerRoundTrip(t *testing.T) {
	manager := OAuthStateManager{SigningSecret: "test-secret"}
	rec := httptest.NewRecorder()
	state, err := manager.Begin(rec, OAuthState{Provider: "google", SuccessURL: "/admin/dashboard", FailureURL: "/admin/login"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
	for _, cookie := range rec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	got, err := manager.Verify(req, state)
	if err != nil {
		t.Fatal(err)
	}
	if got.Provider != "google" {
		t.Fatalf("provider = %q", got.Provider)
	}
}

func TestOAuthStateManagerRejectsWrongState(t *testing.T) {
	manager := OAuthStateManager{SigningSecret: "test-secret"}
	rec := httptest.NewRecorder()
	_, err := manager.Begin(rec, OAuthState{Provider: "google"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/callback?state=wrong", nil)
	for _, cookie := range rec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	if _, err := manager.Verify(req, "wrong"); err == nil {
		t.Fatal("expected verification error")
	}
}

func TestOAuthStateManagerRejectsTamperedCookie(t *testing.T) {
	manager := OAuthStateManager{SigningSecret: "test-secret"}
	rec := httptest.NewRecorder()
	state, err := manager.Begin(rec, OAuthState{Provider: "google"})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
	cookie := rec.Result().Cookies()[0]
	cookie.Value += "tampered"
	req.AddCookie(cookie)
	if _, err := manager.Verify(req, state); err == nil {
		t.Fatal("expected verification error")
	}
}

func TestOAuthStateManagerRejectsExpiredState(t *testing.T) {
	manager := OAuthStateManager{SigningSecret: "test-secret"}
	rec := httptest.NewRecorder()
	state, err := manager.Begin(rec, OAuthState{Provider: "google", RequestedAt: time.Now().Add(-11 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/callback?state="+state, nil)
	for _, cookie := range rec.Result().Cookies() {
		req.AddCookie(cookie)
	}
	if _, err := manager.Verify(req, state); err == nil {
		t.Fatal("expected expired state error")
	}
}

func TestOAuthStateManagerClearExpiresCookie(t *testing.T) {
	manager := OAuthStateManager{SigningSecret: "test-secret", Secure: true}
	rec := httptest.NewRecorder()
	manager.Clear(rec)
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies len = %d", len(cookies))
	}
	if cookies[0].Name != "gomyadmin_oauth" || cookies[0].MaxAge != -1 || !cookies[0].Secure {
		t.Fatalf("clear cookie = %#v", cookies[0])
	}
}

func TestOAuthProviderAuthorizationURLDefaultsScopes(t *testing.T) {
	provider := OAuthProvider{ClientID: "client", AuthURL: "https://example.test/auth"}
	raw := provider.AuthorizationURL("https://app.test/callback", "state-1")
	if !strings.HasPrefix(raw, "https://example.test/auth?") {
		t.Fatalf("authorization url = %q", raw)
	}
	req := httptest.NewRequest(http.MethodGet, raw, nil)
	values := req.URL.Query()
	if values.Get("client_id") != "client" || values.Get("redirect_uri") != "https://app.test/callback" || values.Get("state") != "state-1" {
		t.Fatalf("query = %v", values)
	}
	if values.Get("scope") != "openid email profile" {
		t.Fatalf("scope = %q", values.Get("scope"))
	}
}

func TestOAuthProviderExchangeFetchesUserInfo(t *testing.T) {
	providerBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("code") != "code-1" || r.Form.Get("redirect_uri") != "https://app.test/callback" {
				t.Fatalf("token form = %v", r.Form)
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"access_token": "access-1"})
		case "/userinfo":
			if r.Header.Get("Authorization") != "Bearer access-1" {
				t.Fatalf("authorization = %q", r.Header.Get("Authorization"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"sub": "subject-1", "email": "user@example.com", "name": "User"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer providerBackend.Close()

	provider := OAuthProvider{
		Name:         "test",
		ClientID:     "client",
		ClientSecret: "secret",
		TokenURL:     providerBackend.URL + "/token",
		UserInfoURL:  providerBackend.URL + "/userinfo",
	}
	identity, err := provider.Exchange(context.Background(), "code-1", "https://app.test/callback")
	if err != nil {
		t.Fatal(err)
	}
	if identity.Provider != "test" || identity.Subject != "subject-1" || identity.Email != "user@example.com" || identity.Name != "User" || identity.AccessToken != "access-1" {
		t.Fatalf("identity = %#v", identity)
	}
}

func TestOAuthProviderExchangeReturnsTokenErrorBody(t *testing.T) {
	providerBackend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "denied", http.StatusUnauthorized)
	}))
	defer providerBackend.Close()

	provider := OAuthProvider{TokenURL: providerBackend.URL}
	_, err := provider.Exchange(context.Background(), "code", "redirect")
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("err = %v", err)
	}
}

func TestParseAPIKeyPrefix(t *testing.T) {
	prefix, ok := parseAPIKeyPrefix("gma_ab12cd_deadbeef")
	if !ok {
		t.Fatal("expected valid api key")
	}
	if prefix != "ab12cd" {
		t.Fatalf("prefix = %q", prefix)
	}
}

func TestParseAPIKeyPrefixRejectsMalformedKeys(t *testing.T) {
	for _, raw := range []string{"", "gma_onlytwo", "bad_ab12cd_secret", "gma__secret", "gma_prefix_"} {
		if prefix, ok := parseAPIKeyPrefix(raw); ok {
			t.Fatalf("parseAPIKeyPrefix(%q) = %q, true", raw, prefix)
		}
	}
}

func TestAPIKeyHelpersCloneAndHash(t *testing.T) {
	values := []string{"a", "b"}
	cloned := cloneStrings(values)
	values[0] = "changed"
	if cloned[0] != "a" {
		t.Fatalf("clone was mutated: %#v", cloned)
	}
	if cloneStrings(nil) != nil {
		t.Fatal("nil clone should stay nil")
	}
	if hashAPIKey("secret") != hashAPIKey("secret") || hashAPIKey("secret") == hashAPIKey("other") {
		t.Fatal("hashAPIKey should be deterministic and input-sensitive")
	}
	if nullableString("") != nil {
		t.Fatal("empty nullable string should be nil")
	}
	if nullableString("tenant") != "tenant" {
		t.Fatalf("nullableString = %#v", nullableString("tenant"))
	}
}

func TestActorWithPermissionsClonesPermissions(t *testing.T) {
	permissions := []string{"users.view"}
	actor := actorWithPermissions(adminActor("actor-1"), permissions)
	permissions[0] = "changed"
	if len(actor.Permissions) != 1 || actor.Permissions[0] != "users.view" {
		t.Fatalf("permissions = %#v", actor.Permissions)
	}
}

func adminActor(id string) admin.Actor {
	return admin.Actor{ID: id, Email: id + "@example.com", Permissions: []string{"*"}}
}
