//go:build integration

package integration

import (
	"context"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/internal/generator"
)

func TestGeneratedAppDockerCompose(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not installed")
	}
	dir := t.TempDir()
	appDir := filepath.Join(dir, "acme-admin")
	if err := generator.InitProject(generator.InitOptions{Name: appDir, Module: "github.com/acme/admin"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, ".env"), []byte("DATABASE_URL=postgres://gomyadmin:gomyadmin@postgres:5432/gomyadmin?sslmode=disable\nGOMYADMIN_SESSION_SECRET=test-secret\nGOMYADMIN_PUBLIC_URL=http://localhost:3000\nGOMYADMIN_BACKEND_URL=http://localhost:8080\nNEXT_PUBLIC_ADMIN_API_URL=http://localhost:8080\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	up := exec.CommandContext(ctx, "docker", "compose", "up", "--build", "-d")
	up.Dir = appDir
	if out, err := up.CombinedOutput(); err != nil {
		t.Fatalf("docker compose up failed: %v\n%s", err, out)
	}
	defer func() {
		down := exec.Command("docker", "compose", "down", "-v")
		down.Dir = appDir
		_ = down.Run()
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://localhost:8080/admin/api/resources")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusUnauthorized {
				return
			}
		}
		time.Sleep(time.Second)
	}
	t.Fatal("backend did not become reachable")
}
