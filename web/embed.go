// SPDX-License-Identifier: Apache-2.0

// Package web holds the embedded templates and static files.
package web

import "embed"

// Files contains templates/ and static/.
//
//go:embed templates static
var Files embed.FS
