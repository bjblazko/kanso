// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"html"
	"net/url"
	"regexp"
	"strings"
)

var (
	metaTag  = regexp.MustCompile(`(?is)<meta\b[^>]*>`)
	metaAttr = regexp.MustCompile(`(?is)\b(property|name|content)\s*=\s*("[^"]*"|'[^']*')`)
)

// imageURL reads schema.org's image: an address, an ImageObject or a list of them.
func imageURL(v any, base string) string {
	switch x := v.(type) {
	case string:
		return resolve(base, x)
	case map[string]any:
		if u, _ := x["url"].(string); u != "" {
			return resolve(base, u)
		}
		u, _ := x["contentUrl"].(string)
		return resolve(base, u)
	case []any:
		for _, e := range x {
			if u := imageURL(e, base); u != "" {
				return u
			}
		}
	}
	return ""
}

// PageImage is the photo a page names for sharing (og:image, else twitter:image).
func PageImage(page, base string) string {
	found := map[string]string{}
	for _, tag := range metaTag.FindAllString(page, -1) {
		attrs := map[string]string{}
		for _, m := range metaAttr.FindAllStringSubmatch(tag, -1) {
			attrs[strings.ToLower(m[1])] = html.UnescapeString(strings.Trim(m[2], `"'`))
		}
		key := strings.ToLower(attrs["property"] + attrs["name"])
		if _, seen := found[key]; !seen && attrs["content"] != "" {
			found[key] = attrs["content"]
		}
	}
	if u := resolve(base, found["og:image"]); u != "" {
		return u
	}
	return resolve(base, found["twitter:image"])
}

// resolve makes ref absolute against base; only http and https count.
func resolve(base, ref string) string {
	b, err1 := url.Parse(base)
	r, err2 := url.Parse(strings.TrimSpace(ref))
	if err1 != nil || err2 != nil || ref == "" {
		return ""
	}
	u := b.ResolveReference(r)
	if u.Scheme != "http" && u.Scheme != "https" {
		return ""
	}
	u.User = nil
	return u.String()
}
