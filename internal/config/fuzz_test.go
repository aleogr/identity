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
