package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/darwvin-dev/gomyadmin/internal/generator"
)

func TestGeneratedAppContainsRunnableStack(t *testing.T) {
	dir := t.TempDir()
	appDir := filepath.Join(dir, "acme-admin")
	if err := generator.InitProject(generator.InitOptions{Name: appDir, Module: "github.com/acme/admin"}); err != nil {
		t.Fatal(err)
	}

	for _, rel := range []string{
		"docker-compose.yml",
		"backend/cmd/server/main.go",
		"backend/internal/db/migrations/001_init.sql",
		"backend/internal/db/seeds/001_demo.sql",
		"frontend/app/admin/dashboard/page.tsx",
		"frontend/app/admin/resources/page.tsx",
		"frontend/lib/api.ts",
	} {
		if _, err := os.Stat(filepath.Join(appDir, rel)); err != nil {
			t.Fatalf("missing generated file %s: %v", rel, err)
		}
	}
}
