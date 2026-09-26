// SPDX-License-Identifier: AGPL-3.0-only

package main_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

const testBuildID = "test-build-id"

var binary string

// TestMain builds the binary once, with a known build identifier.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "identity-bin-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	binary = filepath.Join(dir, "identity")
	build := exec.CommandContext(context.Background(), "go", "build", "-ldflags",
		"-X github.com/aleogr/identity/internal/buildinfo.version="+testBuildID, "-o", binary, ".")
	build.Stdout, build.Stderr = os.Stderr, os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "building the binary:", err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(dir)
	os.Exit(code)
}

// environ returns the test's environment without the variables the binary
// owns, plus extra.
func environ(extra ...string) []string {
	var env []string
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "IDENTITY_") && !strings.HasPrefix(e, "PORT=") {
			env = append(env, e)
		}
	}
	return append(env, extra...)
}

func TestTypoInABooleanRefusesToStart(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), binary, "serve")
	cmd.Env = environ("IDENTITY_LOG_SOURCE=ture")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 {
		t.Fatalf("exit = %v, want status 2", err)
	}
	var line struct {
		Problems []string `json:"problems"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &line); err != nil {
		t.Fatalf("stderr %q: %v", stderr.String(), err)
	}
	if len(line.Problems) != 1 || line.Problems[0] != `IDENTITY_LOG_SOURCE="ture": must be true or false` {
		t.Errorf("problems = %q", line.Problems)
	}
}

func TestStartsAnnouncesItsBuildAndStopsOnSIGTERM(t *testing.T) {
	cmd := exec.CommandContext(t.Context(), binary, "serve")
	cmd.Env = environ("IDENTITY_HTTP_ADDR=127.0.0.1:0")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill() })

	lines := bufio.NewScanner(stdout)
	if !lines.Scan() {
		t.Fatal("no start-up line")
	}
	var start map[string]string
	if err := json.Unmarshal(lines.Bytes(), &start); err != nil {
		t.Fatal(err)
	}
	if start["message"] != "serving" || start["build_id"] != testBuildID || start["severity"] != "INFO" {
		t.Fatalf("start-up line = %v", start)
	}

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, "http://"+start["addr"]+"/health", nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("health = %d", res.StatusCode)
	}

	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	var last map[string]string
	for lines.Scan() {
		_ = json.Unmarshal(lines.Bytes(), &last)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("exit = %v, want status 0", err)
	}
	if last["message"] != "stopped" {
		t.Errorf("last line = %v", last)
	}
}
