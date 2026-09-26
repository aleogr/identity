// SPDX-License-Identifier: AGPL-3.0-only

//go:build lintrules

// Package lintrules proves that the depguard rules of .golangci.yml refuse the
// imports docs/design.md, section 1.2, forbids. It copies the workspace's Go
// files into a temporary directory, adds probe files that break each rule, and
// runs golangci-lint (the binary named by GOLANGCI_LINT) over them.
// `make check` runs it with the lintrules tag.
package lintrules

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// probe is a file added to the copy, and the depguard rule expected to refuse
// it ("" when no depguard finding is expected).
type probe struct {
	path, content, rule string
}

func source(pkg string, imports ...string) string {
	var b strings.Builder
	b.WriteString("// SPDX-License-Identifier: Apache-2.0\n\npackage " + pkg + "\n\nimport (\n")
	for i, imp := range imports {
		b.WriteString("\tp" + string(rune('a'+i)) + " \"" + imp + "\"\n")
	}
	b.WriteString(")\n\nvar _ = []any{")
	for i := range imports {
		b.WriteString("p" + string(rune('a'+i)) + ".X, ")
	}
	b.WriteString("}\n")
	return b.String()
}

const root = "github.com/aleogr/identity"

var probes = []probe{
	// Files directly in a module's top directory.
	{"core/probe_top.go", source("core", root+"/adapters/postgres"), "core"},
	{"spec/probe_top.go", source("spec", root+"/core"), "spec"},
	{"conformance/probe_top.go", source("conformance", root+"/core"), "conformance"},
	{"adapters/postgres/probe_top.go", source("postgres", root+"/adapters/gcp"), "adapter-postgres"},
	{"adapters/gcp/probe_top.go", source("gcp", root), "adapters-licence"},
	{"adapters/mail/probe_top.go", source("mail", root+"/internal/buildinfo"), "adapters-licence"},
	// Files in a sub-package.
	{"core/probe/p.go", source("probe", root+"/adapters/mail"), "core"},
	{"adapters/mail/probe/p.go", source("probe", root+"/adapters/postgres"), "adapter-mail"},
	{"adapters/postgres/probe2/p.go", source("probe2", root), "adapters-licence"},
	// Randomness.
	{"internal/probe/p.go", strings.Replace(source("probe", "math/rand"), "Apache-2.0", "AGPL-3.0-only", 1), "randomness"},
}

func TestDepguardRefusesForbiddenImports(t *testing.T) {
	linter := os.Getenv("GOLANGCI_LINT")
	if linter == "" {
		t.Fatal("GOLANGCI_LINT must name the golangci-lint binary (make check sets it)")
	}
	repo, err := filepath.Abs("../../../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	copyWorkspace(t, repo, dir)
	for _, p := range probes {
		// Give each imported package an exported X so the probe type-checks.
		write(t, filepath.Join(dir, p.path), p.content)
	}
	for _, pkg := range []string{"core", "spec", "adapters/postgres", "adapters/gcp", "adapters/mail", "internal/buildinfo"} {
		name := filepath.Base(pkg)
		write(t, filepath.Join(dir, pkg, "probe_x.go"), "// SPDX-License-Identifier: Apache-2.0\n\npackage "+name+"\n\n// X lets probes use the package.\nvar X int\n")
	}
	write(t, filepath.Join(dir, "probe_x.go"), "// SPDX-License-Identifier: AGPL-3.0-only\n\npackage identity\n\n// X lets probes use the package.\nvar X int\n")
	write(t, filepath.Join(dir, "internal/probe/x.go"), "// SPDX-License-Identifier: AGPL-3.0-only\n\npackage probe\n")
	// math/rand has no X; the randomness probe uses rand.Int instead.
	write(t, filepath.Join(dir, "internal/probe/p.go"),
		"// SPDX-License-Identifier: AGPL-3.0-only\n\npackage probe\n\nimport \"math/rand\"\n\n// N is predictable.\nfunc N() int { return rand.Int() }\n")

	//nolint:gosec // the linter binary is make check's own GOLANGCI_LINT
	cmd := exec.CommandContext(context.Background(), linter, "run",
		"--config", filepath.Join(dir, ".golangci.yml"),
		"--enable-only", "depguard", "--uniq-by-line=false", "--max-issues-per-linter=0", "--max-same-issues=0",
		"--output.text.path=stdout", "--output.text.print-issued-lines=false",
		"./...", "./spec/...", "./core/...", "./adapters/postgres/...", "./adapters/gcp/...", "./adapters/mail/...", "./conformance/...")
	cmd.Dir = dir
	// The test runs outside the workspace (GOWORK=off); the linter must see it.
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "GOWORK=") {
			cmd.Env = append(cmd.Env, e)
		}
	}
	out, _ := cmd.CombinedOutput()
	report := string(out)
	for _, p := range probes {
		want := p.path + ":"
		found := false
		for _, line := range strings.Split(report, "\n") {
			if strings.HasPrefix(line, want) && strings.Contains(line, "list '"+p.rule+"'") {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: no depguard finding from rule %q", p.path, p.rule)
		}
	}
	if t.Failed() {
		t.Logf("golangci-lint output:\n%s", report)
	}
}

func copyWorkspace(t *testing.T, from, to string) {
	t.Helper()
	out, err := exec.CommandContext(context.Background(), "git", "-C", from, "ls-files", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if strings.HasPrefix(f, "tools/") || strings.HasPrefix(f, "e2e/") {
			continue
		}
		if !isGoInput(f) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(from, f)) //nolint:gosec // paths listed by git in this repository
		if err != nil {
			t.Fatal(err)
		}
		write(t, filepath.Join(to, f), string(data))
	}
}

// isGoInput reports whether f is a file golangci-lint needs to load the workspace.
func isGoInput(f string) bool {
	switch {
	case strings.HasSuffix(f, ".go"), strings.HasSuffix(f, "go.mod"), strings.HasSuffix(f, "go.sum"):
		return true
	default:
		return f == "go.work" || f == ".golangci.yml"
	}
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil { //nolint:gosec // paths inside t.TempDir()
		t.Fatal(err)
	}
}
