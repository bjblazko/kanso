// SPDX-License-Identifier: Apache-2.0

package recipes

import (
	"slices"
	"strings"
)

// NormalizeTags reads comma-separated tags: trimmed, lower case, without
// empty ones or doubles, in the order first named.
func NormalizeTags(raw string) []string {
	var tags []string
	for _, t := range strings.Split(raw, ",") {
		t = strings.ToLower(strings.Join(strings.Fields(t), " "))
		if t != "" && !slices.Contains(tags, t) {
			tags = append(tags, t)
		}
	}
	return tags
}
