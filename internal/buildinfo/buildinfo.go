// SPDX-License-Identifier: AGPL-3.0-only

// Package buildinfo reports which build of the binary is running.
package buildinfo

import "runtime/debug"

// version is set at link time by make build and ko:
//
//	-ldflags "-X github.com/aleogr/identity/internal/buildinfo.version=$(git describe --tags --always --dirty --abbrev=12)"
var version string

// ID returns the build identifier: the value injected at link time or, for a
// build without it such as go run, "devel-" and the VCS revision Go embedded.
func ID() string { return resolve(version, debug.ReadBuildInfo) }

func resolve(injected string, read func() (*debug.BuildInfo, bool)) string {
	if injected != "" {
		return injected
	}
	info, ok := read()
	if !ok || info == nil {
		return "devel"
	}
	var revision string
	var modified bool
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			revision = s.Value
		case "vcs.modified":
			modified = s.Value == "true"
		}
	}
	if revision == "" {
		return "devel"
	}
	if len(revision) > 12 {
		revision = revision[:12]
	}
	id := "devel-" + revision
	if modified {
		id += "-dirty"
	}
	return id
}
