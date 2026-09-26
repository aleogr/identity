# F1 — Repository skeleton, modules, pipeline and build identifier: Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A Go workspace whose module boundaries are enforced by lint, a binary that starts, announces its build and answers `/health`, and a CI pipeline of six named jobs that runs every check of `CLAUDE.md`.

**Architecture:** Seven workspace modules (the AGPL root with `cmd/identity` and `internal/*`, and six Apache modules). Tools live outside the workspace, one module per tool, built into `bin/tools/` and run from there. The binary is layered: `internal/config` (strict environment parsing) → `internal/logging` (Cloud Logging JSON) → `internal/server` (HTTP and graceful shutdown) → `internal/app` (commands) → `cmd/identity` (signals and exit). `core/secret` is the one piece of `core` in F1.

**Tech Stack:** Go 1.27.1 (standard library only in product code), golangci-lint v2.14.0, staticcheck 0.8.1 (2026.2.1), gosec v2.29.0, govulncheck v1.8.0, gitleaks v8.30.1, actionlint v1.7.12, ko v0.19.1, zizmor 1.30.1, `go.yaml.in/yaml/v3` v3.0.5 (tools only), Python 3.11 with pytest 9.1.1 and Playwright 1.56.0, GitHub Actions, CodeQL, Trivy v0.74.0.

**Spec:** `docs/superpowers/specs/2026-09-26-f1-repository-skeleton-design.md` — read it before any task.

## Global Constraints

- Everything versioned is in English. No model identifier anywhere except commit trailers.
- Every `.go`, `.py`, `.sh` file and every `Makefile` starts with an SPDX identifier (after a shebang or `//go:build` line): `// SPDX-License-Identifier: Apache-2.0` under `spec/`, `core/`, `adapters/`, `conformance/`; `// SPDX-License-Identifier: AGPL-3.0-only` everywhere else (`#` instead of `//` in Python, shell and the `Makefile`). `.claude/` is exempt.
- Workspace modules declare `go 1.27.0`; `go.work` and the root `go.mod` also declare `toolchain go1.27.1`.
- `core` imports only the standard library, `golang.org/x/crypto`, `spec` and `core`. No product code imports anything outside the standard library in F1.
- `math/rand` and `math/rand/v2` are never imported outside `_test.go` files.
- Tests never use real external services.
- Configuration errors exit with status 2, run-time failures with status 1.
- Never skip, disable or weaken a check to get green. A lint exclusion needs a written reason next to it.
- Every commit ends with the trailer:

  ```
  Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>
  Claude-Session: https://claude.ai/code/session_01MK1diph3ZZociszxhZevtT
  ```

- Commit on branch `claude/beautiful-cannon-vzf0sf`; push only in Task 15.
- In this session, export `GOTOOLCHAIN=auto` (the default) and let `go` fetch go1.27.1; `PLAYWRIGHT_BROWSERS_PATH=/opt/pw-browsers` is already set.

## Review Focus

1. **A struct holding `secret.String` in an unexported field printed with `%+v` or `%#v`** — `fmt` cannot call methods on unexported fields and would print the raw struct; the value must not appear. Test in Task 5.
2. **A log attribute a caller names `time` or `level` whose value is not a time or a level** — the Cloud Logging key rewriter must not panic, and must leave it alone. Test in Task 8.
3. **A configuration value that is very long or holds control characters** — the error must quote it (escaping control characters) and truncate it, never flood or garble the log. Test in Task 7.
4. **`PORT=+80`, `PORT=0` or `IDENTITY_HTTP_ADDR=example.com:80`** — values `strconv.Atoi` or `net.Listen` would accept must be refused as configuration errors (status 2), not surface later. Test in Task 7.
5. **The listening address already taken** — `identity serve` must exit with status 1 and a log line naming the address, not hang. Test in Task 10.

---

### Task 1: The workspace, the modules and their licences

**Files:**
- Create: `go.work`, `go.mod`, `doc.go`
- Create: `spec/go.mod`, `spec/doc.go`, `spec/LICENSE`
- Create: `core/go.mod`, `core/doc.go`, `core/LICENSE`
- Create: `adapters/postgres/go.mod`, `adapters/postgres/doc.go`, `adapters/postgres/LICENSE`
- Create: `adapters/gcp/go.mod`, `adapters/gcp/doc.go`, `adapters/gcp/LICENSE`
- Create: `adapters/mail/go.mod`, `adapters/mail/doc.go`, `adapters/mail/LICENSE`
- Create: `conformance/go.mod`, `conformance/doc.go`, `conformance/LICENSE`
- Modify: `.gitignore`

**Interfaces:**
- Produces: module paths `github.com/aleogr/identity`, `.../spec`, `.../core`, `.../adapters/postgres`, `.../adapters/gcp`, `.../adapters/mail`, `.../conformance`.

- [ ] **Step 1: Write `go.work`**

```
go 1.27.0

toolchain go1.27.1

use (
	.
	./adapters/gcp
	./adapters/mail
	./adapters/postgres
	./conformance
	./core
	./spec
)
```

- [ ] **Step 2: Write the root `go.mod` and `doc.go`**

`go.mod`:

```
module github.com/aleogr/identity

go 1.27.0

toolchain go1.27.1

require github.com/aleogr/identity/core v0.0.0

// The root module is the server, never imported as a library, so it may
// replace its own libraries with their directories. Builds outside the
// workspace (ko, Dependabot) resolve them through these lines.
replace github.com/aleogr/identity/core => ./core
```

`doc.go`:

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Package identity is the root of the identity platform's server module. The
// binary is in cmd/identity; the server's packages are in internal/.
package identity
```

- [ ] **Step 3: Write each Apache module**

For each `(dir, package, sentence)` below, write `<dir>/go.mod`:

```
module github.com/aleogr/identity/<dir>

go 1.27.0
```

and `<dir>/doc.go`:

```go
// SPDX-License-Identifier: Apache-2.0

// Package <package> <sentence>
package <package>
```

| dir | package | sentence |
|---|---|---|
| `spec` | `spec` | `holds the platform's contracts: the domain model, token formats, the event schema and the hook contract.` |
| `core` | `core` | `holds the platform's domain logic. It performs no I/O: every effect goes through a port it defines.` |
| `adapters/postgres` | `postgres` | `implements core's storage ports on PostgreSQL.` |
| `adapters/gcp` | `gcp` | `implements core's ports on Google Cloud: KMS, Secret Manager, Cloud Tasks and Cloud Scheduler.` |
| `adapters/mail` | `mail` | `implements core's mail port.` |
| `conformance` | `conformance` | `is a black-box HTTP suite that checks a deployment against the protocols it implements.` |

Then copy the licence text into each: `cp LICENSES/Apache-2.0.txt <dir>/LICENSE`.

- [ ] **Step 4: Extend `.gitignore`**

Append:

```
# Python virtual environments and end-to-end artefacts
.venv/
e2e/artifacts/
```

(`/bin/` and `/dist/` are already ignored.)

- [ ] **Step 5: Verify every module builds and vets**

Run: `go vet ./... ./spec/... ./core/... ./adapters/postgres/... ./adapters/gcp/... ./adapters/mail/... ./conformance/... && go version`
Expected: no output from `go vet`, then `go version go1.27.1 linux/amd64`.

- [ ] **Step 6: Commit**

```bash
git add go.work go.mod doc.go spec core adapters conformance .gitignore
git commit -m "Add the Go workspace and the modules of phase 1"
```

---

### Task 2: `spdxcheck`, the licence identifier checker

**Files:**
- Create: `tools/checks/go.mod`, `tools/checks/LICENSE`
- Create: `tools/checks/internal/spdx/spdx.go`, `tools/checks/internal/spdx/spdx_test.go`
- Create: `tools/checks/cmd/spdxcheck/main.go`

**Interfaces:**
- Produces: `spdx.Expected(path string) (id string, covered bool)`, `spdx.Identifier(content []byte) string`, `spdx.Check(fsys fs.FS, files []string) ([]spdx.Finding, error)`, `spdx.Finding{Path, Problem string}` with `String()`. Binary `spdxcheck` (no arguments; run from the repository root; exit 1 on findings).

- [ ] **Step 1: Create the module**

```bash
mkdir -p tools/checks && cp LICENSE tools/checks/LICENSE
cat > tools/checks/go.mod <<'EOF'
module github.com/aleogr/identity/tools/checks

go 1.27.0
EOF
```

- [ ] **Step 2: Write the failing tests** — `tools/checks/internal/spdx/spdx_test.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

package spdx_test

import (
	"reflect"
	"testing"
	"testing/fstest"

	"github.com/aleogr/identity/tools/checks/internal/spdx"
)

func TestExpected(t *testing.T) {
	cases := []struct {
		path    string
		want    string
		covered bool
	}{
		{"core/secret/secret.go", spdx.Apache, true},
		{"spec/doc.go", spdx.Apache, true},
		{"adapters/gcp/doc.go", spdx.Apache, true},
		{"conformance/doc.go", spdx.Apache, true},
		{"lib/doc.go", spdx.Apache, true},
		{"sdk/python/client.py", spdx.Apache, true},
		{"cmd/identity/main.go", spdx.AGPL, true},
		{"internal/config/config.go", spdx.AGPL, true},
		{"e2e/conftest.py", spdx.AGPL, true},
		{"Makefile", spdx.AGPL, true},
		{"scripts/x.sh", spdx.AGPL, true},
		{"tools/checks/cmd/spdxcheck/main.go", spdx.AGPL, true},
		{"internal/core/x.go", spdx.AGPL, true},
		{"README.md", "", false},
		{".github/workflows/ci.yml", "", false},
		{".claude/skills/webapp-testing/scripts/with_server.py", "", false},
		{"core/secret/testdata/fuzz/FuzzX/abc", "", false},
		{"internal/x/testdata/sample.go", "", false},
	}
	for _, c := range cases {
		got, covered := spdx.Expected(c.path)
		if got != c.want || covered != c.covered {
			t.Errorf("Expected(%q) = %q, %v; want %q, %v", c.path, got, covered, c.want, c.covered)
		}
	}
}

func TestIdentifier(t *testing.T) {
	cases := []struct {
		name, content, want string
	}{
		{"go", "// SPDX-License-Identifier: Apache-2.0\n\npackage x\n", "Apache-2.0"},
		{"build constraint first", "//go:build integration\n\n// SPDX-License-Identifier: AGPL-3.0-only\n\npackage x\n", "AGPL-3.0-only"},
		{"shebang first", "#!/usr/bin/env bash\n# SPDX-License-Identifier: AGPL-3.0-only\n", "AGPL-3.0-only"},
		{"python", "# SPDX-License-Identifier: AGPL-3.0-only\n\"\"\"Doc.\"\"\"\n", "AGPL-3.0-only"},
		{"missing", "package x\n", ""},
		{"not first", "package x\n\n// SPDX-License-Identifier: Apache-2.0\n", ""},
		{"empty", "", ""},
	}
	for _, c := range cases {
		if got := spdx.Identifier([]byte(c.content)); got != c.want {
			t.Errorf("%s: Identifier() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestCheck(t *testing.T) {
	fsys := fstest.MapFS{
		"go.mod":                    {Data: []byte("module x\n")},
		"LICENSE":                   {Data: []byte("AGPL")},
		"cmd/identity/main.go":      {Data: []byte("// SPDX-License-Identifier: AGPL-3.0-only\n\npackage main\n")},
		"core/go.mod":               {Data: []byte("module x/core\n")},
		"core/doc.go":               {Data: []byte("// SPDX-License-Identifier: AGPL-3.0-only\n\npackage core\n")},
		"core/secret/secret.go":     {Data: []byte("package secret\n")},
		"spec/go.mod":               {Data: []byte("module x/spec\n")},
		"spec/LICENSE":              {Data: []byte("Apache")},
		"spec/doc.go":               {Data: []byte("// SPDX-License-Identifier: Apache-2.0\n\npackage spec\n")},
		"tools/ko/go.mod":           {Data: []byte("module x/tools/ko\n")},
		"README.md":                 {Data: []byte("# x\n")},
	}
	files := []string{
		"go.mod", "LICENSE", "cmd/identity/main.go",
		"core/go.mod", "core/doc.go", "core/secret/secret.go",
		"spec/go.mod", "spec/LICENSE", "spec/doc.go",
		"tools/ko/go.mod", "README.md",
		"deleted/but/listed.go",
	}
	got, err := spdx.Check(fsys, files)
	if err != nil {
		t.Fatal(err)
	}
	want := []spdx.Finding{
		{Path: "core", Problem: "Go module without a LICENSE file"},
		{Path: "core/doc.go", Problem: "SPDX-License-Identifier is AGPL-3.0-only, expected Apache-2.0"},
		{Path: "core/secret/secret.go", Problem: "missing SPDX-License-Identifier (expected Apache-2.0)"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Check() =\n%v\nwant\n%v", got, want)
	}
}

func TestFindingString(t *testing.T) {
	f := spdx.Finding{Path: "a.go", Problem: "p"}
	if got := f.String(); got != "a.go: p" {
		t.Errorf("String() = %q", got)
	}
}
```

