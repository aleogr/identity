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
