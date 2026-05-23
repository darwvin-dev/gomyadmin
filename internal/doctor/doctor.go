package doctor

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Check struct {
	Name    string
	OK      bool
	Message string
}

type Options struct {
	DatabaseURL string
}

func Run(ctx context.Context, options Options) []Check {
	checks := []Check{
		command("Go", "go", "version"),
		command("Node.js", "node", "--version"),
		command("npm", "npm", "--version"),
		optionalCommand("pnpm", "pnpm", "--version"),
		env("DATABASE_URL", options.DatabaseURL),
		env("GOMYADMIN_SESSION_SECRET", os.Getenv("GOMYADMIN_SESSION_SECRET")),
		fileWritable("."),
	}
	if options.DatabaseURL != "" {
		checks = append(checks, postgres(ctx, options.DatabaseURL))
	}
	return checks
}

func command(name, binary string, args ...string) Check {
	out, err := exec.Command(binary, args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(out))
		if message == "" {
			message = fmt.Sprintf("%s not available", binary)
		}
		return Check{Name: name, OK: false, Message: message}
	}
	return Check{Name: name, OK: true, Message: strings.TrimSpace(string(out))}
}

func optionalCommand(name, binary string, args ...string) Check {
	out, err := exec.Command(binary, args...).CombinedOutput()
	if err != nil {
		return Check{Name: name, OK: true, Message: fmt.Sprintf("%s not available (optional)", binary)}
	}
	return Check{Name: name, OK: true, Message: strings.TrimSpace(string(out))}
}

func env(name, value string) Check {
	if value == "" {
		return Check{Name: name, OK: false, Message: "not set"}
	}
	return Check{Name: name, OK: true, Message: "set"}
}

func fileWritable(path string) Check {
	file, err := os.CreateTemp(path, ".gomyadmin-doctor-*")
	if err != nil {
		return Check{Name: "file permissions", OK: false, Message: err.Error()}
	}
	name := file.Name()
	_ = file.Close()
	_ = os.Remove(name)
	return Check{Name: "file permissions", OK: true, Message: "current directory is writable"}
}

func postgres(ctx context.Context, databaseURL string) Check {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return Check{Name: "PostgreSQL", OK: false, Message: err.Error()}
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return Check{Name: "PostgreSQL", OK: false, Message: err.Error()}
	}
	return Check{Name: "PostgreSQL", OK: true, Message: "connection ok"}
}
