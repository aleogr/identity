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
		"go.mod":                {Data: []byte("module x\n")},
		"LICENSE":               {Data: []byte("AGPL")},
		"cmd/identity/main.go":  {Data: []byte("// SPDX-License-Identifier: AGPL-3.0-only\n\npackage main\n")},
		"core/go.mod":           {Data: []byte("module x/core\n")},
		"core/doc.go":           {Data: []byte("// SPDX-License-Identifier: AGPL-3.0-only\n\npackage core\n")},
		"core/secret/secret.go": {Data: []byte("package secret\n")},
		"spec/go.mod":           {Data: []byte("module x/spec\n")},
		"spec/LICENSE":          {Data: []byte("Apache")},
		"spec/doc.go":           {Data: []byte("// SPDX-License-Identifier: Apache-2.0\n\npackage spec\n")},
		"tools/ko/go.mod":       {Data: []byte("module x/tools/ko\n")},
		"README.md":             {Data: []byte("# x\n")},
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
