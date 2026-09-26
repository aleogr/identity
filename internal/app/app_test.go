// SPDX-License-Identifier: AGPL-3.0-only

package app_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"strings"
	"testing"

	"github.com/aleogr/identity/internal/app"
	"github.com/aleogr/identity/internal/buildinfo"
)

func run(t *testing.T, args, env []string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := app.Run(t.Context(), args, env, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestCommands(t *testing.T) {
	cases := []struct {
		args     []string
		code     int
		inStdout string
		inStderr string
	}{
		{nil, app.ExitUsage, "", "Usage: identity <command>"},
		{[]string{"bogus"}, app.ExitUsage, "", `unknown command "bogus"`},
		{[]string{"help"}, app.ExitOK, "Usage: identity <command>", ""},
		{[]string{"--help"}, app.ExitOK, "Usage: identity <command>", ""},
		{[]string{"version"}, app.ExitOK, buildinfo.ID() + "\n", ""},
		{[]string{"version", "x"}, app.ExitUsage, "", `unexpected argument "x"`},
		{[]string{"serve", "x"}, app.ExitUsage, "", `unexpected argument "x"`},
	}
	for _, c := range cases {
		code, stdout, stderr := run(t, c.args, nil)
		if code != c.code || !strings.Contains(stdout, c.inStdout) || !strings.Contains(stderr, c.inStderr) {
			t.Errorf("%v: code %d, stdout %q, stderr %q", c.args, code, stdout, stderr)
		}
	}
}

func TestServeRefusesInvalidConfiguration(t *testing.T) {
	code, stdout, stderr := run(t, []string{"serve"}, []string{"IDENTITY_LOG_SOURCE=ture", "IDENTITY_TYPO=1"})
	if code != app.ExitUsage || stdout != "" {
		t.Fatalf("code %d, stdout %q", code, stdout)
	}
	var line struct {
		Severity string   `json:"severity"`
		Message  string   `json:"message"`
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal([]byte(stderr), &line); err != nil {
		t.Fatalf("stderr is not one JSON line: %q", stderr)
	}
	want := []string{`IDENTITY_TYPO="1": unknown variable`, `IDENTITY_LOG_SOURCE="ture": must be true or false`}
	if line.Severity != "ERROR" || line.Message != "invalid configuration; refusing to start" ||
		strings.Join(line.Problems, "|") != strings.Join(want, "|") {
		t.Errorf("line = %+v", line)
	}
}

func TestServeStartsAndStops(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	out, w := io.Pipe()
	done := make(chan int, 1)
	go func() {
		done <- app.Run(ctx, []string{"serve"}, []string{"IDENTITY_HTTP_ADDR=127.0.0.1:0"}, w, io.Discard)
		_ = w.Close()
	}()
	lines := bufio.NewScanner(out)
	if !lines.Scan() {
		t.Fatal("no start-up line")
	}
	var start map[string]string
	if err := json.Unmarshal(lines.Bytes(), &start); err != nil {
		t.Fatal(err)
	}
	if start["message"] != "serving" || start["build_id"] != buildinfo.ID() ||
		!strings.HasPrefix(start["go_version"], "go") || !strings.HasPrefix(start["addr"], "127.0.0.1:") {
		t.Fatalf("start-up line = %v", start)
	}
	cancel()
	var last map[string]string
	for lines.Scan() {
		_ = json.Unmarshal(lines.Bytes(), &last)
	}
	if code := <-done; code != app.ExitOK || last["message"] != "stopped" {
		t.Errorf("code %d, last line %v", code, last)
	}
}

func TestServeFailsWhenTheAddressIsTaken(t *testing.T) {
	taken, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer taken.Close()
	code, stdout, _ := run(t, []string{"serve"}, []string{"IDENTITY_HTTP_ADDR=" + taken.Addr().String()})
	if code != app.ExitFailure || !strings.Contains(stdout, `"message":"cannot listen"`) ||
		!strings.Contains(stdout, taken.Addr().String()) {
		t.Errorf("code %d, stdout %q", code, stdout)
	}
}
