//go:build integration

package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/darwvin-dev/gomyadmin/internal/generator"
)

func TestGeneratedAppDockerCompose(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker is not installed")
	}
	if os.Getenv("GOMYADMIN_RUN_DOCKER_COMPOSE_TEST") != "1" {
		t.Skip("set GOMYADMIN_RUN_DOCKER_COMPOSE_TEST=1 to run docker compose smoke tests")
	}
	dir := t.TempDir()
	appDir := filepath.Join(dir, "acme-admin")
	if err := generator.InitProject(generator.InitOptions{Name: appDir, Module: "github.com/acme/admin"}); err != nil {
		t.Fatal(err)
	}
	stageLocalModuleForDocker(t, appDir)
	backendPort := freePort(t)
	prepareComposeForSmokeTest(t, appDir, backendPort)
	env := fmt.Sprintf("DATABASE_URL=postgres://gomyadmin:gomyadmin@postgres:5432/gomyadmin?sslmode=disable\nGOMYADMIN_SESSION_SECRET=test-secret\nGOMYADMIN_PUBLIC_URL=http://localhost:%d\nGOMYADMIN_BACKEND_URL=http://localhost:%d\nNEXT_PUBLIC_ADMIN_API_URL=http://localhost:%d\n", backendPort, backendPort, backendPort)
	if err := os.WriteFile(filepath.Join(appDir, ".env"), []byte(env), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()
	up := exec.CommandContext(ctx, "docker", "compose", "up", "--build", "-d", "postgres", "backend")
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
		resp, err := client.Get(fmt.Sprintf("http://localhost:%d/admin/api/resources", backendPort))
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

func prepareComposeForSmokeTest(t *testing.T, appDir string, backendPort int) {
	t.Helper()
	path := filepath.Join(appDir, "docker-compose.yml")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(string(content), "\n")
	out := make([]string, 0, len(lines))
	var (
		inPostgres bool
		inBackend  bool
		inPorts    bool
	)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(line, "  ") && !strings.HasPrefix(line, "    ") && strings.HasSuffix(trimmed, ":") {
			inPostgres = trimmed == "postgres:"
			inBackend = trimmed == "backend:"
			inPorts = false
		}
		if strings.TrimSpace(line) == "ports:" {
			inPorts = true
			// Skip postgres host port publishing entirely.
			if inPostgres {
				continue
			}
		}
		if inPostgres && inPorts && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			continue
		}
		if inBackend && inPorts && strings.HasPrefix(strings.TrimSpace(line), "- ") {
			out = append(out, fmt.Sprintf("      - \"%d:8080\"", backendPort))
			inPorts = false
			continue
		}
		out = append(out, line)
	}
	if err := os.WriteFile(path, []byte(strings.Join(out, "\n")), 0o644); err != nil {
		t.Fatal(err)
	}
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func stageLocalModuleForDocker(t *testing.T, appDir string) {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goVersion := moduleGoVersion(t, filepath.Join(repoRoot, "go.mod"))
	moduleRoot := filepath.Join(appDir, "backend", "third_party", "gomyadmin")
	for _, file := range []string{"go.mod", "go.sum"} {
		copyFile(t, filepath.Join(repoRoot, file), filepath.Join(moduleRoot, file))
	}
	copyDir(t, filepath.Join(repoRoot, "pkg"), filepath.Join(moduleRoot, "pkg"))

	goModPath := filepath.Join(appDir, "backend", "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	updated := string(content)
	replaceLine := "replace github.com/darwvin-dev/gomyadmin => ./third_party/gomyadmin\n"
	if !strings.Contains(updated, replaceLine) {
		if !strings.HasSuffix(updated, "\n") {
			updated += "\n"
		}
		updated += "\n" + replaceLine
	}
	updated = replaceGoDirective(updated, goVersion)
	if err := os.WriteFile(goModPath, []byte(updated), 0o644); err != nil {
		t.Fatal(err)
	}

	dockerfilePath := filepath.Join(appDir, "backend", "Dockerfile")
	dockerfile, err := os.ReadFile(dockerfilePath)
	if err != nil {
		t.Fatal(err)
	}
	rewrittenDockerfile := strings.Replace(string(dockerfile), "FROM golang:1.23 AS build", "FROM golang:"+goVersionMajorMinor(goVersion)+" AS build", 1)
	rewrittenDockerfile = strings.Replace(rewrittenDockerfile, "FROM golang:1.25 AS build", "FROM golang:"+goVersionMajorMinor(goVersion)+" AS build", 1)
	rewrittenDockerfile = strings.Replace(rewrittenDockerfile, "FROM golang:1.23-bookworm AS runner", "FROM golang:"+goVersionMajorMinor(goVersion)+"-bookworm AS runner", 1)
	rewrittenDockerfile = strings.Replace(rewrittenDockerfile, "FROM golang:1.25-bookworm AS runner", "FROM golang:"+goVersionMajorMinor(goVersion)+"-bookworm AS runner", 1)
	if err := os.WriteFile(dockerfilePath, []byte(rewrittenDockerfile), 0o644); err != nil {
		t.Fatal(err)
	}
}

func moduleGoVersion(t *testing.T, goModPath string) string {
	t.Helper()
	content, err := os.ReadFile(goModPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	t.Fatalf("missing go directive in %s", goModPath)
	return ""
}

func replaceGoDirective(content, version string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "go ") {
			lines[i] = "go " + version
			return strings.Join(lines, "\n")
		}
	}
	return content
}

func goVersionMajorMinor(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return version
	}
	return parts[0] + "." + parts[1]
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(dst, 0o755); err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if entry.IsDir() {
			copyDir(t, srcPath, dstPath)
			continue
		}
		copyFile(t, srcPath, dstPath)
	}
}

func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
