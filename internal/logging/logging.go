// SPDX-License-Identifier: AGPL-3.0-only

// Package logging builds the process logger: one JSON object per line, with
// the field names Cloud Logging reads.
package logging

import (
	"io"
	"log/slog"
	"strconv"
	"time"
)

// Field names Cloud Logging recognises in a JSON log line.
const (
	KeySeverity       = "severity"
	KeyMessage        = "message"
	KeyTime           = "time"
	KeySourceLocation = "logging.googleapis.com/sourceLocation"
)

// New returns a logger writing JSON lines to w, dropping records below level,
// and adding the source location when addSource is true.
func New(w io.Writer, level slog.Leveler, addSource bool) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{
		AddSource:   addSource,
		Level:       level,
		ReplaceAttr: replace,
	}))
}

// replace renames slog's built-in attributes to Cloud Logging's. It checks
// each value's kind, so a caller's own attribute named "time" or "level" is
// left alone rather than panicking.
func replace(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.TimeKey:
		if a.Value.Kind() == slog.KindTime {
			return slog.String(KeyTime, a.Value.Time().UTC().Format(time.RFC3339Nano))
		}
	case slog.LevelKey:
		if level, ok := a.Value.Any().(slog.Level); ok {
			return slog.String(KeySeverity, severity(level))
		}
	case slog.MessageKey:
		return slog.Attr{Key: KeyMessage, Value: a.Value}
	case slog.SourceKey:
		if src, ok := a.Value.Any().(*slog.Source); ok {
			return slog.Group(KeySourceLocation,
				slog.String("file", src.File),
				slog.String("line", strconv.Itoa(src.Line)),
				slog.String("function", src.Function))
		}
	}
	return a
}

func severity(l slog.Level) string {
	switch {
	case l < slog.LevelInfo:
		return "DEBUG"
	case l < slog.LevelWarn:
		return "INFO"
	case l < slog.LevelError:
		return "WARNING"
	default:
		return "ERROR"
	}
}
