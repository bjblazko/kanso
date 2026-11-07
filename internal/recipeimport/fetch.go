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

// isPublic tells whether a is an address on the internet, not in an own network.
func isPublic(a netip.Addr) bool {
	a = a.Unmap()
	cgnat := netip.MustParsePrefix("100.64.0.0/10")
	return a.IsGlobalUnicast() && !a.IsPrivate() && !a.IsLoopback() && !a.IsLinkLocalUnicast() && !cgnat.Contains(a)
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
	if mt, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); mt != "text/html" && mt != "application/xhtml+xml" {
		return Page{}, ErrNotHTML
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, maxPage+1))
	if err != nil || len(b) > maxPage {
		return Page{}, ErrFetch
	}
	return Page{URL: resp.Request.URL.String(), HTML: string(b)}, nil
}
