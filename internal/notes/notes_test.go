// SPDX-License-Identifier: Apache-2.0

package notes

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bjblazko/kanso/internal/store"
)

const max = 1 << 20

func newStore(t *testing.T) *Store {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	repo, err := store.OpenRepo(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewStore(repo.Dir(), repo)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func mustCreate(t *testing.T, s *Store, owner, id, body string, files ...Upload) Note {
	t.Helper()
	n, _, err := s.Create(owner, id, body, files, max)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestCreateAndGet(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "Milch kaufen\n")
	if !idPattern.MatchString(n.Meta.ID) || n.Meta.Created.IsZero() {
		t.Fatalf("note = %+v", n)
	}
	got, err := s.Get("anna", n.Meta.ID)
	if err != nil || got.Body != "Milch kaufen\n" || got.Owner != "anna" {
		t.Errorf("get = %+v, %v", got, err)
	}
}

func TestCreateIsIdempotent(t *testing.T) {
	s := newStore(t)
	mustCreate(t, s, "anna", "abcdefghijkl", "erste\n")
	n, created, err := s.Create("anna", "abcdefghijkl", "zweite\n", nil, max)
	if err != nil || created || n.Body != "erste\n" {
		t.Errorf("second create = %+v, %v, %v", n, created, err)
	}
	if len(s.List("anna")) != 1 {
		t.Errorf("notes = %d", len(s.List("anna")))
	}
}

func TestCreateRejectsBadIDs(t *testing.T) {
	s := newStore(t)
	for _, id := range []string{"../x", "ABCDEFGHIJKL", "abc", "abcdefghijklm", "abcdefghijk1"} {
		if _, _, err := s.Create("anna", id, "x", nil, max); !errors.Is(err, ErrBadID) {
			t.Errorf("id %q: %v", id, err)
		}
	}
	for _, owner := range []string{"", "../anna", "a/b"} {
		if _, _, err := s.Create(owner, "", "x", nil, max); !errors.Is(err, ErrBadID) {
			t.Errorf("owner %q: %v", owner, err)
		}
	}
}

func TestCreateWithPhotosLinksThem(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "Typenschild\n",
		Upload{Name: "IMG 1.jpg", R: strings.NewReader("a")}, Upload{Name: "IMG 1.jpg", R: strings.NewReader("b")})
	if !strings.HasSuffix(n.Body, "\n![](attachments/img-1.jpg)\n![](attachments/img-1-2.jpg)\n") {
		t.Errorf("body = %q", n.Body)
	}
	if len(n.Attachments) != 2 {
		t.Errorf("attachments = %v", n.Attachments)
	}
}

func TestCreateCleansPhotos(t *testing.T) {
	s := newStore(t)
	in, err := os.ReadFile("../photo/testdata/gps.jpg")
	if err != nil {
		t.Fatal(err)
	}
	n := mustCreate(t, s, "anna", "", "", Upload{Name: "foto.jpeg", R: bytes.NewReader(in)})
	f, err := s.OpenAttachment("anna", n.Meta.ID, n.Attachments[0])
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var got bytes.Buffer
	_, _ = got.ReadFrom(f)
	if bytes.Equal(got.Bytes(), in) {
		t.Error("photo stored with its location")
	}
}

func TestFileNamesStayInside(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "x", Upload{Name: "../../evil.jpg", R: strings.NewReader("a")})
	dir, err := s.Dir("anna", n.Meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "attachments", "evil.jpg")); err != nil {
		t.Error(err)
	}
	if _, err := s.OpenAttachment("anna", n.Meta.ID, "../index.md"); err == nil {
		t.Error("opened a file outside attachments")
	}
}

func TestTooLargeUploadLeavesNothing(t *testing.T) {
	s := newStore(t)
	_, _, err := s.Create("anna", "abcdefghijkl", "x", []Upload{{Name: "a.jpg", R: strings.NewReader("123456")}}, 5)
	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("err = %v", err)
	}
	if _, err := s.Get("anna", "abcdefghijkl"); !errors.Is(err, ErrNotFound) {
		t.Errorf("half a note left: %v", err)
	}
}

func TestOtherOwnersSeeNothing(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "geheim")
	if _, err := s.Get("ben", n.Meta.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("ben get: %v", err)
	}
	if len(s.List("ben")) != 0 {
		t.Error("ben lists anna's note")
	}
	if err := s.Delete("ben", n.Meta.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("ben delete: %v", err)
	}
	if _, err := s.Get("anna", n.Meta.ID); err != nil {
		t.Errorf("note gone after ben's delete: %v", err)
	}
}

