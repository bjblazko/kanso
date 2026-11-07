// SPDX-License-Identifier: Apache-2.0

// Package recipeimport turns a recipe web page into a recipe draft: from the
// page's own recipe data (JSON-LD) or, without it, with the AI. Pages are
// fetched safely: never from addresses in the own network (SSRF).
package recipeimport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Errors a person can understand.
var (
	ErrBlocked = errors.New("this address cannot be fetched")
	ErrNotHTML = errors.New("the address is no web page")
	ErrFetch   = errors.New("the page could not be fetched")
)

const (
	fetchTimeout = 10 * time.Second
	maxPage      = 3 << 20
	maxRedirects = 5
	userAgent    = "Kanso (+https://github.com/bjblazko/kanso)"
)

// Page is a fetched web page.
type Page struct {
	URL  string // after redirects
	HTML string
}

// Fetcher fetches recipe pages.
type Fetcher struct {
	client *http.Client
	ports  []int // allowed ports; nil = any (tests)
}

// NewFetcher fetches only public addresses on ports 80 and 443.
func NewFetcher() *Fetcher {
	return newFetcher(func(a netip.AddrPort) bool { return isPublic(a.Addr()) }, []int{80, 443})
}

// NewTestFetcher allows every address and port; for tests against local servers only.
func NewTestFetcher() *Fetcher { return newFetcher(func(netip.AddrPort) bool { return true }, nil) }

func newFetcher(allow func(netip.AddrPort) bool, ports []int) *Fetcher {
	dialer := &net.Dialer{Timeout: 5 * time.Second, Control: func(_, address string, _ syscall.RawConn) error {
		// Checked on the resolved address of every connection, redirects included.
		if ap, err := netip.ParseAddrPort(address); err != nil || !allow(ap) {
			return ErrBlocked
		}
		return nil
	}}
	transport := &http.Transport{Proxy: nil, DialContext: dialer.DialContext, TLSHandshakeTimeout: 5 * time.Second,
		ResponseHeaderTimeout: fetchTimeout, MaxResponseHeaderBytes: 64 << 10}
	f := &Fetcher{ports: ports}
	f.client = &http.Client{Transport: transport, Timeout: fetchTimeout, CheckRedirect: func(req *http.Request, via []*http.Request) error {
		if len(via) >= maxRedirects {
			return ErrFetch
		}
		return f.check(req.URL)
	}}
	return f
}

// special are ranges that are no ordinary internet hosts, or that can hide
// an address of an own network (NAT64, 6to4, IPv4-compatible).
var special = func() []netip.Prefix {
	var out []netip.Prefix
	for _, p := range []string{"0.0.0.0/8", "100.64.0.0/10", "192.0.0.0/24", "198.18.0.0/15", "240.0.0.0/4",
		"64:ff9b::/96", "64:ff9b:1::/48", "2002::/16", "::/96", "fec0::/10", "2001:db8::/32"} {
		out = append(out, netip.MustParsePrefix(p))
	}
	return out
}()

// isPublic tells whether a is an address on the internet, not in an own network.
func isPublic(a netip.Addr) bool {
	a = a.Unmap()
	if !a.IsGlobalUnicast() || a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast() {
		return false
	}
	return !slices.ContainsFunc(special, func(p netip.Prefix) bool { return p.Contains(a) })
}

// check allows http and https on the allowed ports.
func (f *Fetcher) check(u *url.URL) error {
	if (u.Scheme != "http" && u.Scheme != "https") || u.Hostname() == "" {
		return ErrBlocked
	}
	port := map[string]int{"http": 80, "https": 443}[u.Scheme]
	if p := u.Port(); p != "" {
		port, _ = strconv.Atoi(p)
	}
	if f.ports != nil && !slices.Contains(f.ports, port) {
		return ErrBlocked
	}
	return nil
}

// Fetch loads a web page of at most 3 MB.
func (f *Fetcher) Fetch(ctx context.Context, raw string) (Page, error) {
	u, err := url.Parse(raw)
	if err != nil || f.check(u) != nil {
		return Page{}, ErrBlocked
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return Page{}, ErrBlocked
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := f.client.Do(req)
	if err != nil {
		if errors.Is(err, ErrBlocked) {
			return Page{}, ErrBlocked
		}
		return Page{}, fmt.Errorf("%w: %v", ErrFetch, err)
	}
	defer func() { _ = resp.Body.Close() }()
	return read(resp)
}

func read(resp *http.Response) (Page, error) {
	if resp.StatusCode >= 400 {
		return Page{}, fmt.Errorf("%w: HTTP %d", ErrFetch, resp.StatusCode)
	}
	mt, params, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	if mt != "text/html" && mt != "application/xhtml+xml" {
		return Page{}, ErrNotHTML
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxPage+1))
	if err != nil || len(b) > maxPage {
		return Page{}, ErrFetch
	}
	u := *resp.Request.URL
	u.User = nil // no credentials in the recipe's source
	return Page{URL: u.String(), HTML: decode(b, params["charset"])}, nil
}

// decode reads Latin-1 pages (ISO-8859-1, windows-1252) as such; everything else as UTF-8.
func decode(b []byte, charset string) string {
	switch strings.ToLower(charset) {
	case "iso-8859-1", "iso-8859-15", "latin1", "windows-1252", "cp1252":
		r := make([]rune, len(b))
		for i, c := range b {
			r[i] = rune(c)
		}
		return string(r)
	}
	return string(b)
}
