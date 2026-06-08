package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/lexis-app/lexis-api/internal/shared/logging"
)

func TestParseLevel(t *testing.T) {
	tests := []struct {
		in   string
		want slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"error", slog.LevelError},
		{"", slog.LevelInfo},
		{"nonsense", slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := logging.ParseLevel(tt.in); got != tt.want {
				t.Errorf("ParseLevel(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestNew_JSONInProduction_RespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, "warn", "production")

	logger.Info("should be filtered")
	logger.Warn("should appear", "k", "v")

	out := buf.String()
	if strings.Contains(out, "should be filtered") {
		t.Errorf("info must be filtered at warn level; got: %s", out)
	}
	// Production handler emits JSON.
	line := strings.TrimSpace(out)
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("production output must be JSON, got %q: %v", line, err)
	}
	if rec["msg"] != "should appear" || rec["k"] != "v" {
		t.Errorf("unexpected log record: %v", rec)
	}
}

func TestNew_TextInDevelopment(t *testing.T) {
	var buf bytes.Buffer
	logger := logging.New(&buf, "debug", "development")
	logger.Debug("dev message")
	if !strings.Contains(buf.String(), "dev message") {
		t.Errorf("debug message should appear in dev text output; got %q", buf.String())
	}
	// Text handler is not JSON.
	if json.Valid([]byte(strings.TrimSpace(buf.String()))) {
		t.Errorf("development output should be text, not JSON: %q", buf.String())
	}
}
