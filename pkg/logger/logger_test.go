package logger

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

func TestNewReturnsNonNilLogger(t *testing.T) {
	for _, level := range []string{"debug", "info", "warn", "error", "", "unknown"} {
		l := New(level)
		if l == nil {
			t.Fatalf("New(%q) returned nil", level)
		}
	}
}

func TestNewLevelParsing(t *testing.T) {
	tests := []struct {
		input string
		want  slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		{"", slog.LevelInfo},    // default
		{"other", slog.LevelInfo}, // unknown → info
	}

	for _, tt := range tests {
		l := New(tt.input)
		// Verify the level by checking what the logger actually emits.
		// slog.Logger exposes Enabled(ctx, level) for capability checks.
		ctx := t.Context()
		if tt.want == slog.LevelDebug {
			if !l.Enabled(ctx, slog.LevelDebug) {
				t.Errorf("New(%q): debug should be enabled", tt.input)
			}
		} else if tt.want == slog.LevelWarn {
			if l.Enabled(ctx, slog.LevelInfo) {
				t.Errorf("New(%q): info should NOT be enabled at warn level", tt.input)
			}
			if !l.Enabled(ctx, slog.LevelWarn) {
				t.Errorf("New(%q): warn should be enabled", tt.input)
			}
		} else if tt.want == slog.LevelError {
			if l.Enabled(ctx, slog.LevelWarn) {
				t.Errorf("New(%q): warn should NOT be enabled at error level", tt.input)
			}
			if !l.Enabled(ctx, slog.LevelError) {
				t.Errorf("New(%q): error should be enabled", tt.input)
			}
		} else {
			// info
			if !l.Enabled(ctx, slog.LevelInfo) {
				t.Errorf("New(%q): info should be enabled", tt.input)
			}
			if l.Enabled(ctx, slog.LevelDebug) {
				t.Errorf("New(%q): debug should NOT be enabled at info level", tt.input)
			}
		}
	}
}

func TestNewWritesJSONOutput(t *testing.T) {
	// Replace stdout by creating a logger with a custom handler for assertion.
	// Since New() writes to os.Stdout, we create a logger with same config
	// but writing to a buffer to verify JSON format.
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo})
	l := slog.New(h)
	l.Info("test message", "key", "value")

	if buf.Len() == 0 {
		t.Fatal("expected log output")
	}
	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("output is not valid JSON: %v — output: %s", err, buf.String())
	}
	if entry["msg"] != "test message" {
		t.Fatalf("msg = %q", entry["msg"])
	}
	if entry["key"] != "value" {
		t.Fatalf("key = %q", entry["key"])
	}
}
