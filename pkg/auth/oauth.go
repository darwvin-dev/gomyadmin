package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrOAuthStateInvalid = errors.New("invalid oauth state")

type OAuthProvider struct {
	Name           string
	ClientID       string
	ClientSecret   string
	AuthURL        string
	TokenURL       string
	UserInfoURL    string
	Scopes         []string
	UserIDField    string
	UserEmailField string
	UserNameField  string
}

type OAuthIdentity struct {
	Provider    string         `json:"provider"`
	Subject     string         `json:"subject"`
	Email       string         `json:"email"`
	Name        string         `json:"name"`
	AccessToken string         `json:"-"`
	Raw         map[string]any `json:"raw"`
}

type OAuthState struct {
	Provider    string    `json:"provider"`
	State       string    `json:"state"`
	SuccessURL  string    `json:"success_url"`
	FailureURL  string    `json:"failure_url"`
	RequestedAt time.Time `json:"requested_at"`
}

type OAuthStateManager struct {
	SigningSecret string
	CookieName    string
	Secure        bool
}

func GoogleOAuthProvider(clientID, clientSecret string) OAuthProvider {
	return OAuthProvider{
		Name:           "google",
		ClientID:       clientID,
		ClientSecret:   clientSecret,
		AuthURL:        "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:       "https://oauth2.googleapis.com/token",
		UserInfoURL:    "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:         []string{"openid", "email", "profile"},
		UserIDField:    "sub",
		UserEmailField: "email",
		UserNameField:  "name",
	}
}

func (m OAuthStateManager) Begin(w http.ResponseWriter, state OAuthState) (string, error) {
	if m.CookieName == "" {
		m.CookieName = "gomyadmin_oauth"
	}
	if state.State == "" {
		nonce, err := secureToken(18)
		if err != nil {
			return "", err
		}
		state.State = nonce
	}
	if state.RequestedAt.IsZero() {
		state.RequestedAt = time.Now().UTC()
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	signed := m.sign(payload)
	http.SetCookie(w, &http.Cookie{
		Name:     m.CookieName,
		Value:    signed,
		Path:     "/",
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600,
	})
	return state.State, nil
}

func (m OAuthStateManager) Verify(r *http.Request, state string) (OAuthState, error) {
	if m.CookieName == "" {
		m.CookieName = "gomyadmin_oauth"
	}
	cookie, err := r.Cookie(m.CookieName)
	if err != nil || cookie.Value == "" {
		return OAuthState{}, ErrOAuthStateInvalid
	}
	payload, err := m.verify(cookie.Value)
	if err != nil {
		return OAuthState{}, err
	}
	var stored OAuthState
	if err := json.Unmarshal(payload, &stored); err != nil {
		return OAuthState{}, ErrOAuthStateInvalid
	}
	if subtleEqual(stored.State, state) != 1 {
		return OAuthState{}, ErrOAuthStateInvalid
	}
	if time.Since(stored.RequestedAt) > 10*time.Minute {
		return OAuthState{}, ErrOAuthStateInvalid
	}
	return stored, nil
}

func (m OAuthStateManager) Clear(w http.ResponseWriter) {
	if m.CookieName == "" {
		m.CookieName = "gomyadmin_oauth"
	}
	http.SetCookie(w, &http.Cookie{
		Name:     m.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (p OAuthProvider) AuthorizationURL(redirectURI, state string) string {
	values := url.Values{}
	values.Set("client_id", p.ClientID)
	values.Set("redirect_uri", redirectURI)
	values.Set("response_type", "code")
	values.Set("scope", strings.Join(defaultScopes(p.Scopes), " "))
	values.Set("state", state)
	return p.AuthURL + "?" + values.Encode()
}

func (p OAuthProvider) Exchange(ctx context.Context, code, redirectURI string) (OAuthIdentity, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("client_id", p.ClientID)
	form.Set("client_secret", p.ClientSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.TokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return OAuthIdentity{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return OAuthIdentity{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return OAuthIdentity{}, errors.New("oauth token exchange failed: " + string(body))
	}
	var token struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(res.Body).Decode(&token); err != nil {
		return OAuthIdentity{}, err
	}
	return p.fetchUserInfo(ctx, token.AccessToken)
}

func (p OAuthProvider) fetchUserInfo(ctx context.Context, accessToken string) (OAuthIdentity, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.UserInfoURL, nil)
	if err != nil {
		return OAuthIdentity{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return OAuthIdentity{}, err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return OAuthIdentity{}, errors.New("oauth userinfo failed: " + string(body))
	}
	var raw map[string]any
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return OAuthIdentity{}, err
	}
	return OAuthIdentity{
		Provider:    p.Name,
		Subject:     stringField(raw, firstNonEmptyString(p.UserIDField, "sub", "id")),
		Email:       stringField(raw, firstNonEmptyString(p.UserEmailField, "email")),
		Name:        stringField(raw, firstNonEmptyString(p.UserNameField, "name")),
		AccessToken: accessToken,
		Raw:         raw,
	}, nil
}

func (m OAuthStateManager) sign(payload []byte) string {
	mac := hmac.New(sha256.New, []byte(m.SigningSecret))
	mac.Write(payload)
	sum := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + hex.EncodeToString(sum)
}

func (m OAuthStateManager) verify(signed string) ([]byte, error) {
	parts := strings.Split(signed, ".")
	if len(parts) != 2 {
		return nil, ErrOAuthStateInvalid
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, ErrOAuthStateInvalid
	}
	expected := m.sign(payload)
	if subtleEqual(expected, signed) != 1 {
		return nil, ErrOAuthStateInvalid
	}
	return payload, nil
}

func subtleEqual(a, b string) int {
	if len(a) != len(b) {
		return 0
	}
	if hmac.Equal([]byte(a), []byte(b)) {
		return 1
	}
	return 0
}

func defaultScopes(scopes []string) []string {
	if len(scopes) == 0 {
		return []string{"openid", "email", "profile"}
	}
	return scopes
}

func stringField(raw map[string]any, key string) string {
	if raw == nil || key == "" {
		return ""
	}
	v, _ := raw[key]
	s, _ := v.(string)
	return s
}

func firstNonEmptyString(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
