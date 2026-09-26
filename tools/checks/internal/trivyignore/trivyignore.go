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
