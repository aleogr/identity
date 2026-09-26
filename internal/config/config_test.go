// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/aleogr/identity/internal/config"
)

func TestDefaults(t *testing.T) {
	cfg, err := config.Load([]string{"HOME=/root", "PATH=/bin", "NOEQUALSSIGN"})
	if err != nil {
		t.Fatal(err)
	}
	want := config.Config{HTTPAddr: ":8080", LogLevel: slog.LevelInfo, LogSource: false, ShutdownTimeout: 8 * time.Second}
	if cfg != want || config.Default() != want {
		t.Errorf("Load() = %+v, Default() = %+v, want %+v", cfg, config.Default(), want)
	}
}

func TestValidValues(t *testing.T) {
	cases := []struct {
		env   string
		check func(config.Config) bool
	}{
		{"IDENTITY_HTTP_ADDR=127.0.0.1:0", func(c config.Config) bool { return c.HTTPAddr == "127.0.0.1:0" }},
		{"IDENTITY_HTTP_ADDR=[::1]:9000", func(c config.Config) bool { return c.HTTPAddr == "[::1]:9000" }},
		{"IDENTITY_HTTP_ADDR=localhost:9000", func(c config.Config) bool { return c.HTTPAddr == "localhost:9000" }},
		{"IDENTITY_HTTP_ADDR=:65535", func(c config.Config) bool { return c.HTTPAddr == ":65535" }},
		{"PORT=9090", func(c config.Config) bool { return c.HTTPAddr == ":9090" }},
		{"IDENTITY_LOG_LEVEL=debug", func(c config.Config) bool { return c.LogLevel == slog.LevelDebug }},
		{"IDENTITY_LOG_LEVEL=warn", func(c config.Config) bool { return c.LogLevel == slog.LevelWarn }},
		{"IDENTITY_LOG_LEVEL=error", func(c config.Config) bool { return c.LogLevel == slog.LevelError }},
		{"IDENTITY_LOG_SOURCE=true", func(c config.Config) bool { return c.LogSource }},
		{"IDENTITY_LOG_SOURCE=false", func(c config.Config) bool { return !c.LogSource }},
		{"IDENTITY_SHUTDOWN_TIMEOUT=1s", func(c config.Config) bool { return c.ShutdownTimeout == time.Second }},
		{"IDENTITY_SHUTDOWN_TIMEOUT=60s", func(c config.Config) bool { return c.ShutdownTimeout == time.Minute }},
		{"IDENTITY_SHUTDOWN_TIMEOUT=1m", func(c config.Config) bool { return c.ShutdownTimeout == time.Minute }},
	}
	for _, c := range cases {
		cfg, err := config.Load([]string{c.env})
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.env, err)
			continue
		}
		if !c.check(cfg) {
			t.Errorf("%s: got %+v", c.env, cfg)
		}
	}
}

