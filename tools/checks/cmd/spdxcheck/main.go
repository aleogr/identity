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