- [ ] **Step 3: Run the tests to see them fail**

Run: `cd tools/checks && GOWORK=off go test ./...`
Expected: FAIL — `no required module provides package .../internal/spdx` or `undefined: spdx.Expected`.

- [ ] **Step 4: Implement** — `tools/checks/internal/spdx/spdx.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Package spdx checks that every source file carries the SPDX licence
// identifier its directory requires (LICENSING.md), and that every Go module
// with source code carries a LICENSE file.
package spdx

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// The two licences of the repository.
const (
	Apache = "Apache-2.0"
	AGPL   = "AGPL-3.0-only"
)

// apacheRoots are the directories LICENSING.md places under Apache-2.0.
var apacheRoots = []string{"spec/", "core/", "adapters/", "conformance/", "lib/", "sdk/"}

// Finding is one violation.
type Finding struct {
	Path    string
	Problem string
}

func (f Finding) String() string { return f.Path + ": " + f.Problem }

// Expected returns the identifier the file at p (slash-separated, relative to
// the repository root) must carry, and whether the rule covers it at all.
func Expected(p string) (string, bool) {
	if strings.HasPrefix(p, ".claude/") || strings.Contains("/"+p, "/testdata/") {
		return "", false
	}
	base := path.Base(p)
	if base != "Makefile" && !strings.HasSuffix(p, ".go") && !strings.HasSuffix(p, ".py") && !strings.HasSuffix(p, ".sh") {
		return "", false
	}
	for _, root := range apacheRoots {
		if strings.HasPrefix(p, root) {
			return Apache, true
		}
	}
	return AGPL, true
}

// Identifier returns the SPDX identifier content declares on its first line
// that is not blank, a shebang or a build constraint, or "" when that line
// declares none.
func Identifier(content []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(content))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#!") || strings.HasPrefix(line, "//go:build") {
			continue
		}
		for _, prefix := range []string{"// SPDX-License-Identifier: ", "# SPDX-License-Identifier: "} {
			if rest, ok := strings.CutPrefix(line, prefix); ok {
				return strings.TrimSpace(rest)
			}
		}
		return ""
	}
	return ""
}

// Check reads every covered file of files from fsys and returns every
// violation, sorted by path. Listed files that no longer exist are skipped.
func Check(fsys fs.FS, files []string) ([]Finding, error) {
	var findings []Finding
	modules := map[string]bool{}  // directories holding a go.mod
	licences := map[string]bool{} // directories holding a LICENSE
	var goFiles []string
	for _, f := range files {
		switch path.Base(f) {
		case "go.mod":
			modules[path.Dir(f)] = true
		case "LICENSE":
			licences[path.Dir(f)] = true
		}
		if strings.HasSuffix(f, ".go") {
			goFiles = append(goFiles, f)
		}
		want, covered := Expected(f)
		if !covered {
			continue
		}
		content, err := fs.ReadFile(fsys, f)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("spdx: %w", err)
		}
		switch got := Identifier(content); {
		case got == "":
			findings = append(findings, Finding{f, "missing SPDX-License-Identifier (expected " + want + ")"})
		case got != want:
			findings = append(findings, Finding{f, "SPDX-License-Identifier is " + got + ", expected " + want})
		}
	}
	withSource := map[string]bool{}
	for _, f := range goFiles {
		if m, ok := moduleOf(path.Dir(f), modules); ok {
			withSource[m] = true
		}
	}
	for m := range withSource {
		if !licences[m] {
			findings = append(findings, Finding{m, "Go module without a LICENSE file"})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Problem < findings[j].Problem
	})
	return findings, nil
}

// moduleOf returns the nearest directory at or above dir that holds a go.mod.
func moduleOf(dir string, modules map[string]bool) (string, bool) {
	for {
		if modules[dir] {
			return dir, true
		}
		if dir == "." || dir == "/" {
			return "", false
		}
		dir = path.Dir(dir)
	}
}
```

- [ ] **Step 5: Run the tests to see them pass**

Run: `cd tools/checks && GOWORK=off go test -race ./...`
Expected: `ok  github.com/aleogr/identity/tools/checks/internal/spdx`

- [ ] **Step 6: Write the command** — `tools/checks/cmd/spdxcheck/main.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Command spdxcheck checks the licence identifiers of every file git knows
// about (tracked, or untracked and not ignored). Run it from the repository
// root; it exits with status 1 when it finds a violation.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/aleogr/identity/tools/checks/internal/spdx"
)

func main() {
	out, err := exec.CommandContext(context.Background(), "git", "ls-files", "-z", "--cached", "--others", "--exclude-standard").Output()
	if err != nil {
		fmt.Fprintln(os.Stderr, "spdxcheck: git ls-files:", err)
		os.Exit(2)
	}
	var files []string
	for _, f := range bytes.Split(bytes.TrimRight(out, "\x00"), []byte{0}) {
		if len(f) > 0 {
			files = append(files, string(f))
		}
	}
	findings, err := spdx.Check(os.DirFS("."), files)
	if err != nil {
		fmt.Fprintln(os.Stderr, "spdxcheck:", err)
		os.Exit(2)
	}
	for _, f := range findings {
		fmt.Println(f)
	}
	if len(findings) > 0 {
		fmt.Fprintf(os.Stderr, "spdxcheck: %d violation(s)\n", len(findings))
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Run it on the repository**

Run: `GOWORK=off go build -C tools/checks -o "$PWD/bin/tools/" ./cmd/... && bin/tools/spdxcheck; echo "exit $?"`
Expected: `exit 0` (Task 1's files all carry identifiers; `tools/checks` has its `LICENSE`).

- [ ] **Step 8: Commit**

```bash
git add tools/checks
git commit -m "Add spdxcheck, which enforces the licence identifier of every source file"
```

---

### Task 3: `trivyignorecheck`, the exception policy checker

**Files:**
- Create: `tools/checks/internal/trivyignore/trivyignore.go`, `tools/checks/internal/trivyignore/trivyignore_test.go`
- Create: `tools/checks/cmd/trivyignorecheck/main.go`
- Create: `.trivyignore.yaml`
- Modify: `tools/checks/go.mod`, create `tools/checks/go.sum`

**Interfaces:**
- Produces: `trivyignore.MaxDays = 90`, `trivyignore.Check(content []byte, today time.Time) ([]string, error)`. Binary `trivyignorecheck [file]` (default `.trivyignore.yaml`; a missing file is not an error; exit 1 on problems).

- [ ] **Step 1: Add the YAML dependency**

Run: `cd tools/checks && GOWORK=off go get go.yaml.in/yaml/v3@v3.0.5`
Expected: `go: added go.yaml.in/yaml/v3 v3.0.5`.

- [ ] **Step 2: Write the failing tests** — `tools/checks/internal/trivyignore/trivyignore_test.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

package trivyignore_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/aleogr/identity/tools/checks/internal/trivyignore"
)

var today = time.Date(2026, 9, 26, 15, 4, 5, 0, time.UTC)

func TestCheck(t *testing.T) {
	cases := []struct {
		name, content string
		want          []string
	}{
		{"empty file", "", nil},
		{"no exceptions", "vulnerabilities: []\n", nil},
		{"valid entry", `vulnerabilities:
  - id: CVE-2026-0001
    statement: The vulnerable function is not reachable; govulncheck reports no call path.
    expired_at: 2026-12-01
`, nil},
		{"exactly 90 days ahead", `vulnerabilities:
  - id: CVE-2026-0001
    statement: Justified.
    expired_at: 2026-12-25
`, nil},
		{"91 days ahead", `vulnerabilities:
  - id: CVE-2026-0001
    statement: Justified.
    expired_at: 2026-12-26
`, []string{"vulnerabilities[0] CVE-2026-0001: expires on 2026-12-26, more than 90 days after 2026-09-26"}},
		{"expires today", `vulnerabilities:
  - id: CVE-2026-0001
    statement: Justified.
    expired_at: 2026-09-26
`, []string{"vulnerabilities[0] CVE-2026-0001: expired on 2026-09-26"}},
		{"no statement", `vulnerabilities:
  - id: CVE-2026-0001
    statement: "   "
    expired_at: 2026-12-01
`, []string{"vulnerabilities[0] CVE-2026-0001: has no statement justifying the exception"}},
		{"no date", `vulnerabilities:
  - id: CVE-2026-0001
    statement: Justified.
`, []string{"vulnerabilities[0] CVE-2026-0001: has no expired_at date"}},
		{"bad date", `vulnerabilities:
  - id: CVE-2026-0001
    statement: Justified.
    expired_at: next year
`, []string{`vulnerabilities[0] CVE-2026-0001: expired_at "next year" is not a date (YYYY-MM-DD)`}},
		{"no id, other section", `secrets:
  - statement: Justified.
    expired_at: 2026-12-01
`, []string{"secrets[0] : has no id"}},
	}
	for _, c := range cases {
		got, err := trivyignore.Check([]byte(c.content), today)
		if err != nil {
			t.Errorf("%s: unexpected error %v", c.name, err)
			continue
		}
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %q\nwant %q", c.name, got, c.want)
		}
	}
}

func TestCheckRefusesUnknownFields(t *testing.T) {
	_, err := trivyignore.Check([]byte("vulnerabilities:\n  - id: X\n    reason: typo of statement\n"), today)
	if err == nil {
		t.Fatal("expected an error for an unknown field")
	}
}
```

- [ ] **Step 3: Run the tests to see them fail**

Run: `cd tools/checks && GOWORK=off go test ./internal/trivyignore/`
Expected: FAIL — `undefined: trivyignore.Check`.

- [ ] **Step 4: Implement** — `tools/checks/internal/trivyignore/trivyignore.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Package trivyignore enforces the exception policy for Trivy findings
// (docs/threat-model.md, B7-01): every entry of .trivyignore.yaml carries a
// written justification and an expiry date at most MaxDays ahead.
package trivyignore

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"
)

// MaxDays is the longest an exception may run from the day it is checked.
const MaxDays = 90

const dateLayout = "2006-01-02"

