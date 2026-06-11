package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
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

func TestParseAPIKeyPrefix(t *testing.T) {
	prefix, ok := parseAPIKeyPrefix("gma_ab12cd_deadbeef")
	if !ok {
		t.Fatal("expected valid api key")
	}
	if prefix != "ab12cd" {
		t.Fatalf("prefix = %q", prefix)
	}
}
