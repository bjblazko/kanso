// SPDX-License-Identifier: Apache-2.0

package wiki

import (
	"errors"
	"testing"
	"time"
)

type clock struct{ t time.Time }

func (c *clock) now() time.Time      { return c.t }
func (c *clock) add(d time.Duration) { c.t = c.t.Add(d) }
func newLocks(c *clock) *Locks       { return NewLocks(5*time.Minute, c.now) }
func holder(err error) (Lock, bool) {
	var e *ErrLocked
	ok := errors.As(err, &e)
	if ok {
		return e.Holder, true
	}
	return Lock{}, false
}

func TestSecondUserCannotAcquire(t *testing.T) {
	c := &clock{t: time.Now()}
	l := newLocks(c)
	if _, err := l.Acquire("p1", "anna", "Anna"); err != nil {
		t.Fatal(err)
	}
	_, err := l.Acquire("p1", "ben", "Ben")
	if h, ok := holder(err); !ok || h.Name != "Anna" {
		t.Errorf("err = %v", err)
	}
}

func TestLockExpiresWithoutHeartbeat(t *testing.T) {
	c := &clock{t: time.Now()}
	l := newLocks(c)
	l.Acquire("p1", "anna", "Anna")
	c.add(5*time.Minute + time.Second)
	if _, err := l.Acquire("p1", "ben", "Ben"); err != nil {
		t.Errorf("expired lock still held: %v", err)
	}
}

func TestHeartbeatExtends(t *testing.T) {
	c := &clock{t: time.Now()}
	l := newLocks(c)
	lk, _ := l.Acquire("p1", "anna", "Anna")
	c.add(4 * time.Minute)
	if err := l.Heartbeat("p1", lk.Token); err != nil {
		t.Fatal(err)
	}
	c.add(4 * time.Minute)
	if err := l.Check("p1", lk.Token); err != nil {
		t.Errorf("lock lost despite heartbeat: %v", err)
	}
}

func TestSameUserReacquires(t *testing.T) {
	c := &clock{t: time.Now()}
	l := newLocks(c)
	first, _ := l.Acquire("p1", "anna", "Anna")
	second, err := l.Acquire("p1", "anna", "Anna")
	if err != nil || second.Token == "" {
		t.Fatal(err)
	}
	if l.Check("p1", first.Token) == nil && first.Token != second.Token {
		t.Error("old tab's token still valid after reacquire")
	}
}

func TestBreakAndRelease(t *testing.T) {
	c := &clock{t: time.Now()}
	l := newLocks(c)
	lk, _ := l.Acquire("p1", "anna", "Anna")
	l.Break("p1")
	if err := l.Check("p1", lk.Token); err == nil {
		t.Error("broken lock still valid")
	}
	lk, _ = l.Acquire("p1", "anna", "Anna")
	l.Release("p1", "wrong")
	if l.Check("p1", lk.Token) != nil {
		t.Error("release with wrong token removed the lock")
	}
	l.Release("p1", lk.Token)
	if _, err := l.Acquire("p1", "ben", "Ben"); err != nil {
		t.Error("released lock still held")
	}
}

func TestHolderReportsCurrentLock(t *testing.T) {
	c := &clock{t: time.Now()}
	l := newLocks(c)
	if _, ok := l.Holder("p1"); ok {
		t.Error("holder without lock")
	}
	l.Acquire("p1", "anna", "Anna")
	if h, ok := l.Holder("p1"); !ok || h.Sub != "anna" {
		t.Errorf("holder %+v %v", h, ok)
	}
}