type entry struct {
	ID        string   `yaml:"id"`
	Paths     []string `yaml:"paths"`
	PURLs     []string `yaml:"purls"`
	Statement string   `yaml:"statement"`
	ExpiredAt string   `yaml:"expired_at"`
}

type file struct {
	Vulnerabilities   []entry `yaml:"vulnerabilities"`
	Misconfigurations []entry `yaml:"misconfigurations"`
	Secrets           []entry `yaml:"secrets"`
	Licenses          []entry `yaml:"licenses"`
}

// Check validates the content of a .trivyignore.yaml file as of today and
// returns one line per problem. Unknown fields are an error, so a misspelt
// key cannot silently drop a statement or a date.
func Check(content []byte, today time.Time) ([]string, error) {
	var f file
	dec := yaml.NewDecoder(bytes.NewReader(content))
	dec.KnownFields(true)
	if err := dec.Decode(&f); err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("trivyignore: %w", err)
	}
	y, m, d := today.UTC().Date()
	day := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	limit := day.AddDate(0, 0, MaxDays)

	var problems []string
	sections := []struct {
		name    string
		entries []entry
	}{
		{"vulnerabilities", f.Vulnerabilities},
		{"misconfigurations", f.Misconfigurations},
		{"secrets", f.Secrets},
		{"licenses", f.Licenses},
	}
	for _, s := range sections {
		for i, e := range s.entries {
			where := fmt.Sprintf("%s[%d] %s", s.name, i, e.ID)
			report := func(problem string) { problems = append(problems, where+": "+problem) }
			if strings.TrimSpace(e.ID) == "" {
				report("has no id")
			}
			if strings.TrimSpace(e.Statement) == "" {
				report("has no statement justifying the exception")
			}
			if e.ExpiredAt == "" {
				report("has no expired_at date")
				continue
			}
			expires, err := time.Parse(dateLayout, e.ExpiredAt)
			switch {
			case err != nil:
				report(fmt.Sprintf("expired_at %q is not a date (YYYY-MM-DD)", e.ExpiredAt))
			case !expires.After(day):
				report("expired on " + e.ExpiredAt)
			case expires.After(limit):
				report(fmt.Sprintf("expires on %s, more than %d days after %s", e.ExpiredAt, MaxDays, day.Format(dateLayout)))
			}
		}
	}
	return problems, nil
}
```

- [ ] **Step 5: Run the tests to see them pass**

Run: `cd tools/checks && GOWORK=off go test -race ./...`
Expected: both packages `ok`.

- [ ] **Step 6: Write the command** — `tools/checks/cmd/trivyignorecheck/main.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Command trivyignorecheck enforces the exception policy of B7-01 on a
// .trivyignore.yaml file (the first argument, default ".trivyignore.yaml").
// A missing file means there are no exceptions.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/aleogr/identity/tools/checks/internal/trivyignore"
)

func main() {
	name := ".trivyignore.yaml"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}
	content, err := os.ReadFile(name) //nolint:gosec // the path is the operator's own argument
	if errors.Is(err, fs.ErrNotExist) {
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "trivyignorecheck:", err)
		os.Exit(2)
	}
	problems, err := trivyignore.Check(content, time.Now())
	if err != nil {
		fmt.Fprintf(os.Stderr, "trivyignorecheck: %s: %v\n", name, err)
		os.Exit(2)
	}
	for _, p := range problems {
		fmt.Printf("%s: %s\n", name, p)
	}
	if len(problems) > 0 {
		os.Exit(1)
	}
}
```

- [ ] **Step 7: Write `.trivyignore.yaml`**

```yaml
# Exceptions to the Trivy gate (docs/threat-model.md, B7-01).
# Each entry needs `id`, a `statement` justifying it and an `expired_at` date
# at most 90 days ahead; `make check` refuses anything else. An exception is
# added only in a pull request the owner approves.
vulnerabilities: []
```

- [ ] **Step 8: Run it**

Run: `GOWORK=off go build -C tools/checks -o "$PWD/bin/tools/" ./cmd/... && bin/tools/trivyignorecheck; echo "exit $?"`
Expected: `exit 0`.

- [ ] **Step 9: Commit**

```bash
git add tools/checks .trivyignore.yaml
git commit -m "Add trivyignorecheck, which enforces the policy for Trivy exceptions"
```

---

### Task 4: The pinned tools, the lint configuration and the `Makefile`

**Files:**
- Create: `tools/golangci-lint/go.mod` (+ `go.sum`), and the same for `staticcheck`, `gosec`, `govulncheck`, `gitleaks`, `actionlint`, `ko`
- Create: `tools/requirements.in`, `tools/requirements.txt`
- Create: `.golangci.yml`
- Create: `Makefile`

**Interfaces:**
- Consumes: `bin/tools/spdxcheck`, `bin/tools/trivyignorecheck` (Tasks 2 and 3).
- Produces: `make tools`, `make check`, `make test`, `make integration`, `make compile`, `make help`; variables `PKGS`, `MODULES`, `T` (= `bin/tools`) used by later tasks.

- [ ] **Step 1: Create one module per tool**

```bash
pin() { # pin <dir> <package@version>
  mkdir -p "tools/$1"
  (cd "tools/$1" && GOWORK=off go mod init "github.com/aleogr/identity/tools/$1" && GOWORK=off go get -tool "$2")
}
pin golangci-lint github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.14.0
pin staticcheck   honnef.co/go/tools/cmd/staticcheck@v0.8.1
pin gosec         github.com/securego/gosec/v2/cmd/gosec@v2.29.0
pin govulncheck   golang.org/x/vuln/cmd/govulncheck@v1.8.0
pin gitleaks      github.com/zricethezav/gitleaks/v8@v8.30.1
pin actionlint    github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
pin ko            github.com/google/ko@v0.19.1
```

Expected: seven `go.mod` files, each with a `tool` directive. (Why one module per tool: in one module, minimal version selection breaks the builds of `gitleaks` and `actionlint`; spec section 4.3.)

- [ ] **Step 2: Pin `zizmor` with hashes**

`tools/requirements.in`:

```
zizmor==1.30.1
```

Run: `uv pip compile --generate-hashes --universal --python-version 3.11 --custom-compile-command "uv pip compile --generate-hashes --universal --python-version 3.11 tools/requirements.in -o tools/requirements.txt" tools/requirements.in -o tools/requirements.txt`
Expected: `tools/requirements.txt` with `zizmor==1.30.1` and its `--hash=sha256:` lines.

- [ ] **Step 3: Write `.golangci.yml`**

```yaml
# Lint configuration for every Go module (docs/design.md, section 1.2).
version: "2"

run:
  timeout: 10m

linters:
  default: standard
  enable:
    - bodyclose
    - contextcheck
    - depguard
    - errorlint
    - forcetypeassert
    - gocritic
    - gosec
    - nilerr
    - noctx
    - revive
  settings:
    depguard:
      rules:
        # The direction of dependencies (docs/design.md, section 1.2).
        spec:
          list-mode: strict
          files: ["**/spec/**/*.go"]
          allow:
            - $gostd
            - github.com/aleogr/identity/spec
        core:
          list-mode: strict
          files: ["**/core/**/*.go"]
          allow:
            - $gostd
            - golang.org/x/crypto
            - github.com/aleogr/identity/spec
            - github.com/aleogr/identity/core
        conformance:
          list-mode: strict
          files: ["**/conformance/**/*.go"]
          allow:
            - $gostd
            - github.com/aleogr/identity/spec
            - github.com/aleogr/identity/conformance
        # Apache-2.0 code never imports the AGPL-3.0 server (LICENSING.md).
        adapters-licence:
          files: ["**/adapters/**/*.go"]
          deny:
            - pkg: github.com/aleogr/identity/internal
              desc: adapters are Apache-2.0 and never import the AGPL-3.0 server
            - pkg: github.com/aleogr/identity/cmd
              desc: adapters are Apache-2.0 and never import the AGPL-3.0 server
        # An adapter never imports another, so a PostgreSQL user never downloads the GCP libraries.
        adapter-postgres:
          files: ["**/adapters/postgres/**/*.go"]
          deny:
            - pkg: github.com/aleogr/identity/adapters/gcp
              desc: an adapter never imports another adapter
            - pkg: github.com/aleogr/identity/adapters/mail
              desc: an adapter never imports another adapter
        adapter-gcp:
          files: ["**/adapters/gcp/**/*.go"]
          deny:
            - pkg: github.com/aleogr/identity/adapters/postgres
              desc: an adapter never imports another adapter
            - pkg: github.com/aleogr/identity/adapters/mail
              desc: an adapter never imports another adapter
        adapter-mail:
          files: ["**/adapters/mail/**/*.go"]
          deny:
            - pkg: github.com/aleogr/identity/adapters/postgres
              desc: an adapter never imports another adapter
            - pkg: github.com/aleogr/identity/adapters/gcp
              desc: an adapter never imports another adapter
        # Threat X-02: the only source of randomness is crypto/rand.
        randomness:
          files: ["$all", "!$test"]
          deny:
            - pkg: math/rand
              desc: predictable; use crypto/rand (threat X-02)
            - pkg: math/rand/v2
              desc: predictable; use crypto/rand (threat X-02)
  exclusions:
    # golangci-lint's own preset for the unchecked errors Go code conventionally
    # ignores, such as Close on a response body already read.
    presets:
      - std-error-handling
    rules:
      # Tests start the binary they build and git; the arguments are the tests' own.
      - path: _test\.go
        linters: [gosec]
        text: "G204"
```

- [ ] **Step 4: Write the `Makefile`**

```make
# SPDX-License-Identifier: AGPL-3.0-only

# The checks CI runs are the ones to run before pushing (CLAUDE.md).

SHELL := /bin/bash
.SHELLFLAGS := -euo pipefail -c
.DEFAULT_GOAL := help

PYTHON ?= python3
T := bin/tools

# The workspace modules, and the package patterns that cover all of them.
MODULES := . spec core adapters/postgres adapters/gcp adapters/mail conformance
PKGS := ./... ./spec/... ./core/... ./adapters/postgres/... ./adapters/gcp/... ./adapters/mail/... ./conformance/...

# Each tool lives in its own module under tools/ (spec, section 4.3).
TOOLS := \
	golangci-lint=github.com/golangci/golangci-lint/v2/cmd/golangci-lint \
	staticcheck=honnef.co/go/tools/cmd/staticcheck \
	gosec=github.com/securego/gosec/v2/cmd/gosec \
	govulncheck=golang.org/x/vuln/cmd/govulncheck \
	gitleaks=github.com/zricethezav/gitleaks/v8 \
	actionlint=github.com/rhysd/actionlint/cmd/actionlint \
	ko=github.com/google/ko

