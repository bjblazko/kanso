// SPDX-License-Identifier: Apache-2.0

package todos

import (
	"errors"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/bjblazko/kanso/internal/access"
)

// Limits of one list and of one sync.
const (
	MaxName      = 80
	MaxText      = 500
	maxItems     = 1000
	MaxOps       = 500
	tombstoneAge = 30 * 24 * time.Hour
)

// Op is one change made on a device. At is the device time in ms; for each
// field of an item the later change wins. Sending an op twice does no harm.
type Op struct {
	ID    string `json:"id"`
	Kind  string `json:"kind"` // create, rename, add, text, done, sub, delete
	List  string `json:"list"`
	Item  string `json:"item,omitempty"`
	After string `json:"after,omitempty"` // add: insert below this item; "" = at the end
	Text  string `json:"text,omitempty"`
	Done  bool   `json:"done,omitempty"`
	Sub   bool   `json:"sub,omitempty"`
	At    int64  `json:"at"`
}

var errRejected = errors.New("operation rejected")

// Apply runs ops for sub in order and returns the ids of those it refused
// (unknown or invisible list, owner-only change, invalid text, limits).
// Refused or not, every op is done with: the device may drop it.
func (s *Store) Apply(sub string, g access.Groups, ops []Op) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	lists := map[string]*List{}
	changed := map[string]bool{}
	var rejected []string
	for _, op := range ops {
		l, err := s.listFor(sub, g, op, lists)
		if err == nil {
			var c bool
			if c, err = applyOp(l, sub, op, s.now()); c {
				changed[l.ID] = true
			}
		}
		if err != nil {
			rejected = append(rejected, op.ID)
		}
	}
	s.save(lists, changed)
	return rejected
}

// listFor loads (or, for create, makes) the list an op works on.
func (s *Store) listFor(sub string, g access.Groups, op Op, lists map[string]*List) (*List, error) {
	if l, ok := lists[op.List]; ok {
		return l, nil
	}
	l, err := s.read(op.List)
	switch {
	case errors.Is(err, ErrNotFound) && op.Kind == "create" && idPattern.MatchString(op.List):
		now := s.now().UTC()
		l = List{ID: op.List, Owner: sub, Created: now, Updated: now}
	case err != nil:
		return nil, err
	case !canSee(l, sub, g):
		return nil, ErrNotFound
	}
	lists[op.List] = &l
	return &l, nil
}

func (s *Store) save(lists map[string]*List, changed map[string]bool) {
	now := s.now().UTC()
	for id := range changed {
		l := lists[id]
		l.Updated = now
		_ = s.write(*l) // a failed write leaves the old state; the device sends again
	}
}

// applyOp changes l by op; it reports whether anything changed.
func applyOp(l *List, sub string, op Op, now time.Time) (bool, error) {
	switch op.Kind {
	case "create", "rename":
		return rename(l, sub, op)
	case "add":
		return addItem(l, op)
	case "text", "done", "sub":
		return setField(l, op)
	case "delete":
		return deleteItem(l, op, now), nil
	}
	return false, errRejected
}

func validText(s string, max int) bool {
	return strings.TrimSpace(s) != "" && utf8.RuneCountInString(s) <= max
}

func rename(l *List, sub string, op Op) (bool, error) {
	if l.Owner != sub || !validText(op.Text, MaxName) {
		return false, errRejected
	}
	if op.Kind == "create" && l.Name != "" {
		return false, nil // created before: the same op again
	}
	l.Name = strings.TrimSpace(op.Text)
	return true, nil
}

func (l *List) index(id string) int {
	return slices.IndexFunc(l.Items, func(it Item) bool { return it.ID == id })
}

func addItem(l *List, op Op) (bool, error) {
	if !idPattern.MatchString(op.Item) || !validText(op.Text, MaxText) {
		return false, errRejected
	}
	if l.index(op.Item) >= 0 || l.Deleted[op.Item] != 0 {
		return false, nil
	}
	if len(l.Items) >= maxItems {
		return false, errRejected
	}
	at := l.index(op.After) + 1 // not found: -1 + 1 would be the top, so append instead
	if op.After == "" || at == 0 {
		at = len(l.Items)
	}
	it := Item{ID: op.Item, Text: strings.TrimSpace(op.Text), Sub: op.Sub && at > 0,
		At: Stamps{Text: op.At, Done: op.At, Sub: op.At}}
	l.Items = slices.Insert(l.Items, at, it)
	return true, nil
}

// setField changes one field of an item if op is the later change.
func setField(l *List, op Op) (bool, error) {
	if op.Kind == "text" && !validText(op.Text, MaxText) {
		return false, errRejected
	}
	i := l.index(op.Item)
	if i < 0 {
		return false, nil // deleted meanwhile
	}
	it := &l.Items[i]
	switch {
	case op.Kind == "text" && op.At >= it.At.Text:
		it.Text, it.At.Text = strings.TrimSpace(op.Text), op.At
	case op.Kind == "done" && op.At >= it.At.Done:
		it.Done, it.At.Done = op.Done, op.At
	case op.Kind == "sub" && op.At >= it.At.Sub && (i > 0 || !op.Sub):
		it.Sub, it.At.Sub = op.Sub, op.At
	default:
		return false, nil
	}
	return true, nil
}

// deleteItem removes an item; a main item takes its sub-items with it.
func deleteItem(l *List, op Op, now time.Time) bool {
	i := l.index(op.Item)
	if i < 0 {
		return false
	}
	end := i + 1
	for !l.Items[i].Sub && end < len(l.Items) && l.Items[end].Sub {
		end++
	}
	if l.Deleted == nil {
		l.Deleted = map[string]int64{}
	}
	for _, it := range l.Items[i:end] {
		l.Deleted[it.ID] = now.UnixMilli()
	}
	l.Items = slices.Delete(l.Items, i, end)
	return true
}
