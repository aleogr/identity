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