ZIZMOR := tools/.venv/bin/zizmor
WORKFLOWS := $(wildcard .github/workflows/*.yml)

.PHONY: help
help: ## List the targets
	@grep -hE '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  %-12s %s\n", $$1, $$2}'

.PHONY: tools
tools: ## Build the pinned tools into bin/tools
	@for entry in $(TOOLS); do \
		name=$${entry%%=*}; pkg=$${entry#*=}; \
		GOWORK=off go build -C tools/$$name -o $(CURDIR)/$(T)/$$name $$pkg; \
	done
	@GOWORK=off go build -C tools/checks -o $(CURDIR)/$(T)/ ./cmd/...

$(ZIZMOR): tools/requirements.txt
	$(PYTHON) -m venv tools/.venv
	tools/.venv/bin/python -m pip install --quiet --require-hashes --no-deps -r tools/requirements.txt
	@touch $@

.PHONY: check
check: tools $(ZIZMOR) ## vet, staticcheck, golangci-lint, gosec, govulncheck, licences, exceptions, secrets, workflows
	go vet $(PKGS)
	cd tools/checks && GOWORK=off go vet ./...
	$(T)/staticcheck $(PKGS)
	@mkdir -p dist/sarif
	$(T)/golangci-lint run --config .golangci.yml --output.text.path=stdout --output.sarif.path=dist/sarif/golangci-lint.sarif $(PKGS)
	cd tools/checks && GOWORK=off $(CURDIR)/$(T)/golangci-lint run --config $(CURDIR)/.golangci.yml ./...
	@for m in $(MODULES); do echo "gosec $$m"; (cd $$m && $(CURDIR)/$(T)/gosec -quiet ./...); done
	@for m in $(MODULES); do echo "govulncheck $$m"; (cd $$m && $(CURDIR)/$(T)/govulncheck ./...); done
	$(T)/spdxcheck
	$(T)/trivyignorecheck .trivyignore.yaml
	$(T)/gitleaks git --redact --no-banner .
ifneq ($(WORKFLOWS),)
	$(T)/actionlint
	$(ZIZMOR) --offline .github/workflows
endif

.PHONY: test
test: ## Unit tests with the race detector, across the workspace and tools/checks
	go test -race -cover $(PKGS)
	cd tools/checks && GOWORK=off go test -race -cover ./...

.PHONY: integration
integration: ## Tests behind the integration tag (a real PostgreSQL from F4)
	go test -race -tags integration $(PKGS)

.PHONY: compile
compile: ## Compile every workspace package (CodeQL's build)
	go build $(PKGS)
```

- [ ] **Step 5: Run the checks on the skeleton**

Run: `make check`
Expected: every step passes; the `actionlint` and `zizmor` lines are skipped (no workflows yet). If `govulncheck` fails with a network error on `vuln.go.dev`, the owner has not yet allowed that host (spec section 10, step 0): stop and report it; do not remove the step.

Run: `make test && make integration`
Expected: `ok` or `[no test files]` for every package, exit 0.

- [ ] **Step 6: Commit**

```bash
git add tools .golangci.yml Makefile
git commit -m "Pin the tools, add the lint configuration and the Makefile"
```

---

### Task 5: `core/secret`, the redacting secret type

**Files:**
- Create: `core/secret/secret.go`, `core/secret/secret_test.go`

**Interfaces:**
- Produces: `secret.String` (zero value is empty), `secret.New(v string) secret.String`, `(secret.String).Reveal() string`, `(secret.String).IsZero() bool`, `secret.Redacted = "[REDACTED]"`.

- [ ] **Step 1: Write the failing tests** — `core/secret/secret_test.go`

```go
// SPDX-License-Identifier: Apache-2.0

package secret_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"

	"github.com/aleogr/identity/core/secret"
)

const value = "hunter2-correct-horse"

func assertRedacted(t *testing.T, name, got string) {
	t.Helper()
	if strings.Contains(got, value) {
		t.Errorf("%s leaked the secret: %q", name, got)
	}
}

func TestFormattingRedacts(t *testing.T) {
	s := secret.New(value)
	for _, verb := range []string{"%v", "%+v", "%#v", "%s", "%q", "%x", "%X", "%d", "%10s", "%-10v"} {
		got := fmt.Sprintf(verb, s)
		assertRedacted(t, verb, got)
		if !strings.Contains(got, secret.Redacted) {
			t.Errorf("%s = %q, want it to contain %q", verb, got, secret.Redacted)
		}
	}
	assertRedacted(t, "Sprint", fmt.Sprint(s))
	assertRedacted(t, "Sprintln", fmt.Sprintln(s))
	assertRedacted(t, "String", s.String())
	assertRedacted(t, "GoString", s.GoString())
	assertRedacted(t, "Errorf", fmt.Errorf("login failed for %v", s).Error())
}

func TestUnexportedFieldDoesNotLeak(t *testing.T) {
	type holder struct {
		name string
		key  secret.String
	}
	h := holder{name: "client", key: secret.New(value)}
	for _, verb := range []string{"%v", "%+v", "%#v"} {
		assertRedacted(t, "holder "+verb, fmt.Sprintf(verb, h))
	}
}

func TestExportedFieldRedacts(t *testing.T) {
	type holder struct{ Key secret.String }
	h := holder{Key: secret.New(value)}
	for _, verb := range []string{"%v", "%+v", "%#v"} {
		assertRedacted(t, "Holder "+verb, fmt.Sprintf(verb, h))
	}
	out, err := json.Marshal(h)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != `{"Key":"[REDACTED]"}` {
		t.Errorf("json = %s", got)
	}
}

func TestTextMarshaling(t *testing.T) {
	out, err := secret.New(value).MarshalText()
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != secret.Redacted {
		t.Errorf("MarshalText = %q", out)
	}
	m, err := json.Marshal(map[secret.String]int{secret.New(value): 1})
	if err != nil {
		t.Fatal(err)
	}
	assertRedacted(t, "json map key", string(m))
}

func TestSlogRedacts(t *testing.T) {
	s := secret.New(value)
	for name, h := range map[string]func(*bytes.Buffer) slog.Handler{
		"json": func(b *bytes.Buffer) slog.Handler { return slog.NewJSONHandler(b, nil) },
		"text": func(b *bytes.Buffer) slog.Handler { return slog.NewTextHandler(b, nil) },
	} {
		var buf bytes.Buffer
		slog.New(h(&buf)).Info("m", "k", s, slog.Any("a", s), slog.Group("g", "s", s))
		assertRedacted(t, name, buf.String())
		if !strings.Contains(buf.String(), secret.Redacted) {
			t.Errorf("%s: %q", name, buf.String())
		}
	}
}

func TestRevealAndZero(t *testing.T) {
	if got := secret.New(value).Reveal(); got != value {
		t.Errorf("Reveal = %q", got)
	}
	var zero secret.String
	if zero.Reveal() != "" || !zero.IsZero() {
		t.Error("the zero value must be an empty secret")
	}
	if secret.New("x").IsZero() {
		t.Error("a non-empty secret is not zero")
	}
}
```

- [ ] **Step 2: Run the tests to see them fail**

Run: `go test ./core/...`
Expected: FAIL — `undefined: secret.New`.

- [ ] **Step 3: Implement** — `core/secret/secret.go`

```go
// SPDX-License-Identifier: Apache-2.0

// Package secret holds values that must never reach a log, an error message or
// serialised output by accident (threat B4-04).
package secret

import (
	"fmt"
	"io"
	"log/slog"
)

// Redacted is what every formatting path of a String produces.
const Redacted = "[REDACTED]"

// String is a secret value. The zero value is an empty secret.
//
// The value is held behind a pointer: fmt cannot call methods on an unexported
// struct field, so a struct holding a String there prints the field's own
// fields, which is then an address, never the value.
type String struct {
	value *string
}

var (
	_ fmt.Stringer   = String{}
	_ fmt.GoStringer = String{}
	_ fmt.Formatter  = String{}
	_ slog.LogValuer = String{}
)

// New wraps v.
func New(v string) String { return String{value: &v} }

// Reveal returns the secret value. It is the only way to read it.
func (s String) Reveal() string {
	if s.value == nil {
		return ""
	}
	return *s.value
}

// IsZero reports whether the secret is empty.
func (s String) IsZero() bool { return s.Reveal() == "" }

// String returns Redacted.
func (String) String() string { return Redacted }

// GoString returns Redacted.
func (String) GoString() string { return Redacted }

// Format writes Redacted whatever the verb and flags.
func (String) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, Redacted) }

// LogValue makes slog write Redacted.
func (String) LogValue() slog.Value { return slog.StringValue(Redacted) }

// MarshalJSON encodes Redacted.
func (String) MarshalJSON() ([]byte, error) { return []byte(`"` + Redacted + `"`), nil }

// MarshalText encodes Redacted.
func (String) MarshalText() ([]byte, error) { return []byte(Redacted), nil }
```

- [ ] **Step 4: Run the tests to see them pass**

Run: `go test -race ./core/...`
Expected: `ok  github.com/aleogr/identity/core/secret`

- [ ] **Step 5: Lint and commit**

Run: `make check`
Expected: pass.

```bash
git add core/secret
git commit -m "Add secret.String, a secret type that redacts itself on every output path"
```

---

### Task 6: `internal/buildinfo`

**Files:**
- Create: `internal/buildinfo/buildinfo.go`, `internal/buildinfo/buildinfo_test.go`

**Interfaces:**
- Produces: `buildinfo.ID() string`; the link-time variable `github.com/aleogr/identity/internal/buildinfo.version`.

- [ ] **Step 1: Write the failing tests** — `internal/buildinfo/buildinfo_test.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

package buildinfo

import (
	"runtime/debug"
	"testing"
)

func info(settings ...debug.BuildSetting) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Settings: settings}, true }
}

func TestResolve(t *testing.T) {
	const rev = "0c9bc7fc8d9ec3d8aa5a3e701fc55e4d5be798fa"
	cases := []struct {
		name     string
		injected string
		read     func() (*debug.BuildInfo, bool)
		want     string
	}{
		{"injected wins", "v0.1.0-3-gabcdef012345", info(debug.BuildSetting{Key: "vcs.revision", Value: rev}), "v0.1.0-3-gabcdef012345"},
		{"revision", "", info(debug.BuildSetting{Key: "vcs.revision", Value: rev}), "devel-0c9bc7fc8d9e"},
		{"dirty revision", "", info(
			debug.BuildSetting{Key: "vcs.revision", Value: rev},
			debug.BuildSetting{Key: "vcs.modified", Value: "true"}), "devel-0c9bc7fc8d9e-dirty"},
		{"short revision", "", info(debug.BuildSetting{Key: "vcs.revision", Value: "abc"}), "devel-abc"},
		{"no revision", "", info(), "devel"},
		{"no build info", "", func() (*debug.BuildInfo, bool) { return nil, false }, "devel"},
	}
	for _, c := range cases {
		if got := resolve(c.injected, c.read); got != c.want {
			t.Errorf("%s: resolve() = %q, want %q", c.name, got, c.want)
		}
	}
}

func TestIDIsNeverEmpty(t *testing.T) {
	if ID() == "" {
		t.Fatal("ID() is empty")
	}
}
```

- [ ] **Step 2: Run to see it fail**

Run: `go test ./internal/buildinfo/`
Expected: FAIL — `undefined: resolve`.

- [ ] **Step 3: Implement** — `internal/buildinfo/buildinfo.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Package buildinfo reports which build of the binary is running.
package buildinfo

import "runtime/debug"

// version is set at link time by make build and ko:
//
//	-ldflags "-X github.com/aleogr/identity/internal/buildinfo.version=$(git describe --tags --always --dirty --abbrev=12)"
var version string

// ID returns the build identifier: the value injected at link time or, for a
// build without it such as go run, "devel-" and the VCS revision Go embedded.
func ID() string { return resolve(version, debug.ReadBuildInfo) }

func resolve(injected string, read func() (*debug.BuildInfo, bool)) string {
	if injected != "" {
		return injected
	}
	info, ok := read()
	if !ok || info == nil {
		return "devel"
	}
	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if revision == "" {
		return "devel"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	id := "devel-" + revision
	if modified {
		id += "-dirty"
	}
	return id
}
```

- [ ] **Step 4: Run to see it pass**

Run: `go test -race ./internal/buildinfo/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/buildinfo
git commit -m "Add the build identifier"
```

---

### Task 7: `internal/config`, strict configuration from the environment

**Files:**
- Create: `internal/config/config.go`, `internal/config/config_test.go`, `internal/config/fuzz_test.go`

**Interfaces:**
- Produces: `config.Config{HTTPAddr string; LogLevel slog.Level; LogSource bool; ShutdownTimeout time.Duration}`, `config.Default() Config`, `config.Load(environ []string) (Config, error)`, `config.Error{Problems []Problem}`, `config.Problem{Variable, Value, Reason string}` with `String()` rendering `NAME="value": reason`, `config.Prefix = "IDENTITY_"`.

- [ ] **Step 1: Write the failing tests** — `internal/config/config_test.go`

```go
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
```

- [ ] **Step 2: Write the fuzz test** — `internal/config/fuzz_test.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

package config_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/aleogr/identity/internal/config"
)

// FuzzLoad checks that no input panics, that every refusal is a *config.Error,
// and that every accepted configuration respects its bounds (threat X-03).
func FuzzLoad(f *testing.F) {
	for _, seed := range [][2]string{
		{"IDENTITY_LOG_SOURCE", "ture"},
		{"IDENTITY_HTTP_ADDR", "[::1]:8080"},
		{"IDENTITY_SHUTDOWN_TIMEOUT", "59.999s"},
		{"PORT", "65535"},
		{"IDENTITY_", ""},
	} {
		f.Add(seed[0], seed[1])
	}
	f.Fuzz(func(t *testing.T, name, value string) {
		cfg, err := config.Load([]string{name + "=" + value})
		if err != nil {
			var cerr *config.Error
			if !errors.As(err, &cerr) || len(cerr.Problems) == 0 {
				t.Fatalf("Load(%q=%q) error %v is not a *config.Error with problems", name, value, err)
			}
			if strings.ContainsAny(err.Error(), "\n\r") {
				t.Fatalf("error contains a line break: %q", err.Error())
			}
			return
		}
		if cfg.ShutdownTimeout < time.Second || cfg.ShutdownTimeout > time.Minute {
			t.Fatalf("accepted shutdown timeout %v", cfg.ShutdownTimeout)
		}
		if cfg.HTTPAddr == "" {
			t.Fatal("accepted an empty address")
		}
	})
}
```

- [ ] **Step 3: Run to see them fail**

Run: `go test ./internal/config/`
Expected: FAIL — `undefined: config.Load`.

- [ ] **Step 4: Implement** — `internal/config/config.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Package config loads the deployment configuration from IDENTITY_*
// environment variables. It refuses anything it does not understand, naming
// the variable and quoting the value (docs/design.md, section 7; threat X-05).
package config

import (
	"fmt"
	"log/slog"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Prefix marks the variables this package owns. An unknown variable with this
// prefix is refused, so a typo cannot silently leave a default in place.
const Prefix = "IDENTITY_"

const (
	varHTTPAddr        = "IDENTITY_HTTP_ADDR"
	varLogLevel        = "IDENTITY_LOG_LEVEL"
	varLogSource       = "IDENTITY_LOG_SOURCE"
	varShutdownTimeout = "IDENTITY_SHUTDOWN_TIMEOUT"
	// varPort is set by Cloud Run; it is the only variable read without the prefix.
	varPort = "PORT"
)

var known = map[string]bool{varHTTPAddr: true, varLogLevel: true, varLogSource: true, varShutdownTimeout: true}

var levels = map[string]slog.Level{
	"debug": slog.LevelDebug,
	"info":  slog.LevelInfo,
	"warn":  slog.LevelWarn,
	"error": slog.LevelError,
}

const (
	minShutdownTimeout = time.Second
	maxShutdownTimeout = time.Minute
	// maxQuoted is the most bytes of a value an error message repeats.
	maxQuoted = 64
)

// Config is the validated deployment configuration.
type Config struct {
	HTTPAddr        string
	LogLevel        slog.Level
	LogSource       bool
	ShutdownTimeout time.Duration
}

// Default returns the configuration used when no variable is set.
func Default() Config {
	return Config{HTTPAddr: ":8080", LogLevel: slog.LevelInfo, ShutdownTimeout: 8 * time.Second}
}

// Problem is one refused variable.
type Problem struct {
	Variable string
	Value    string
	Reason   string
}

// String renders the problem as NAME="value": reason, with the value quoted
// (control characters escaped) and truncated. A name that is not plain
// printable text (possible for an unknown variable) is quoted the same way.
func (p Problem) String() string { return name(p.Variable) + "=" + quote(p.Value) + ": " + p.Reason }

// name returns v unchanged when quoting would only add the quotes.
func name(v string) string {
	if q := strconv.Quote(v); len(v) <= maxQuoted && q[1:len(q)-1] == v {
		return v
	}
	return quote(v)
}

// Error lists every refused variable.
type Error struct {
	Problems []Problem
}

func (e *Error) Error() string {
	parts := make([]string, len(e.Problems))
	for i, p := range e.Problems {
		parts[i] = p.String()
	}
	return "invalid configuration: " + strings.Join(parts, "; ")
}

// Load reads the configuration from environ, "KEY=value" entries as
// os.Environ returns them. It validates every variable before returning and
// reports every problem at once.
func Load(environ []string) (Config, error) {
	env := map[string]string{}
	for _, entry := range environ {
		k, v, ok := strings.Cut(entry, "=")
		if ok && (strings.HasPrefix(k, Prefix) || k == varPort) {
			env[k] = v
		}
	}
	cfg := Default()
	var problems []Problem
	refuse := func(name, reason string) {
		problems = append(problems, Problem{Variable: name, Value: env[name], Reason: reason})
	}

	var unknown []string
	for k := range env {
		if strings.HasPrefix(k, Prefix) && !known[k] {
			unknown = append(unknown, k)
		}
	}
	sort.Strings(unknown)
	for _, k := range unknown {
		refuse(k, "unknown variable")
	}

	addr, hasAddr := env[varHTTPAddr]
	port, hasPort := env[varPort]
	switch {
	case hasAddr && hasPort:
		refuse(varHTTPAddr, "conflicts with PORT; set only one of them")
	case hasAddr:
		if reason := checkAddr(addr); reason != "" {
			refuse(varHTTPAddr, reason)
		} else {
			cfg.HTTPAddr = addr
		}
	case hasPort:
		if n, err := strconv.ParseUint(port, 10, 16); err != nil || n == 0 {
			refuse(varPort, "must be a number from 1 to 65535")
		} else {
			cfg.HTTPAddr = ":" + port
		}
	}

	if v, ok := env[varLogLevel]; ok {
		if level, ok := levels[v]; ok {
			cfg.LogLevel = level
		} else {
			refuse(varLogLevel, "must be one of debug, info, warn, error")
		}
	}

	if v, ok := env[varLogSource]; ok {
		switch v {
		case "true":
			cfg.LogSource = true
		case "false":
			cfg.LogSource = false
		default:
			refuse(varLogSource, "must be true or false")
		}
	}

	if v, ok := env[varShutdownTimeout]; ok {
		d, err := time.ParseDuration(v)
		switch {
		case err != nil:
			refuse(varShutdownTimeout, "must be a duration such as 8s")
		case d < minShutdownTimeout || d > maxShutdownTimeout:
			refuse(varShutdownTimeout, fmt.Sprintf("must be between %s and %s", minShutdownTimeout, maxShutdownTimeout))
		default:
			cfg.ShutdownTimeout = d
		}
	}

	if len(problems) > 0 {
		return Config{}, &Error{Problems: problems}
	}
	return cfg, nil
}

// checkAddr returns why addr is not an acceptable listening address, or "".
func checkAddr(addr string) string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "must be host:port, such as :8080"
	}
	if host != "" && host != "localhost" && net.ParseIP(host) == nil {
		return "the host must be empty, localhost or an IP address"
	}
	if _, err := strconv.ParseUint(port, 10, 16); err != nil {
		return "the port must be a number from 0 to 65535"
	}
	return ""
}

// quote renders v for an error message: quoted with control characters
// escaped, and cut at maxQuoted bytes on a rune boundary.
func quote(v string) string {
	if len(v) <= maxQuoted {
		return strconv.Quote(v)
	}
	cut := maxQuoted
	for cut > 0 && !utf8.RuneStart(v[cut]) {
		cut--
	}
	return strconv.Quote(v[:cut]) + "..."
}
```

- [ ] **Step 5: Run the tests to see them pass**

Run: `go test -race ./internal/config/`
Expected: `ok`.

- [ ] **Step 6: Run the fuzzer briefly**

Run: `go test ./internal/config/ -run '^$' -fuzz FuzzLoad -fuzztime 30s`
Expected: `PASS` with no failing input. If it finds one, fix the code, keep the input the fuzzer wrote under `internal/config/testdata/fuzz/FuzzLoad/` as a regression case, and rerun.

- [ ] **Step 7: Commit**

```bash
git add internal/config
git commit -m "Add strict configuration loading that refuses to start on unknown or invalid values"
```

---

### Task 8: `internal/logging`, JSON logs in Cloud Logging's fields

**Files:**
- Create: `internal/logging/logging.go`, `internal/logging/logging_test.go`

**Interfaces:**
- Consumes: `secret.String` (tests only).
- Produces: `logging.New(w io.Writer, level slog.Leveler, addSource bool) *slog.Logger`; constants `logging.KeySeverity = "severity"`, `KeyMessage = "message"`, `KeyTime = "time"`, `KeySourceLocation = "logging.googleapis.com/sourceLocation"`.

- [ ] **Step 1: Write the failing tests** — `internal/logging/logging_test.go`

```go
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
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/logging/`
Expected: FAIL — `undefined: logging.New`.

- [ ] **Step 3: Implement** — `internal/logging/logging.go`

```go
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
```

- [ ] **Step 4: Add the `require` the test needs and run**

The root module's `go.mod` already requires `core` (Task 1). Run: `go test -race ./internal/logging/`
Expected: `ok`.

- [ ] **Step 5: Commit**

```bash
git add internal/logging
git commit -m "Add JSON logging with Cloud Logging's field names"
```

---

### Task 9: `internal/server`, `/health` and graceful shutdown

**Files:**
- Create: `internal/server/server.go`, `internal/server/server_test.go`

**Interfaces:**
- Produces: `server.NewHandler() http.Handler`, `server.New(handler http.Handler, logger *slog.Logger) *http.Server`, `server.Run(ctx context.Context, srv *http.Server, ln net.Listener, shutdownTimeout time.Duration) error`, `server.ErrShutdownTimeout`.

- [ ] **Step 1: Write the failing tests** — `internal/server/server_test.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

package server_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/aleogr/identity/internal/server"
)

func TestHealth(t *testing.T) {
	h := server.NewHandler()
	for _, target := range []string{"/health", "/health?probe=1"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		res := rec.Result()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status = %d", target, res.StatusCode)
		}
		for k, v := range map[string]string{
			"Content-Type":           "application/json",
			"Cache-Control":          "no-store",
			"X-Content-Type-Options": "nosniff",
		} {
			if got := res.Header.Get(k); got != v {
				t.Errorf("%s: %s = %q, want %q", target, k, got, v)
			}
		}
		if body := rec.Body.String(); body != "{\"status\":\"ok\"}\n" {
			t.Errorf("%s: body = %q", target, body)
		}
	}
}

func TestHealthHead(t *testing.T) {
	srv := httptest.NewServer(server.NewHandler())
	defer srv.Close()
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodHead, srv.URL+"/health", nil)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || len(body) != 0 || res.Header.Get("Content-Type") != "application/json" {
		t.Errorf("HEAD: status %d, body %q, headers %v", res.StatusCode, body, res.Header)
	}
}

func TestOtherMethodsAndPaths(t *testing.T) {
	h := server.NewHandler()
	for _, m := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(m, "/health", nil))
		if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != "GET, HEAD" {
			t.Errorf("%s: status %d, Allow %q", m, rec.Code, rec.Header().Get("Allow"))
		}
	}
	for _, p := range []string{"/", "/health/", "/healthz", "/version"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, p, nil))
		if rec.Code != http.StatusNotFound {
			t.Errorf("%s: status %d", p, rec.Code)
		}
	}
}

func TestNewSetsLimits(t *testing.T) {
	s := server.New(server.NewHandler(), slog.New(slog.DiscardHandler))
	if s.ReadHeaderTimeout != 10*time.Second || s.ReadTimeout != 30*time.Second ||
		s.WriteTimeout != 30*time.Second || s.IdleTimeout != 120*time.Second ||
		s.MaxHeaderBytes != 64<<10 || s.ErrorLog == nil {
		t.Errorf("limits = %+v", s)
	}
}

func listen(t *testing.T) net.Listener {
	t.Helper()
	ln, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return ln
}

func get(ctx context.Context, url string) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()
	_, _ = io.Copy(io.Discard, res.Body)
	return res.StatusCode, nil
}

func TestRunServesUntilCancelled(t *testing.T) {
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, server.New(server.NewHandler(), slog.New(slog.DiscardHandler)), ln, time.Second) }()
	if code, err := get(t.Context(), "http://"+ln.Addr().String()+"/health"); err != nil || code != http.StatusOK {
		t.Fatalf("health: %d, %v", code, err)
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run = %v", err)
	}
}

func TestRunLetsRequestsInFlightFinish(t *testing.T) {
	started := make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /slow", func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, server.New(mux, slog.New(slog.DiscardHandler)), ln, 5*time.Second) }()
	result := make(chan int, 1)
	go func() {
		code, _ := get(context.Background(), "http://"+ln.Addr().String()+"/slow")
		result <- code
	}()
	<-started
	cancel()
	if code := <-result; code != http.StatusOK {
		t.Errorf("request in flight got %d", code)
	}
	if err := <-done; err != nil {
		t.Fatalf("Run = %v", err)
	}
}

func TestRunReportsShutdownTimeout(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	mux := http.NewServeMux()
	mux.HandleFunc("GET /stuck", func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
	})
	ln := listen(t)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, server.New(mux, slog.New(slog.DiscardHandler)), ln, 50*time.Millisecond) }()
	go func() { _, _ = get(context.Background(), "http://"+ln.Addr().String()+"/stuck") }()
	<-started
	cancel()
	err := <-done
	close(release)
	if !errors.Is(err, server.ErrShutdownTimeout) {
		t.Fatalf("Run = %v, want ErrShutdownTimeout", err)
	}
}

func TestRunReturnsServeError(t *testing.T) {
	ln := listen(t)
	_ = ln.Close()
	err := server.Run(t.Context(), server.New(server.NewHandler(), slog.New(slog.DiscardHandler)), ln, time.Second)
	if err == nil {
		t.Fatal("Run on a closed listener returned nil")
	}
}
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/server/`
Expected: FAIL — `undefined: server.NewHandler`.

- [ ] **Step 3: Implement** — `internal/server/server.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Package server serves the platform's HTTP endpoints and shuts down
// gracefully.
package server

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"time"
)

// Limits of every connection (docs/threat-model.md, X-04).
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 30 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = 120 * time.Second
	maxHeaderBytes    = 64 << 10
)

// ErrShutdownTimeout reports that requests in flight did not finish within the
// shutdown timeout; their connections were closed.
var ErrShutdownTimeout = errors.New("server: requests in flight did not finish before the shutdown timeout")

// NewHandler returns the handler for every route the server answers.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	// A GET pattern also serves HEAD; other methods get 405 with Allow: GET, HEAD.
	mux.HandleFunc("GET /health", health)
	return mux
}

// health reports that the process is up. It carries no build identifier:
// that is the operator-only version endpoint's job (F3).
func health(w http.ResponseWriter, _ *http.Request) {
	h := w.Header()
	h.Set("Content-Type", "application/json")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, "{\"status\":\"ok\"}\n")
}

// New returns a server for handler with the platform's limits, logging its
// own errors through logger.
func New(handler http.Handler, logger *slog.Logger) *http.Server {
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}
}

// Run serves on ln until ctx is done, then stops accepting connections and
// waits up to shutdownTimeout for requests in flight. It returns nil after a
// clean shutdown, ErrShutdownTimeout when requests were cut off, and the
// serving error when serving stopped by itself.
func Run(ctx context.Context, srv *http.Server, ln net.Listener, shutdownTimeout time.Duration) error {
	served := make(chan error, 1)
	go func() { served <- srv.Serve(ln) }()

	select {
	case err := <-served:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		_ = srv.Close()
		if errors.Is(err, context.DeadlineExceeded) {
			return ErrShutdownTimeout
		}
		return fmt.Errorf("server: shutdown: %w", err)
	}
	if err := <-served; !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run to see them pass**

Run: `go test -race -count=3 ./internal/server/`
Expected: `ok` (three runs, to catch timing flakiness).

- [ ] **Step 5: Commit**

```bash
git add internal/server
git commit -m "Add the HTTP server with /health and graceful shutdown"
```

---

### Task 10: `internal/app`, `cmd/identity` and the process tests

**Files:**
- Create: `internal/app/app.go`, `internal/app/app_test.go`
- Create: `cmd/identity/main.go`, `cmd/identity/main_test.go`
- Modify: `Makefile` (add `build`, `run`)

**Interfaces:**
- Consumes: `config.Load`, `config.Error`, `logging.New`, `server.New`, `server.NewHandler`, `server.Run`, `buildinfo.ID`.
- Produces: `app.Run(ctx context.Context, args, environ []string, stdout, stderr io.Writer) int`; exit statuses `app.ExitOK = 0`, `app.ExitFailure = 1`, `app.ExitUsage = 2`; the start-up log line `{"message":"serving","build_id":…,"go_version":…,"addr":…}` and the final line `{"message":"stopped"}`; `make build` producing `bin/identity`.

- [ ] **Step 1: Write the failing unit tests** — `internal/app/app_test.go`

```go
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
```

- [ ] **Step 2: Run to see them fail**

Run: `go test ./internal/app/`
Expected: FAIL — `undefined: app.Run`.

- [ ] **Step 3: Implement** — `internal/app/app.go`

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Package app is the command line of the identity binary.
package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"runtime"

	"github.com/aleogr/identity/internal/buildinfo"
	"github.com/aleogr/identity/internal/config"
	"github.com/aleogr/identity/internal/logging"
	"github.com/aleogr/identity/internal/server"
)

// Exit statuses.
const (
	ExitOK      = 0
	ExitFailure = 1 // a run-time failure
	ExitUsage   = 2 // a usage or configuration error
)

const usage = `Usage: identity <command>

Commands:
  serve     Start the HTTP server.
  version   Print the build identifier.
  help      Print this help.

The server reads its configuration from IDENTITY_* environment variables and
refuses to start on an unknown variable or an invalid value.
`

// Run executes the command in args (without the program name) and returns
// the process exit status.
func Run(ctx context.Context, args, environ []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprint(stderr, usage)
		return ExitUsage
	}
	switch cmd := args[0]; cmd {
	case "serve", "version":
		if len(args) > 1 {
			_, _ = fmt.Fprintf(stderr, "identity %s: unexpected argument %q\n\n%s", cmd, args[1], usage)
			return ExitUsage
		}
		if cmd == "version" {
			_, _ = fmt.Fprintln(stdout, buildinfo.ID())
			return ExitOK
		}
		return serve(ctx, environ, stdout, stderr)
	case "help", "-h", "-help", "--help":
		_, _ = fmt.Fprint(stdout, usage)
		return ExitOK
	default:
		_, _ = fmt.Fprintf(stderr, "identity: unknown command %q\n\n%s", cmd, usage)
		return ExitUsage
	}
}

func serve(ctx context.Context, environ []string, stdout, stderr io.Writer) int {
	cfg, err := config.Load(environ)
	if err != nil {
		reportConfig(logging.New(stderr, slog.LevelInfo, false), err)
		return ExitUsage
	}
	logger := logging.New(stdout, cfg.LogLevel, cfg.LogSource)

	ln, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.HTTPAddr)
	if err != nil {
		logger.Error("cannot listen", "addr", cfg.HTTPAddr, "error", err.Error())
		return ExitFailure
	}
	logger.Info("serving",
		"build_id", buildinfo.ID(),
		"go_version", runtime.Version(),
		"addr", ln.Addr().String())

	if err := server.Run(ctx, server.New(server.NewHandler(), logger), ln, cfg.ShutdownTimeout); err != nil {
		logger.Error("server stopped with an error", "error", err.Error())
		return ExitFailure
	}
	logger.Info("stopped")
	return ExitOK
}

func reportConfig(logger *slog.Logger, err error) {
	var cerr *config.Error
	if !errors.As(err, &cerr) {
		logger.Error("invalid configuration; refusing to start", "error", err.Error())
		return
	}
	problems := make([]string, len(cerr.Problems))
	for i, p := range cerr.Problems {
		problems[i] = p.String()
	}
	logger.Error("invalid configuration; refusing to start", "problems", problems)
}
```

- [ ] **Step 4: Run to see them pass**

Run: `go test -race ./internal/app/`
Expected: `ok`.

- [ ] **Step 5: Write `cmd/identity/main.go`**

```go
// SPDX-License-Identifier: AGPL-3.0-only

// Command identity is the identity platform's server and command line.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/aleogr/identity/internal/app"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt)
	code := app.Run(ctx, os.Args[1:], os.Environ(), os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
```

- [ ] **Step 6: Write the failing process tests** — `cmd/identity/main_test.go`

```go
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
```

- [ ] **Step 7: Run all tests**

Run: `go test -race ./...`
Expected: every package `ok`, including `github.com/aleogr/identity/cmd/identity`.

- [ ] **Step 8: Add `build` and `run` to the `Makefile`**

Insert after the `compile` target:

```make
BUILD_ID := $(shell git describe --tags --always --dirty --abbrev=12 2>/dev/null || echo unknown)
LDFLAGS := -X github.com/aleogr/identity/internal/buildinfo.version=$(BUILD_ID)

.PHONY: build
build: ## Build bin/identity with the build identifier
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/identity ./cmd/identity

.PHONY: run
run: build ## Build and run identity serve
	bin/identity serve
```

Run: `make build && bin/identity version && git describe --tags --always --dirty --abbrev=12`
Expected: the two lines are identical.

- [ ] **Step 9: Check and commit**

Run: `make check && make test`
Expected: pass.

```bash
git add internal/app cmd Makefile
git commit -m "Add identity serve and identity version, with process tests"
```

---

### Task 11: The end-to-end suite

**Files:**
- Create: `e2e/requirements.in`, `e2e/requirements.txt`, `e2e/pytest.ini`, `e2e/conftest.py`, `e2e/test_smoke.py`
- Modify: `Makefile` (add `e2e-deps`, `e2e`)

**Interfaces:**
- Consumes: `make build` → `bin/identity`; the `serving` and `stopped` log lines (Task 10).
- Produces: `make e2e-deps`, `make e2e`; screenshots in `e2e/artifacts/`.

- [ ] **Step 1: Pin the dependencies**

`e2e/requirements.in`:

```
# playwright is pinned to the release built against the Chromium revision in
# /opt/pw-browsers (1194 on 2026-09-26); it moves only with that revision.
playwright==1.56.0
pytest==9.1.1
```

Run: `uv pip compile --generate-hashes --universal --python-version 3.11 --custom-compile-command "uv pip compile --generate-hashes --universal --python-version 3.11 e2e/requirements.in -o e2e/requirements.txt" e2e/requirements.in -o e2e/requirements.txt`
Expected: `e2e/requirements.txt` listing `playwright==1.56.0`, `pytest==9.1.1` and their dependencies, each with hashes.

`e2e/pytest.ini`:

```ini
[pytest]
addopts = -ra --strict-markers
testpaths = .
```

- [ ] **Step 2: Add the targets to the `Makefile`**

```make
E2E_VENV := e2e/.venv

.PHONY: e2e-deps
e2e-deps: ## Create the end-to-end virtual environment (Playwright 1.56.0)
	$(PYTHON) -m venv $(E2E_VENV)
	$(E2E_VENV)/bin/python -m pip install --quiet --require-hashes --no-deps -r e2e/requirements.txt

.PHONY: e2e
e2e: build ## Run the end-to-end suite against bin/identity
	cd e2e && .venv/bin/python -m pytest
```

Run: `make e2e-deps`
Expected: exit 0.

- [ ] **Step 3: Write the fixtures** — `e2e/conftest.py`

```python
# SPDX-License-Identifier: AGPL-3.0-only
"""Fixtures that start the identity binary and a Chromium page."""

import json
import os
import pathlib
import queue
import subprocess
import threading
import time
import urllib.request
from dataclasses import dataclass, field

import pytest
from playwright.sync_api import sync_playwright

ROOT = pathlib.Path(__file__).resolve().parent.parent
BINARY = pathlib.Path(os.environ.get("E2E_BINARY", ROOT / "bin" / "identity"))
ARTIFACTS = ROOT / "e2e" / "artifacts"
START_TIMEOUT = 10.0


@dataclass
class Server:
    process: subprocess.Popen
    startup: dict
    base_url: str
    ready_seconds: float
    lines: "queue.Queue[str]" = field(repr=False)

    def stop(self) -> tuple[int, list[dict]]:
        """Send SIGTERM and return the exit status and every later log line."""
        self.process.terminate()
        code = self.process.wait(timeout=START_TIMEOUT)
        later = []
        while not self.lines.empty():
            line = self.lines.get()
            if line:
                later.append(json.loads(line))
        return code, later


def _environment() -> dict:
    # Only what the binary needs: an inherited IDENTITY_* variable would make it refuse to start.
    return {"PATH": os.environ.get("PATH", ""), "IDENTITY_HTTP_ADDR": "127.0.0.1:0"}


def start_server() -> Server:
    started = time.monotonic()
    process = subprocess.Popen(
        [str(BINARY), "serve"],
        env=_environment(),
        stdout=subprocess.PIPE,
        stderr=subprocess.STDOUT,
        text=True,
    )
    lines: "queue.Queue[str]" = queue.Queue()

    def pump() -> None:
        for line in process.stdout:
            lines.put(line.strip())

    threading.Thread(target=pump, daemon=True).start()
    first = lines.get(timeout=START_TIMEOUT)
    startup = json.loads(first)
    assert startup["message"] == "serving", first
    base_url = "http://" + startup["addr"]
    while True:
        try:
            with urllib.request.urlopen(base_url + "/health", timeout=1) as response:
                if response.status == 200:
                    break
        except OSError:
            if time.monotonic() - started > START_TIMEOUT:
                raise
            time.sleep(0.01)
    return Server(process, startup, base_url, time.monotonic() - started, lines)


@pytest.fixture
def server():
    s = start_server()
    yield s
    if s.process.poll() is None:
        s.process.kill()
        s.process.wait()


@pytest.fixture(scope="session")
def browser():
    with sync_playwright() as p:
        b = p.chromium.launch()
        yield b
        b.close()


@pytest.fixture
def page(browser):
    ARTIFACTS.mkdir(parents=True, exist_ok=True)
    context = browser.new_context()
    p = context.new_page()
    yield p
    context.close()
```

- [ ] **Step 4: Write the tests** — `e2e/test_smoke.py`

```python
# SPDX-License-Identifier: AGPL-3.0-only
"""Smoke tests of the identity binary: it starts, answers and stops."""

import json
import subprocess

from conftest import ARTIFACTS, ROOT


def test_chromium_reads_health(server, page):
    response = page.goto(server.base_url + "/health")
    assert response is not None
    assert response.status == 200
    assert response.headers["content-type"] == "application/json"
    assert response.headers["cache-control"] == "no-store"
    assert response.headers["x-content-type-options"] == "nosniff"
    assert json.loads(response.text()) == {"status": "ok"}
    page.screenshot(path=ARTIFACTS / "health.png")


def test_cold_start_under_one_second(server):
    # Requirement §18: the binary serves requests within 1 second of starting.
    assert server.ready_seconds < 1.0, f"ready after {server.ready_seconds:.3f}s"


def test_startup_log_carries_the_build_identifier(server):
    expected = subprocess.run(
        ["git", "describe", "--tags", "--always", "--dirty", "--abbrev=12"],
        cwd=ROOT, capture_output=True, text=True, check=True,
    ).stdout.strip()
    assert server.startup["build_id"] == expected
    assert server.startup["severity"] == "INFO"
    assert server.startup["go_version"].startswith("go")


def test_sigterm_stops_gracefully(server):
    code, later = server.stop()
    assert code == 0
    assert later and later[-1]["message"] == "stopped"
```

- [ ] **Step 5: Run the suite**

Run: `make e2e`
Expected: `4 passed`; `e2e/artifacts/health.png` exists.

- [ ] **Step 6: Commit**

```bash
git add e2e Makefile
git commit -m "Add the end-to-end suite with Playwright"
```

---

### Task 12: The image, built with `ko`

**Files:**
- Create: `.ko.yaml`
- Modify: `Makefile` (add `image`)

**Interfaces:**
- Consumes: `bin/tools/ko`, `BUILD_ID`.
- Produces: `make image` → `dist/identity-image.tar` (a Docker-format tarball Trivy reads with `--input`).

- [ ] **Step 1: Write `.ko.yaml`**

```yaml
# Images built by ko (spec, decision 7). The base image is pinned by digest;
# ko's builds are reproducible (fixed timestamps, -trimpath), which the
# release's digest check relies on (docs/requirements.md, section 19).
defaultBaseImage: gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
defaultPlatforms:
  - linux/amd64
builds:
  - id: identity
    main: ./cmd/identity
    env:
      - CGO_ENABLED=0
    flags:
      - -trimpath
    ldflags:
      - -X github.com/aleogr/identity/internal/buildinfo.version={{.Env.BUILD_ID}}
```

- [ ] **Step 2: Add the target to the `Makefile`**

```make
.PHONY: image
image: tools ## Build the container image into dist/identity-image.tar (no push)
	@mkdir -p dist
	BUILD_ID=$(BUILD_ID) KO_DOCKER_REPO=identity.local/identity $(T)/ko build --bare --push=false \
		--tarball=dist/identity-image.tar ./cmd/identity
```

- [ ] **Step 3: Build twice and compare the digests**

Run: `make image 2>&1 | grep sha256 && make image 2>&1 | grep sha256`
Expected: the same `identity.local/identity@sha256:…` digest twice (reproducible build).

- [ ] **Step 4: Commit**

```bash
git add .ko.yaml Makefile
git commit -m "Build the container image with ko"
```

---

### Task 13: The CI workflows and Dependabot

**Files:**
- Create: `.github/workflows/ci.yml`, `.github/workflows/codeql.yml`
- Modify: `.github/dependabot.yml`

**Interfaces:**
- Consumes: `make check`, `make test`, `make integration`, `make e2e-deps`, `make e2e`, `make image`, `make compile`.
- Produces: the jobs `check`, `test`, `integration`, `e2e`, `image`, `codeql` (the required checks).

Pinned actions (resolved on 2026-09-26 with `git ls-remote --tags`, dereferencing annotated tags):

| Action | Version | Commit |
|---|---|---|
| `actions/checkout` | v7.0.1 | `3d3c42e5aac5ba805825da76410c181273ba90b1` |
| `actions/setup-go` | v7.0.0 | `b7ad1dad31e06c5925ef5d2fc7ad053ef454303e` |
| `actions/setup-python` | v7.0.0 | `5fda3b95a4ea91299a34e894583c3862153e4b97` |
| `actions/upload-artifact` | v7.0.1 | `043fb46d1a93c77aae656e7c1c64a875d1fc6a0a` |
| `github/codeql-action` | v4.38.2 | `2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2` |
| `aquasecurity/setup-trivy` | v0.3.1 | `81e514348e19b6112ce2a7e3ecbafe19c1e1f567` |

Service image: `postgres:16` (16.15) at `sha256:1a6ab3f5345eb6dbe04a1349529caabdb0ab09293a09590fad07b2246bfa4b54`. Trivy: `v0.74.0`.

- [ ] **Step 1: Write `.github/workflows/ci.yml`**

```yaml
# The checks of CLAUDE.md. Each job name is a required status check of the
# protect-main ruleset: check, test, integration, e2e, image.
name: ci

on:
  pull_request:
  push:
    branches: [main]

permissions: {}

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: ${{ github.event_name == 'pull_request' }}

jobs:
  check:
    name: check
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          fetch-depth: 0 # gitleaks scans the whole history
          persist-credentials: false
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
          cache-dependency-path: "**/go.sum"
      - uses: actions/setup-python@5fda3b95a4ea91299a34e894583c3862153e4b97 # v7.0.0
        with:
          python-version: "3.11"
      - run: make check
      - name: Upload golangci-lint findings to code scanning
        if: always() && hashFiles('dist/sarif/golangci-lint.sarif') != '' && (github.event_name == 'push' || github.event.pull_request.head.repo.full_name == github.repository)
        uses: github/codeql-action/upload-sarif@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2 # v4.38.2
        with:
          sarif_file: dist/sarif/golangci-lint.sarif
          category: golangci-lint

  test:
    name: test
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
          cache-dependency-path: "**/go.sum"
      - run: make test

  integration:
    name: integration
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    permissions:
      contents: read
    services:
      postgres:
        image: postgres:16@sha256:1a6ab3f5345eb6dbe04a1349529caabdb0ab09293a09590fad07b2246bfa4b54
        env:
          POSTGRES_USER: identity
          POSTGRES_DB: identity
          POSTGRES_HOST_AUTH_METHOD: trust # a throwaway container reachable only from this job
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U identity"
          --health-interval 2s
          --health-timeout 5s
          --health-retries 30
    env:
      TEST_DATABASE_URL: postgres://identity@localhost:5432/identity?sslmode=disable
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
          cache-dependency-path: "**/go.sum"
      - run: make integration

  e2e:
    name: e2e
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    permissions:
      contents: read
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
          cache-dependency-path: "**/go.sum"
      - uses: actions/setup-python@5fda3b95a4ea91299a34e894583c3862153e4b97 # v7.0.0
        with:
          python-version: "3.11"
      - run: make e2e-deps
      - run: e2e/.venv/bin/python -m playwright install --with-deps chromium
      - run: make e2e
      - name: Keep the screenshots
        if: always()
        uses: actions/upload-artifact@043fb46d1a93c77aae656e7c1c64a875d1fc6a0a # v7.0.1
        with:
          name: e2e-artifacts
          path: e2e/artifacts/
          if-no-files-found: ignore

  image:
    name: image
    runs-on: ubuntu-24.04
    timeout-minutes: 20
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
          cache-dependency-path: "**/go.sum"
      - run: make image
      - uses: aquasecurity/setup-trivy@81e514348e19b6112ce2a7e3ecbafe19c1e1f567 # v0.3.1
        with:
          version: v0.74.0
          cache: true
      - name: Report every finding to code scanning
        run: >-
          trivy image --input dist/identity-image.tar --scanners vuln
          --ignorefile .trivyignore.yaml --format sarif --output dist/trivy.sarif
      - name: Upload Trivy findings to code scanning
        if: always() && hashFiles('dist/trivy.sarif') != '' && (github.event_name == 'push' || github.event.pull_request.head.repo.full_name == github.repository)
        uses: github/codeql-action/upload-sarif@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2 # v4.38.2
        with:
          sarif_file: dist/trivy.sarif
          category: trivy
      - name: Fail on HIGH and CRITICAL, fixed or not (threat B7-01)
        run: >-
          trivy image --input dist/identity-image.tar --scanners vuln
          --ignorefile .trivyignore.yaml --severity HIGH,CRITICAL --exit-code 1
```

- [ ] **Step 2: Write `.github/workflows/codeql.yml`**

```yaml
# GitHub code scanning (advanced setup). The job name is a required status
# check of the protect-main ruleset: codeql.
name: codeql

on:
  pull_request:
  push:
    branches: [main]
  schedule:
    - cron: "23 4 * * 1"

permissions: {}

jobs:
  codeql:
    name: codeql
    runs-on: ubuntu-24.04
    timeout-minutes: 30
    permissions:
      contents: read
      security-events: write
    steps:
      - uses: actions/checkout@3d3c42e5aac5ba805825da76410c181273ba90b1 # v7.0.1
        with:
          persist-credentials: false
      - uses: actions/setup-go@b7ad1dad31e06c5925ef5d2fc7ad053ef454303e # v7.0.0
        with:
          go-version-file: go.mod
          cache-dependency-path: "**/go.sum"
      - uses: github/codeql-action/init@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2 # v4.38.2
        with:
          languages: go
          build-mode: manual
      - run: make compile
      - uses: github/codeql-action/analyze@2892aa5e19bbd11bc0cff5427e3b750a04d9e3c2 # v4.38.2
        with:
          category: "/language:go"
```

- [ ] **Step 3: Replace `.github/dependabot.yml`**

```yaml
version: 2
updates:
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
    groups:
      actions:
        patterns: ["*"]

  # The workspace modules and every tool module. The toolchain line and the
  # image digests in .ko.yaml and ci.yml are not updated by Dependabot: they
  # are bumped by hand when govulncheck or Trivy reports a fix.
  - package-ecosystem: gomod
    directories:
      - /
      - /spec
      - /core
      - /adapters/postgres
      - /adapters/gcp
      - /adapters/mail
      - /conformance
      - /tools/*
    schedule:
      interval: weekly
    groups:
      go:
        patterns: ["*"]

  - package-ecosystem: pip
    directories:
      - /e2e
      - /tools
    schedule:
      interval: weekly
    ignore:
      # Bound to the Chromium revision pre-installed in Claude Code sessions
      # (/opt/pw-browsers); it moves only by a deliberate pull request when
      # that revision changes (docs/claude-code-environment.md).
      - dependency-name: playwright
    groups:
      python:
        patterns: ["*"]
```

- [ ] **Step 4: Lint the workflows**

Run: `make check`
Expected: pass, now including the `actionlint` and `zizmor` lines. Fix any finding in the workflow rather than suppressing it; if `zizmor` reports a finding that is a false positive, stop and ask the owner before adding any `# zizmor: ignore` comment.

- [ ] **Step 5: Commit**

```bash
git add .github
git commit -m "Add the CI and CodeQL workflows and extend Dependabot"
```

---

### Task 14: The documents

**Files:**
- Modify: `docs/roadmap.md`, `docs/design.md`, `docs/threat-model.md`, `CLAUDE.md`, `docs/claude-code-environment.md`, `CONTRIBUTING.md`

- [ ] **Step 1: `docs/roadmap.md`**

- In F1's **Scope**, replace the lint-tools bullet with: "Lint and security tools pinned as `tool` directives, one module per tool under `tools/` outside the workspace; `.golangci.yml` with `depguard` rules enforcing design §1.2's dependency direction and a rule against `math/rand` in all non-test code." Replace the workflow bullet with: "`.github/workflows/ci.yml` with the jobs `check`, `test`, `integration` (a PostgreSQL 16 service container, without tests until F4), `e2e` and `image`, and `.github/workflows/codeql.yml` with the job `codeql`; actions pinned by commit SHA and least-privilege permissions (B7-02); `gitleaks`, `actionlint` and `zizmor` in `make check`; the image built by `ko` and scanned by Trivy, blocking on HIGH and CRITICAL; SARIF uploaded to code scanning." Add the bullet: "`.trivyignore.yaml` and `trivyignorecheck`: exceptions only with a statement and an expiry at most 90 days ahead."
- In F1's **What is needed from the owner**, replace item 1 with "Add the required status checks `check`, `test`, `integration`, `e2e`, `image` and `codeql` to the `protect-main` ruleset." Replace item 2 with "Allow `vuln.go.dev` in the environment's network settings, so `govulncheck` runs in a session."
- In F8's **Scope**, append "; the lint rule that enforces constant-time comparison of secrets (§17)". In F8's **Threats**, add `X-01`.
- In the appendix, change the Go row to "`go1.24.7` on the path, `GOTOOLCHAIN=auto`; latest release `go1.27.1`" with consequence "`go.work` and `go.mod` name `toolchain go1.27.1`"; add a row "Network | `go.dev`, `vuln.go.dev` and GitHub release downloads refused; `proxy.golang.org`, `pypi.org`, `gcr.io`, `ghcr.io` and Docker Hub reachable | Tools come through the Go proxy or PyPI; Trivy runs only in CI; `vuln.go.dev` must be allowed for `govulncheck`"; change the "Python Playwright" row's consequence to "`make e2e-deps` creates `e2e/.venv` with it".
- Mark F1's checkbox `[x]` only in the last commit before merge? **No** — leave it unchecked; the owner marks deliveries done.

- [ ] **Step 2: `docs/design.md`, section 1.2**

In the module tree, after the `e2e/` line add:

```
tools/<tool>/           one module per pinned tool, outside go.work             AGPL-3.0
tools/checks/           the project's own checkers (spdxcheck, trivyignorecheck) AGPL-3.0
```

and after the tree add the bullet: "**Why tools sit outside the workspace, one module per tool.** In the workspace they would raise dependency versions `core` shares with them; in a single module, minimal version selection breaks the builds of some tools (found in F1)."

- [ ] **Step 3: `docs/threat-model.md`**

- B4-04, Verification: append "; `secret.String` redacts on every formatting path, with unit tests (F1)".
- B7-01, Mitigation: append "; Trivy blocks on HIGH and CRITICAL whether or not a fix exists; an exception needs a statement and an expiry at most 90 days ahead, checked by `trivyignorecheck`, and the owner's approval". Verification: "CI: `govulncheck` and Trivy jobs; `trivyignorecheck` in `make check`".
- B7-02, Verification: replace "Workflow lint in CI" with "`actionlint` and `zizmor` in `make check`".
- B7-04, Verification: replace "CI" with "`gitleaks` over the whole history in `make check`".
- X-05, Verification: append "; F1: a process test that a boolean with a typo makes the binary exit with status 2 naming the variable".

- [ ] **Step 4: `CLAUDE.md`, section Commands**

Replace the block and the paragraph after it with:

```
make check        # go vet, staticcheck, golangci-lint (with depguard), gosec, govulncheck, licence
                  # identifiers, Trivy exceptions, gitleaks over the history, actionlint and zizmor
make test         # go test ./... -race -cover, across the workspace and tools/checks
make integration  # tests that need a real PostgreSQL, behind the `integration` tag
make e2e          # builds the binary and runs the end-to-end suite against it
make image        # builds the container image with ko into dist/ (Trivy scans it in CI only)
make tf           # terraform fmt -check, init -backend=false and validate (from F2)
```

The tools are pinned as `tool` directives, one module per tool under `tools/`, outside the workspace;
`make tools` builds them into `bin/tools/`. `make integration` and `make e2e` start a PostgreSQL 16
cluster of their own when `TEST_DATABASE_URL` is unset (from F4), because the Docker daemon does not run
in a session. The end-to-end suite runs from its own virtual environment, `e2e/.venv`, created by
`make e2e-deps`, never with the `pytest` on the path. `govulncheck` needs `vuln.go.dev`, allowed in the
environment's network settings. The OpenID Foundation's conformance suite, Trivy and CodeQL run only in
CI, with Docker. Terraform is only ever applied by CI.

- [ ] **Step 5: `docs/claude-code-environment.md`**

In "Why the end-to-end and Terraform dependencies are installed by the script", replace the last
sentence with: "`make e2e-deps` (F1) creates `e2e/.venv` with the pinned Playwright release, the one
built against the Chromium revision pre-installed in `/opt/pw-browsers` (1194 on 2026-09-26, Playwright
1.56.0), so no browser is downloaded; `make terraform-deps` arrives with F2." Add a section "## Network"
stating that `vuln.go.dev` must be in the environment's allowed domains for `govulncheck`, and that
`go.dev` and GitHub release downloads are not needed by any target.

- [ ] **Step 6: `CONTRIBUTING.md`**

In "How changes are made", change the checks line to: "The checks CI runs must pass: `make check`, `make test`, `make integration`, `make e2e`, and the image scan and CodeQL in CI. No test is skipped or disabled to make a pull request pass."

- [ ] **Step 7: Commit**

```bash
git add docs CLAUDE.md CONTRIBUTING.md
git commit -m "Record F1's decisions in the roadmap, design, threat model and working notes"
```

---

### Task 15: Verification, security review, pull request and CI

**Files:** none new (one temporary commit, reverted).

- [ ] **Step 1: Run every check from a clean tree**

Run: `git status --porcelain && make check && make test && make integration && make e2e && make image`
Expected: `git status` prints nothing; every target exits 0. Save the full output to the scratchpad for the pull request.

- [ ] **Step 2: Run the security review**

Invoke the `identity-security-review` skill over the branch diff (configuration parser, secret type, workflows). Fix every finding it raises, with a test, before continuing.

- [ ] **Step 3: Push and open the pull request**

```bash
git push -u origin claude/beautiful-cannon-vzf0sf
```

Open the pull request with the GitHub tools, following `.github/pull_request_template.md`: the Summary names F1 and links the spec and this plan; Verification pastes the outputs of Step 1 and says that Trivy and CodeQL run only in CI and that `integration` has no tests until F4; the checklist is filled in honestly. End the body with the pull request attribution lines.

- [ ] **Step 4: Show `depguard` refusing a `core` → adapter import**

Add a temporary commit:

```bash
cat > core/violation.go <<'EOF'
// SPDX-License-Identifier: Apache-2.0

package core

import _ "github.com/aleogr/identity/adapters/postgres" // deliberate: shows depguard refusing it
EOF
git add core/violation.go
git commit -m "Demonstrate that depguard refuses an import from core into an adapter (to be reverted)"
git push
```

Wait for the `check` job, copy its `depguard` line (`import 'github.com/aleogr/identity/adapters/postgres' is not allowed from list 'core'`) and the job's link into the pull request, then:

```bash
git revert --no-edit HEAD
git push
```

- [ ] **Step 5: Drive the pull request to green**

Subscribe to the pull request's activity. On every failure, read that job's log, reproduce locally when the session can, fix, rerun the matching `make` target, and push. Never skip, disable or weaken a check. When all six checks are green and the branch is mergeable, report to the owner and give the manual steps of the spec, section 10, one at a time.
