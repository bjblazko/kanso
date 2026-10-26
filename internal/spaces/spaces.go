// SPDX-License-Identifier: Apache-2.0

// Package spaces manages the top-level areas of the wiki and their members.
package spaces

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/store"
)

// Dir holds all spaces, relative to the data directory.
const Dir = "spaces"

// Icons a space may use; drawn in web/templates/icons.html.
var Icons = []string{"folder", "home", "heart", "book", "users", "star"}

// DefaultIcon is used for new spaces.
const DefaultIcon = "folder"

// Space is one area of the wiki (spaces/<slug>/space.yaml).
type Space struct {
	Slug        string          `yaml:"-"`
	Name        string          `yaml:"name"`
	Description string          `yaml:"description,omitempty"`
	Icon        string          `yaml:"icon"`
	Members     []access.Member `yaml:"members"`
}

// Service keeps all spaces in memory; this process is the only writer.
type Service struct {
	dataDir string
	repo    *store.Repo
	mu      sync.RWMutex
	spaces  map[string]Space
}

// NewService loads every space below dataDir/spaces.
func NewService(dataDir string, repo *store.Repo) (*Service, error) {
	svc := &Service{dataDir: dataDir, repo: repo, spaces: map[string]Space{}}
	files, err := filepath.Glob(filepath.Join(dataDir, Dir, "*", "space.yaml"))
	if err != nil {
		return nil, err
	}
	for _, f := range files {
		var sp Space
		if err := store.ReadYAML(f, &sp); err != nil {
			return nil, err
		}
		sp.Slug = filepath.Base(filepath.Dir(f))
		svc.spaces[sp.Slug] = sp
	}
	return svc, nil
}

// List returns all spaces sorted by name.
func (svc *Service) List() []Space {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	list := make([]Space, 0, len(svc.spaces))
	for _, sp := range svc.spaces {
		list = append(list, sp)
	}
	slices.SortFunc(list, func(a, b Space) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return list
}

// Get returns the space with slug.
func (svc *Service) Get(slug string) (Space, bool) {
	svc.mu.RLock()
	defer svc.mu.RUnlock()
	sp, ok := svc.spaces[slug]
	return sp, ok
}

// RoleOf returns p's role in the space slug.
func (svc *Service) RoleOf(p access.Principal, slug string, g access.Groups) access.Role {
	sp, ok := svc.Get(slug)
	if !ok {
		return access.None
	}
	return access.Resolve(p, sp.Members, g)
}

// Visible returns the spaces p may read.
func (svc *Service) Visible(p access.Principal, g access.Groups) []Space {
	var vis []Space
	for _, sp := range svc.List() {
		if access.Can(access.Resolve(p, sp.Members, g), access.Read) {
			vis = append(vis, sp)
		}
	}
	return vis
}

// Create adds a space named name.
func (svc *Service) Create(a store.Author, name, desc string) (Space, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Space{}, errors.New("a space needs a name")
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	slug := store.UniqueSlug(filepath.Join(svc.dataDir, Dir), store.Slugify(name))
	sp := Space{Slug: slug, Name: name, Description: strings.TrimSpace(desc), Icon: DefaultIcon}
	return sp, svc.save(a, sp, "Bereich angelegt: "+name)
}

// Update stores changed name, description, icon or members.
func (svc *Service) Update(a store.Author, sp Space) error {
	if err := validate(sp); err != nil {
		return err
	}
	svc.mu.Lock()
	defer svc.mu.Unlock()
	if _, ok := svc.spaces[sp.Slug]; !ok {
		return errors.New("unknown space")
	}
	return svc.save(a, sp, "Bereich geändert: "+sp.Name)
}

func validate(sp Space) error {
	if strings.TrimSpace(sp.Name) == "" {
		return errors.New("a space needs a name")
	}
	if !slices.Contains(Icons, sp.Icon) {
		return fmt.Errorf("unknown icon %q", sp.Icon)
	}
	for _, m := range sp.Members {
		if (m.User == "") == (m.Group == "") || m.Role == access.None {
			return errors.New("a member is either a user or a group and has a role")
		}
	}
	return nil
}

// save writes and commits sp; the caller holds svc.mu.
func (svc *Service) save(a store.Author, sp Space, msg string) error {
	rel := filepath.Join(Dir, sp.Slug, "space.yaml")
	err := svc.repo.Locked(func() error {
		if err := store.WriteYAML(filepath.Join(svc.dataDir, rel), sp); err != nil {
			return err
		}
		return svc.repo.Commit(a, msg, rel)
	})
	if err == nil {
		svc.spaces[sp.Slug] = sp
	}
	return err
}
