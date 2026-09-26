// SPDX-License-Identifier: AGPL-3.0-only

package logging_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/aleogr/identity/core/secret"
	"github.com/aleogr/identity/internal/logging"
)

func decode(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var lines []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(buf.String()), "\n") {
		if line == "" {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("not JSON: %q: %v", line, err)
		}
		lines = append(lines, m)
	}
	return lines
}

func TestCloudLoggingFields(t *testing.T) {
	var buf bytes.Buffer
	logging.New(&buf, slog.LevelInfo, false).Info("serving", "addr", ":8080")
	lines := decode(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("lines = %v", lines)
	}
	l := lines[0]
	if l["severity"] != "INFO" || l["message"] != "serving" || l["addr"] != ":8080" {
		t.Errorf("line = %v", l)
	}
	for _, k := range []string{"level", "msg", "source", logging.KeySourceLocation} {
		if _, ok := l[k]; ok {
			t.Errorf("unexpected key %q in %v", k, l)
		}
	}
	ts, ok := l["time"].(string)
	if !ok {
		t.Fatalf("time = %v", l["time"])
	}
	parsed, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil || parsed.Location() != time.UTC {
		t.Errorf("time %q is not RFC 3339 in UTC (%v)", ts, err)
	}
}

func TestSeverityMapping(t *testing.T) {
	cases := map[slog.Level]string{
		slog.LevelDebug - 4: "DEBUG",
		slog.LevelDebug:     "DEBUG",
		slog.LevelInfo:      "INFO",
		slog.LevelInfo + 2:  "INFO",
		slog.LevelWarn:      "WARNING",
		slog.LevelError:     "ERROR",
		slog.LevelError + 4: "ERROR",
	}
	for level, want := range cases {
		var buf bytes.Buffer
		logging.New(&buf, slog.LevelDebug-8, false).Log(t.Context(), level, "m")
		if got := decode(t, &buf)[0]["severity"]; got != want {
			t.Errorf("level %v: severity = %v, want %s", level, got, want)
		}
	}
}

func TestLevelFilters(t *testing.T) {
	var buf bytes.Buffer
	l := logging.New(&buf, slog.LevelWarn, false)
	l.Info("hidden")
	l.Warn("shown")
	lines := decode(t, &buf)
	if len(lines) != 1 || lines[0]["message"] != "shown" {
		t.Errorf("lines = %v", lines)
	}
}

func TestSourceLocation(t *testing.T) {
	var buf bytes.Buffer
	logging.New(&buf, slog.LevelInfo, true).Info("m")
	loc, ok := decode(t, &buf)[0][logging.KeySourceLocation].(map[string]any)
	if !ok {
		t.Fatalf("no %s", logging.KeySourceLocation)
	}
	file, _ := loc["file"].(string)
	line, _ := loc["line"].(string)
	fn, _ := loc["function"].(string)
	if !strings.HasSuffix(file, "logging_test.go") || line == "" || !strings.HasSuffix(fn, "TestSourceLocation") {
		t.Errorf("sourceLocation = %v", loc)
	}
}

func TestSecretsAreRedacted(t *testing.T) {
	var buf bytes.Buffer
	logging.New(&buf, slog.LevelInfo, false).Info("m", "client_secret", secret.New("s3cr3t-value"))
	if strings.Contains(buf.String(), "s3cr3t-value") {
		t.Errorf("secret leaked: %s", buf.String())
	}
}

func TestAttributesNamedLikeBuiltInsDoNotPanic(t *testing.T) {
	var buf bytes.Buffer
	logging.New(&buf, slog.LevelInfo, false).Info("m", "time", "yesterday", "level", "high", "source", 3)
	lines := decode(t, &buf)
	if len(lines) != 1 {
		t.Fatalf("lines = %v", lines)
	}
}
