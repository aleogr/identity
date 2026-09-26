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
