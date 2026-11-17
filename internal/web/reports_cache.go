// SPDX-License-Identifier: Apache-2.0

package web

import (
	"sync"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

// reportCache keeps the last history scan until the repository moves on
// (ADR-015). Requests that arrive while a scan runs wait for it.
type reportCache struct {
	mu   sync.Mutex
	head string
	scan store.Scan
	at   time.Time // zero: nothing cached
	wait chan struct{}
}

// get returns the scan for head, scanning only when the cached one is older.
// A failed scan is not kept, so the next request tries again.
func (c *reportCache) get(head string, scan func() (store.Scan, error)) (store.Scan, time.Time, error) {
	c.mu.Lock()
	for c.wait != nil {
		w := c.wait
		c.mu.Unlock()
		<-w
		c.mu.Lock()
	}
	if c.head == head && !c.at.IsZero() {
		defer c.mu.Unlock()
		return c.scan, c.at, nil
	}
	w := make(chan struct{})
	c.wait = w
	c.mu.Unlock()

	sc, err := scan()
	now := time.Now()
	c.mu.Lock()
	c.head, c.scan, c.at, c.wait = head, sc, now, nil
	if err != nil {
		c.at = time.Time{}
	}
	close(w)
	c.mu.Unlock()
	return sc, now, err
}
