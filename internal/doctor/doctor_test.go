package doctor

import (
	"context"
	"os"
	"testing"
)

func TestRunReturnsChecks(t *testing.T) {
	checks := Run(context.Background(), Options{})
	if len(checks) == 0 {
		t.Fatal("expected at least one check")
	}
	for _, c := range checks {
		if c.Name == "" {
			t.Error("check has empty name")
		}
	}
}

func TestEnvSetReturnsOK(t *testing.T) {
	c := env("DATABASE_URL", "postgres://localhost/db")
	if !c.OK {
		t.Fatalf("expected OK for set env, got: %s", c.Message)
	}
	if c.Message != "set" {
		t.Fatalf("message = %q", c.Message)
	}
}

func TestEnvEmptyReturnsFail(t *testing.T) {
	c := env("DATABASE_URL", "")
	if c.OK {
		t.Fatal("expected not OK for empty env")
	}
	if c.Message != "not set" {
		t.Fatalf("message = %q", c.Message)
	}
}

func TestCommandGoVersion(t *testing.T) {
	c := command("Go", "go", "version")
	if !c.OK {
		t.Fatalf("go version check failed: %s", c.Message)
	}
	if c.Name != "Go" {
		t.Fatalf("name = %q", c.Name)
	}
}

func TestCommandMissingBinary(t *testing.T) {
	c := command("NoSuchTool", "this-binary-does-not-exist-xyz", "--version")
	if c.OK {
		t.Fatal("expected not OK for missing binary")
	}
	if c.Name != "NoSuchTool" {
		t.Fatalf("name = %q", c.Name)
	}
}

func TestOptionalCommandMissingBinary(t *testing.T) {
	// Optional commands should always return OK=true even when missing.
	c := optionalCommand("OptionalTool", "this-binary-does-not-exist-xyz", "--version")
	if !c.OK {
		t.Fatalf("expected OK for optional missing binary, got: %s", c.Message)
	}
}

func TestFileWritableCurrentDir(t *testing.T) {
	c := fileWritable(".")
	if !c.OK {
		t.Fatalf("expected current dir to be writable: %s", c.Message)
	}
}

func TestFileWritableNonExistentPath(t *testing.T) {
	c := fileWritable("/this/path/does/not/exist/hopefully")
	if c.OK {
		t.Fatal("expected not OK for non-existent path")
	}
}

func TestRunSkipsPostgresCheckWhenNoDatabaseURL(t *testing.T) {
	checks := Run(context.Background(), Options{DatabaseURL: ""})
	for _, c := range checks {
		if c.Name == "PostgreSQL" {
			t.Fatal("should not run postgres check when DatabaseURL is empty")
		}
	}
}

func TestRunIncludesEnvChecks(t *testing.T) {
	_ = os.Setenv("GOMYADMIN_SESSION_SECRET", "test-secret")
	t.Cleanup(func() { _ = os.Unsetenv("GOMYADMIN_SESSION_SECRET") })

	checks := Run(context.Background(), Options{DatabaseURL: "postgres://localhost/test"})
	found := false
	for _, c := range checks {
		if c.Name == "GOMYADMIN_SESSION_SECRET" && c.OK {
			found = true
		}
	}
	if !found {
		t.Fatal("expected GOMYADMIN_SESSION_SECRET check with OK=true")
	}
}