func TestSaveDetectsConflict(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "eins")
	saved, err := s.Save("anna", n.Meta.ID, "zwei", n.Meta.Updated)
	if err != nil || saved.Body != "zwei" {
		t.Fatalf("save = %+v, %v", saved, err)
	}
	if _, err := s.Save("anna", n.Meta.ID, "drei", n.Meta.Updated); !errors.Is(err, ErrConflict) {
		t.Errorf("stale save: %v", err)
	}
	if got, _ := s.Get("anna", n.Meta.ID); got.Body != "zwei" {
		t.Errorf("body = %q", got.Body)
	}
}

func TestNotesStayOutOfGit(t *testing.T) {
	s := newStore(t)
	mustCreate(t, s, "anna", "", "nicht im Verlauf", Upload{Name: "a.png", R: strings.NewReader("x")})
	if _, err := NewStore(s.dataDir, s.repo); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("git", "-C", s.dataDir, "status", "--porcelain").Output()
	if err != nil || len(out) != 0 {
		t.Errorf("git status = %q, %v", out, err)
	}
	ignore, _ := os.ReadFile(filepath.Join(s.dataDir, ".gitignore"))
	if strings.Count(string(ignore), "/notes/\n") != 1 {
		t.Errorf(".gitignore = %q", ignore)
	}
}

func TestDeleteRemovesFolder(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "weg")
	dir, _ := s.Dir("anna", n.Meta.ID)
	if err := s.Delete("anna", n.Meta.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("folder still there: %v", err)
	}
}

func TestListNewestFirst(t *testing.T) {
	s := newStore(t)
	now := time.Date(2026, 10, 27, 12, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }
	mustCreate(t, s, "anna", "", "alt")
	now = now.Add(time.Minute)
	mustCreate(t, s, "anna", "", "neu")
	l := s.List("anna")
	if len(l) != 2 || l[0].Body != "neu" {
		t.Errorf("list = %+v", l)
	}
}

func TestChangesAreReported(t *testing.T) {
	s := newStore(t)
	var events []string
	s.OnChange = func(n Note, deleted bool) {
		events = append(events, n.Meta.ID+map[bool]string{true: "-", false: "+"}[deleted])
	}
	n := mustCreate(t, s, "anna", "", "x")
	if _, err := s.Save("anna", n.Meta.ID, "y", n.Meta.Updated); err != nil {
		t.Fatal(err)
	}
	if err := s.MoveOut("anna", n.Meta.ID, func(dir string) error { return os.RemoveAll(dir) }); err != nil {
		t.Fatal(err)
	}
	id := n.Meta.ID
	if strings.Join(events, " ") != id+"+ "+id+"+ "+id+"-" {
		t.Errorf("events = %v", events)
	}
}

func TestAllCoversEveryOwner(t *testing.T) {
	s := newStore(t)
	mustCreate(t, s, "anna", "", "a")
	mustCreate(t, s, "ben", "", "b")
	if len(s.All()) != 2 {
		t.Errorf("all = %d", len(s.All()))
	}
}

func TestTitle(t *testing.T) {
	cases := map[string]string{
		"\n\n# Einkauf\nx":         "Einkauf",
		"- Milch\n- Brot":          "Milch",
		"* Punkt":                  "Punkt",
		strings.Repeat("a", 100):   strings.Repeat("a", 80),
		"![](attachments/a.jpg)\n": "",
		"":                         "",
		"   Zählerstand   ":        "Zählerstand",
	}
	for in, want := range cases {
		if got := Title(Note{Body: in}); got != want {
			t.Errorf("Title(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestLinkTask(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "Hof fegen\n")
	if err := s.LinkTask("anna", n.Meta.ID, "aaaaaaaaaaaa/bbbbbbbbbbbb"); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("anna", n.Meta.ID)
	if err != nil || got.Meta.Task != "aaaaaaaaaaaa/bbbbbbbbbbbb" || got.Body != n.Body || !got.Meta.Updated.Equal(n.Meta.Updated) {
		t.Fatalf("note = %+v, %v", got, err)
	}
}

func TestLinkTaskForeignNote(t *testing.T) {
	s := newStore(t)
	n := mustCreate(t, s, "anna", "", "Hof fegen\n")
	if err := s.LinkTask("ben", n.Meta.ID, "x/y"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("LinkTask = %v", err)
	}
}
