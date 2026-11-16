// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestSidenavRemembersFolded(t *testing.T) {
	ts, admin, _ := setupHaus(t)
	_, body := get(t, admin, ts.URL+"/s/haus")
	if !strings.Contains(body, `<details class="sidenav-fold" open>`) {
		t.Error("spaces bar not open by default")
	}
	u, _ := url.Parse(ts.URL)
	admin.Jar.SetCookies(u, []*http.Cookie{{Name: sidenavCookie, Value: "closed", Path: "/"}})
	for _, path := range []string{"/", "/s/haus"} {
		if _, body = get(t, admin, ts.URL+path); !strings.Contains(body, `<details class="sidenav-fold">`) {
			t.Errorf("%s: folded spaces bar renders open", path)
		}
	}
}
