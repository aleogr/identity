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
