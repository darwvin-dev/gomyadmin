//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/pkg/admin"
	"github.com/darwvin-dev/gomyadmin/pkg/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

type integUser struct{}

func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func newIntegServer(t *testing.T, pool *pgxpool.Pool) (*httptest.Server, *http.Cookie) {
	t.Helper()
	app := admin.New("IntegTest")
	app.Resource(integUser{}).
		TableName("integ_users").
		Field("ID").UUID().Primary().Readonly().
		Field("Name").String().Searchable()

	// Create test table
	_, err := pool.Exec(context.Background(), `
		CREATE TABLE IF NOT EXISTS integ_users (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			name TEXT
		)`)
	if err != nil {
		t.Fatalf("create table: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DROP TABLE IF EXISTS integ_users")
	})

	srv, err := server.New(context.Background(), server.Config{
		Pool: pool,
		App:  app,
		Authenticate: func(_ context.Context, email, password string) (admin.Actor, bool, error) {
			if email == "admin@test.com" && password == "secret" {
				return admin.Actor{
					ID: "a1", Email: email,
					Roles: []string{"super_admin"}, Permissions: []string{"*"},
				}, true, nil
			}
			return admin.Actor{}, false, nil
		},
	})
	if err != nil {
		t.Fatalf("server.New: %v", err)
	}
	t.Cleanup(srv.Close)

	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// Login and get session cookie
	body, _ := json.Marshal(map[string]string{"email": "admin@test.com", "password": "secret"})
	resp, err := http.Post(ts.URL+"/admin/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d", resp.StatusCode)
	}
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "gomyadmin_session" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatal("no session cookie")
	}
	return ts, sessionCookie
}

func authReq(t *testing.T, method, url string, body any, cookie *http.Cookie) *http.Response {
	t.Helper()
	var bodyBytes []byte
	if body != nil {
		bodyBytes, _ = json.Marshal(body)
	}
	req, _ := http.NewRequest(method, url, bytes.NewReader(bodyBytes))
	req.AddCookie(cookie)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, url, err)
	}
	return resp
}

func TestIntegServerLoginAndResources(t *testing.T) {
	pool := testPool(t)
	ts, cookie := newIntegServer(t, pool)

	resp := authReq(t, http.MethodGet, ts.URL+"/admin/api/resources", nil, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("resources status = %d", resp.StatusCode)
	}
}

func TestIntegServerCRUD(t *testing.T) {
	pool := testPool(t)
	ts, cookie := newIntegServer(t, pool)

	// Create
	resp := authReq(t, http.MethodPost, ts.URL+"/admin/api/integ_users",
		map[string]string{"name": "Alice"}, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", resp.StatusCode)
	}

	// List
	resp = authReq(t, http.MethodGet, ts.URL+"/admin/api/integ_users", nil, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", resp.StatusCode)
	}

	var listResp struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(listResp.Data) == 0 {
		t.Fatal("expected at least one user")
	}
	id := listResp.Data[0]["id"].(string)

	// Get
	resp = authReq(t, http.MethodGet, ts.URL+"/admin/api/integ_users/"+id, nil, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get status = %d", resp.StatusCode)
	}

	// Update
	resp = authReq(t, http.MethodPatch, ts.URL+"/admin/api/integ_users/"+id,
		map[string]string{"name": "Bob"}, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("update status = %d", resp.StatusCode)
	}

	// Delete
	resp = authReq(t, http.MethodDelete, ts.URL+"/admin/api/integ_users/"+id, nil, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}
}

func TestIntegServerAuditLog(t *testing.T) {
	pool := testPool(t)
	ts, cookie := newIntegServer(t, pool)

	// Create a record to generate audit event
	resp := authReq(t, http.MethodPost, ts.URL+"/admin/api/integ_users",
		map[string]string{"name": "AuditTest"}, cookie)
	resp.Body.Close()

	// Check audit log
	resp = authReq(t, http.MethodGet, ts.URL+"/admin/api/audit", nil, cookie)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("audit status = %d", resp.StatusCode)
	}
}

func TestIntegServerLogout(t *testing.T) {
	pool := testPool(t)
	ts, cookie := newIntegServer(t, pool)

	resp := authReq(t, http.MethodPost, ts.URL+"/admin/api/auth/logout", nil, cookie)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d", resp.StatusCode)
	}
}
