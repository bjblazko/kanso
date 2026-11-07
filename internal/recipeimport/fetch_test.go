// SPDX-License-Identifier: Apache-2.0

package recipeimport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

func TestIsPublic(t *testing.T) {
	for addr, want := range map[string]bool{
		"8.8.8.8": true, "2a00:1450::1": true,
		"127.0.0.1": false, "10.1.2.3": false, "192.168.178.148": false, "172.16.0.1": false,
		"169.254.1.1": false, "100.64.0.1": false, "0.0.0.0": false, "::1": false, "fd00::1": false,
		"fe80::1": false, "::ffff:127.0.0.1": false, "224.0.0.1": false,
	} {
		if got := isPublic(netip.MustParseAddr(addr)); got != want {
			t.Errorf("%s: %v", addr, got)
		}
	}
}

func page(body, contentType string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", contentType)
		_, _ = w.Write([]byte(body))
	}))
}

func TestFetchBlocksInternal(t *testing.T) {
	srv := page("<html></html>", "text/html")
	defer srv.Close()
	if _, err := NewFetcher().Fetch(context.Background(), srv.URL); !errors.Is(err, ErrBlocked) {
		t.Fatalf("loopback: %v", err)
	}
	for _, u := range []string{"ftp://example.org/x", "file:///etc/passwd", "javascript:alert(1)", "http://example.org:8443/x"} {
		if _, err := NewFetcher().Fetch(context.Background(), u); !errors.Is(err, ErrBlocked) {
			t.Errorf("%s: %v", u, err)
		}
	}
}

func TestFetchRedirectToInternalBlocked(t *testing.T) {
	internal := page("secret", "text/html")
	defer internal.Close()
	public := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, internal.URL, http.StatusFound)
	}))
	defer public.Close()
	publicPort := netip.MustParseAddrPort(strings.TrimPrefix(public.URL, "http://")).Port()
	f := newFetcher(func(a netip.AddrPort) bool { return a.Port() == publicPort }, []int{80, 443, int(publicPort)})
	if _, err := f.Fetch(context.Background(), public.URL); !errors.Is(err, ErrBlocked) {
		t.Fatalf("redirect to internal: %v", err)
	}
}

func TestFetchHTMLOnly(t *testing.T) {
	ok := page(`<html><body>Hallo</body></html>`, "text/html; charset=utf-8")
	defer ok.Close()
	p, err := NewTestFetcher().Fetch(context.Background(), ok.URL)
	if err != nil || !strings.Contains(p.HTML, "Hallo") || p.URL != ok.URL {
		t.Fatalf("%+v %v", p, err)
	}
	pdf := page("%PDF", "application/pdf")
	defer pdf.Close()
	if _, err := NewTestFetcher().Fetch(context.Background(), pdf.URL); !errors.Is(err, ErrNotHTML) {
		t.Fatalf("pdf: %v", err)
	}
}

// The address check sits in the dialer: it holds for any port and every redirect.
func TestDialGuardBlocksLoopback(t *testing.T) {
	srv := page("<html></html>", "text/html")
	defer srv.Close()
	f := newFetcher(func(a netip.AddrPort) bool { return isPublic(a.Addr()) }, nil)
	if _, err := f.Fetch(context.Background(), srv.URL); !errors.Is(err, ErrBlocked) {
		t.Fatalf("dial guard: %v", err)
	}
}

func TestIsPublicSpecialRanges(t *testing.T) {
	for _, addr := range []string{"64:ff9b::a00:1", "2002:c0a8:101::1", "::7f00:1", "fec0::1", "0.1.2.3", "192.0.0.8", "198.18.0.1", "240.0.0.1", "255.255.255.255"} {
		if isPublic(netip.MustParseAddr(addr)) {
			t.Errorf("%s counted as public", addr)
		}
	}
}

func TestFetchLatin1AndSourceWithoutCredentials(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=ISO-8859-1")
		_, _ = w.Write([]byte("<p>M\xf6hren</p>"))
	}))
	defer srv.Close()
	u := strings.Replace(srv.URL, "http://", "http://user:pass@", 1)
	p, err := NewTestFetcher().Fetch(context.Background(), u)
	if err != nil || !strings.Contains(p.HTML, "Möhren") || strings.Contains(p.URL, "pass") {
		t.Fatalf("%+v %v", p, err)
	}
}

func TestFetchImage(t *testing.T) {
	jpeg := []byte("\xff\xd8\xff\xe0 fake jpeg")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/a.jpg":
			w.Header().Set("Content-Type", "image/jpeg")
			_, _ = w.Write(jpeg)
		case "/svg":
			w.Header().Set("Content-Type", "image/svg+xml")
			_, _ = w.Write([]byte("<svg/>"))
		}
	}))
	defer srv.Close()
	img, err := NewTestFetcher().FetchImage(context.Background(), srv.URL+"/a.jpg")
	if err != nil || img.Ext != ".jpg" || string(img.Data) != string(jpeg) || img.Type != "image/jpeg" {
		t.Fatalf("%+v %v", img, err)
	}
	if _, err := NewTestFetcher().FetchImage(context.Background(), srv.URL+"/svg"); !errors.Is(err, ErrNotImage) {
		t.Fatalf("svg: %v", err)
	}
	if _, err := NewFetcher().FetchImage(context.Background(), srv.URL+"/a.jpg"); !errors.Is(err, ErrBlocked) {
		t.Fatalf("internal: %v", err)
	}
}
