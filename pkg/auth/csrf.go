package auth

import (
	"net/http"

	"github.com/darwvin/gomyadmin/pkg/admin"
)

const csrfHeader = "X-CSRF-Token"

func CSRFMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			next.ServeHTTP(w, r)
			return
		}
		cookie, err := r.Cookie("gomyadmin_csrf")
		if err != nil || cookie.Value == "" || r.Header.Get(csrfHeader) != cookie.Value {
			admin.WriteError(w, http.StatusForbidden, r.Header.Get("X-Request-ID"), "CSRF_FAILED", "Invalid CSRF token", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func IssueCSRF(w http.ResponseWriter, secure bool) (string, error) {
	token, err := secureToken(24)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     "gomyadmin_csrf",
		Value:    token,
		Path:     "/",
		HttpOnly: false,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
	return token, nil
}
