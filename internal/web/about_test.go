// SPDX-License-Identifier: Apache-2.0

package web

import (
	"strings"
	"testing"

	"github.com/bjblazko/kanso/internal/buildinfo"
)

func TestAboutPage(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t, "anna", "Anna", false)
	for _, path := range []string{"/about", "/about/licenses"} {
		_, body := get(t, c, ts.URL+path)
		main := between(body, "<main", "</main>")
		for _, want := range []string{"Über Kanso", buildinfo.Version, "© 2026 Timo Böwing", "Apache-2.0",
			`href="https://github.com/bjblazko/kanso"`, `href="https://huepattl.de"`, "Lizenzen und Dank", "Was Kanso wohin schickt"} {
			if !strings.Contains(main, want) {
				t.Errorf("%s lacks %q", path, want)
			}
		}
	}
}

func TestFooterShowsVersion(t *testing.T) {
	ts := newTestServer(t)
	c := ts.client(t, "anna", "Anna", false)
	_, body := get(t, c, ts.URL+"/notes")
	footer := between(body, `<footer class="footer">`, `</footer>`)
	if !strings.Contains(footer, "Kanso "+buildinfo.Version) || !strings.Contains(footer, `href="/about"`) {
		t.Fatalf("footer: %s", footer)
	}
}
