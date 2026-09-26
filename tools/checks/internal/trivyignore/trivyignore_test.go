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