func TestInvalidValues(t *testing.T) {
	cases := []struct {
		env  []string
		want string // the single problem, as Problem.String renders it
	}{
		{[]string{"IDENTITY_LOG_SOURCE=ture"}, `IDENTITY_LOG_SOURCE="ture": must be true or false`},
		{[]string{"IDENTITY_LOG_SOURCE=1"}, `IDENTITY_LOG_SOURCE="1": must be true or false`},
		{[]string{"IDENTITY_LOG_SOURCE=TRUE"}, `IDENTITY_LOG_SOURCE="TRUE": must be true or false`},
		{[]string{"IDENTITY_LOG_SOURCE="}, `IDENTITY_LOG_SOURCE="": must be true or false`},
		{[]string{"IDENTITY_LOG_LEVEL=INFO"}, `IDENTITY_LOG_LEVEL="INFO": must be one of debug, info, warn, error`},
		{[]string{"IDENTITY_SHUTDOWN_TIMEOUT=500ms"}, `IDENTITY_SHUTDOWN_TIMEOUT="500ms": must be between 1s and 1m0s`},
		{[]string{"IDENTITY_SHUTDOWN_TIMEOUT=61s"}, `IDENTITY_SHUTDOWN_TIMEOUT="61s": must be between 1s and 1m0s`},
		{[]string{"IDENTITY_SHUTDOWN_TIMEOUT=8"}, `IDENTITY_SHUTDOWN_TIMEOUT="8": must be a duration such as 8s`},
		{[]string{"IDENTITY_HTTP_ADDR=8080"}, `IDENTITY_HTTP_ADDR="8080": must be host:port, such as :8080`},
		{[]string{"IDENTITY_HTTP_ADDR=example.com:80"}, `IDENTITY_HTTP_ADDR="example.com:80": the host must be empty, localhost or an IP address`},
		{[]string{"IDENTITY_HTTP_ADDR=:99999"}, `IDENTITY_HTTP_ADDR=":99999": the port must be a number from 0 to 65535`},
		{[]string{"IDENTITY_HTTP_ADDR=:+80"}, `IDENTITY_HTTP_ADDR=":+80": the port must be a number from 0 to 65535`},
		{[]string{"PORT=abc"}, `PORT="abc": must be a number from 1 to 65535`},
		{[]string{"PORT=0"}, `PORT="0": must be a number from 1 to 65535`},
		{[]string{"PORT=+80"}, `PORT="+80": must be a number from 1 to 65535`},
		{[]string{"IDENTITY_HTTP_ADDR=:8080", "PORT=8080"}, `IDENTITY_HTTP_ADDR=":8080": conflicts with PORT; set only one of them`},
		{[]string{"IDENTITY_LOG_SOURC=true"}, `IDENTITY_LOG_SOURC="true": unknown variable`},
	}
	for _, c := range cases {
		_, err := config.Load(c.env)
		var cerr *config.Error
		if !errors.As(err, &cerr) {
			t.Errorf("%v: error = %v, want *config.Error", c.env, err)
			continue
		}
		if len(cerr.Problems) != 1 || cerr.Problems[0].String() != c.want {
			t.Errorf("%v:\n got %v\nwant [%s]", c.env, cerr.Problems, c.want)
		}
	}
}

func TestEveryProblemIsReported(t *testing.T) {
	_, err := config.Load([]string{
		"IDENTITY_ZZZ=1", "IDENTITY_AAA=2",
		"IDENTITY_LOG_SOURCE=ture", "IDENTITY_SHUTDOWN_TIMEOUT=forever",
	})
	var cerr *config.Error
	if !errors.As(err, &cerr) {
		t.Fatalf("error = %v", err)
	}
	var got []string
	for _, p := range cerr.Problems {
		got = append(got, p.Variable)
	}
	want := []string{"IDENTITY_AAA", "IDENTITY_ZZZ", "IDENTITY_LOG_SOURCE", "IDENTITY_SHUTDOWN_TIMEOUT"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("variables = %v, want %v", got, want)
	}
	if !strings.HasPrefix(err.Error(), "invalid configuration: IDENTITY_AAA=\"2\": unknown variable; ") {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestUnknownNamesAreQuoted(t *testing.T) {
	_, err := config.Load([]string{"IDENTITY_X\nY=1"})
	var cerr *config.Error
	if !errors.As(err, &cerr) {
		t.Fatal(err)
	}
	if got := cerr.Problems[0].String(); got != `"IDENTITY_X\nY"="1": unknown variable` {
		t.Errorf("String() = %q", got)
	}
}

func TestLongAndControlValuesAreQuotedAndTruncated(t *testing.T) {
	long := strings.Repeat("é", 100) // 200 bytes
	_, err := config.Load([]string{"IDENTITY_LOG_LEVEL=" + long})
	var cerr *config.Error
	if !errors.As(err, &cerr) {
		t.Fatal(err)
	}
	got := cerr.Problems[0].String()
	if !strings.Contains(got, `..."`) && !strings.Contains(got, `"...`) {
		t.Errorf("long value not truncated: %q", got)
	}
	if len(got) > 200 {
		t.Errorf("rendering is %d bytes long: %q", len(got), got)
	}

	_, err = config.Load([]string{"IDENTITY_LOG_LEVEL=info\n{\"severity\":\"ERROR\"}\x1b[31m"})
	if !errors.As(err, &cerr) {
		t.Fatal(err)
	}
	got = cerr.Problems[0].String()
	if strings.ContainsAny(got, "\n\x1b") {
		t.Errorf("control characters not escaped: %q", got)
	}
}
