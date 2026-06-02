//go:build integration

package integration

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// cliBin is the path to the compiled gomyadmin binary, built once for all CLI tests.
var cliBin string

// buildCLIOnce compiles the gomyadmin binary into os.TempDir and caches the path.
// It is called from TestMain so the build happens once per test run.
func buildCLIOnce() error {
	binName := "gomyadmin"
	if runtime.GOOS == "windows" {
		binName = "gomyadmin.exe"
	}
	bin := filepath.Join(os.TempDir(), "gomyadmin-integ-"+binName)
	// Find the module root (two levels up from tests/integration)
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		return err
	}
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/gomyadmin")
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		_ = out
		return err
	}
	cliBin = bin
	return nil
}

// buildCLI returns the cached binary path. Tests call this instead of building themselves.
func buildCLI(t *testing.T) string {
	t.Helper()
	if cliBin == "" {
		t.Fatal("CLI binary not built — TestMain should have built it")
	}
	return cliBin
}

func TestMain(m *testing.M) {
	// Build the CLI binary once before all tests.
	if err := buildCLIOnce(); err != nil {
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func TestCLIIntegVersionCommand(t *testing.T) {
	bin := buildCLI(t)
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		t.Fatalf("version: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected version output")
	}
}

func TestCLIIntegUnknownCommandExitCode(t *testing.T) {
	bin := buildCLI(t)
	cmd := exec.Command(bin, "does-not-exist")
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit for unknown command")
	}
}

func TestCLIIntegDoctorRunsAndProducesOutput(t *testing.T) {
	bin := buildCLI(t)
	cmd := exec.Command(bin, "doctor")
	out, _ := cmd.CombinedOutput()
	if len(out) == 0 {
		t.Fatal("expected doctor output")
	}
}

func TestCLIIntegInitCreatesProject(t *testing.T) {
	bin := buildCLI(t)
	dir := t.TempDir()
	projectName := filepath.Join(dir, "test-admin")

	cmd := exec.Command(bin, "init", projectName)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("init failed: %v\n%s", err, out)
	}

	for _, rel := range []string{
		"docker-compose.yml",
		"backend/cmd/server/main.go",
	} {
		if _, err := os.Stat(filepath.Join(projectName, rel)); err != nil {
			t.Fatalf("missing %s: %v", rel, err)
		}
	}
}

func TestCLIIntegIntrospectRequiresDatabaseURL(t *testing.T) {
	bin := buildCLI(t)
	cmd := exec.Command(bin, "introspect")
	// Explicitly unset DATABASE_URL
	env := []string{}
	for _, e := range os.Environ() {
		if len(e) >= 12 && e[:12] == "DATABASE_URL" {
			continue
		}
		env = append(env, e)
	}
	cmd.Env = env
	if err := cmd.Run(); err == nil {
		t.Fatal("expected non-zero exit when DATABASE_URL is missing")
	}
}

func TestCLIIntegIntrospectWithDatabaseURL(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}
	bin := buildCLI(t)
	cmd := exec.Command(bin, "introspect", "--database-url", dbURL)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("introspect: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected JSON output")
	}
	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("output is not valid JSON: %v — got: %s", err, out)
	}
}
