// SPDX-License-Identifier: Apache-2.0

// Package buildinfo names the Kanso version. The number lives in VERSION;
// a build may set it instead with
// -ldflags "-X github.com/bjblazko/kanso/internal/buildinfo.Version=1.2.3".
package buildinfo

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var file string

// Version is the release, e.g. "0.1.0".
var Version string

func init() {
	if Version == "" {
		Version = strings.TrimSpace(file)
	}
}
