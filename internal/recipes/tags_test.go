// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"slices"
	"testing"
)

func TestNormalizeTags(t *testing.T) {
	got := NormalizeTags(" Deftig, süß ,Süß,, Kinderessen")
	if !slices.Equal(got, []string{"deftig", "süß", "kinderessen"}) {
		t.Errorf("%q", got)
	}
	if got := NormalizeTags(" , "); got != nil {
		t.Errorf("empty: %q", got)
	}
}
