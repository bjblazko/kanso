// SPDX-License-Identifier: Apache-2.0

package buildinfo

import (
	"regexp"
	"testing"
)

func TestVersionIsSemVer(t *testing.T) {
	if !regexp.MustCompile(`^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`).MatchString(Version) {
		t.Fatalf("version %q", Version)
	}
}
