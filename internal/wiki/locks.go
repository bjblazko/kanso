// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Lock says who edits a page; it lives in memory only (ADR-003).
type Lock struct {
	PageID  string
	Sub     string
	Name    string
	Token   string
	Expires time.Time
}

// ErrLocked means someone else is editing the page.
type ErrLocked struct{ Holder Lock }

func (e *ErrLocked) Error() string { return fmt.Sprintf("page is being edited by %s", e.Holder.Name) }

// ErrLockLost means the caller's lock expired or was broken.
var ErrLockLost = errors.New("edit lock lost")

// Locks hands out one edit lock per page.
type Locks struct {
	ttl   time.Duration
	now   func() time.Time
	mu    sync.Mutex
	locks map[string]Lock
}

// NewLocks creates the lock table; a lock lives ttl after its last heartbeat.
func NewLocks(ttl time.Duration, now func() time.Time) *Locks {
	return &Locks{ttl: ttl, now: now, locks: map[string]Lock{}}
}

// live returns the unexpired lock of pageID; the caller holds l.mu.
func (l *Locks) live(pageID string) (Lock, bool) {
	lk, ok := l.locks[pageID]
	if ok && l.now().After(lk.Expires) {
		delete(l.locks, pageID)
		return Lock{}, false
	}
	return lk, ok
}

// Acquire gives sub the lock, unless someone else holds it. The same
// person gets a fresh token, which invalidates an older tab.
func (l *Locks) Acquire(pageID, sub, name string) (Lock, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lk, ok := l.live(pageID); ok && lk.Sub != sub {
		return Lock{}, &ErrLocked{Holder: lk}
	}
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	lk := Lock{PageID: pageID, Sub: sub, Name: name, Token: hex.EncodeToString(b), Expires: l.now().Add(l.ttl)}
	l.locks[pageID] = lk
	return lk, nil
}

// Heartbeat extends a lock the caller still holds.
func (l *Locks) Heartbeat(pageID, token string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	lk, ok := l.live(pageID)
	if !ok || !sameToken(lk.Token, token) {
		return ErrLockLost
	}
	lk.Expires = l.now().Add(l.ttl)
	l.locks[pageID] = lk
	return nil
}

// Check reports whether token still holds the lock.
func (l *Locks) Check(pageID, token string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lk, ok := l.live(pageID); ok && sameToken(lk.Token, token) {
		return nil
	}
	return ErrLockLost
}

// Holder returns the current lock of pageID, if any.
func (l *Locks) Holder(pageID string) (Lock, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.live(pageID)
}

// Release gives the lock up, if token holds it.
func (l *Locks) Release(pageID, token string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if lk, ok := l.locks[pageID]; ok && sameToken(lk.Token, token) {
		delete(l.locks, pageID)
	}
}

// Break removes the lock regardless of who holds it.
func (l *Locks) Break(pageID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.locks, pageID)
}

func sameToken(a, b string) bool {
	return a != "" && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
