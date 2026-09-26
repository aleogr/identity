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
