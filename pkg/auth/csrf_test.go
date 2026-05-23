package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCSRFMiddlewareAllowsGET(t *testing.T) {
	called := false
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("GET should pass through CSRF middleware")
	}
}

func TestCSRFMiddlewareRejectsMissingToken(t *testing.T) {
	called := false
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodPost, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("POST without CSRF token should be rejected")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestCSRFMiddlewareAcceptsMatchingToken(t *testing.T) {
	called := false
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	const token = "test-csrf-token-abc123"
	req := httptest.NewRequest(http.MethodPost, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: "gomyadmin_csrf", Value: token})
	req.Header.Set(csrfHeader, token)

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("handler should be called with valid CSRF token")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestCSRFMiddlewareRejectsMismatchedToken(t *testing.T) {
	called := false
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))

	req := httptest.NewRequest(http.MethodDelete, "/admin/resource/1", nil)
	req.AddCookie(&http.Cookie{Name: "gomyadmin_csrf", Value: "cookie-token"})
	req.Header.Set(csrfHeader, "different-token")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if called {
		t.Error("mismatched token should be rejected")
	}
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestIssueCSRFSetsNonHttpOnlyCookie(t *testing.T) {
	w := httptest.NewRecorder()
	token, err := IssueCSRF(w, false)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	cookies := w.Result().Cookies()
	var csrfCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "gomyadmin_csrf" {
			csrfCookie = c
			break
		}
	}
	if csrfCookie == nil {
		t.Fatal("csrf cookie not set")
	}
	if csrfCookie.HttpOnly {
		t.Error("csrf cookie should not be HttpOnly so JS can read it")
	}
	if csrfCookie.Value != token {
		t.Fatalf("cookie value = %q, want %q", csrfCookie.Value, token)
	}
}

func TestCSRFMiddlewareAllowsHEAD(t *testing.T) {
	called := false
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodHead, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !called {
		t.Error("HEAD should pass through")
	}
}

func TestCSRFMiddlewareAllowsOPTIONS(t *testing.T) {
	called := false
	handler := CSRFMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	req := httptest.NewRequest(http.MethodOptions, "/admin", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if !called {
		t.Error("OPTIONS should pass through")
	}
}
