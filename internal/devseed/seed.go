// SPDX-License-Identifier: Apache-2.0

// Package devseed fills an empty data directory with made-up people, groups,
// spaces and pages for local development. It is used only with the dev login.
package devseed

import (
	"errors"
	"time"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/spaces"
	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/users"
	"github.com/bjblazko/kanso/internal/wiki"
)

// ErrNotEmpty means the data directory already has spaces; seeding never
// touches existing data.
var ErrNotEmpty = errors.New("devseed: data directory already has spaces")

// Person is a made-up user the dev login offers for a quick switch.
type Person struct {
	Sub   string
	Name  string
	Admin bool
	Note  string // what this person may do in the seeded data
}

// People are the made-up users, the admin first.
var People = []Person{
	{Sub: "anna", Name: "Anna", Admin: true, Note: "admin"},
	{Sub: "ben", Name: "Ben", Note: "owner of Haus, group familie"},
	{Sub: "lena", Name: "Lena", Note: "owner of Lenas Ecke, group familie, viewer of Haus"},
	{Sub: "oma", Name: "Oma Gerda", Note: "viewer of Familie"},
	{Sub: "gast", Name: "Gast", Note: "no spaces"},
}

type spaceSeed struct {
	name, desc, icon string
	members          []access.Member
	pages            []pageSeed
}

type pageSeed struct {
	title, body string
	children    []pageSeed
}

var groups = access.Groups{"familie": {"anna", "ben", "lena"}}

var seedSpaces = []spaceSeed{
	{name: "Familie", desc: "Termine, Geburtstage, Absprachen", icon: "users",
		members: []access.Member{{Group: "familie", Role: access.Editor}, {User: "oma", Role: access.Viewer}},
		pages:   []pageSeed{{title: "Willkommen", body: welcomeBody, children: []pageSeed{{title: "Geburtstage", body: birthdaysBody}}}}},
	{name: "Haus", desc: "Heizung, Geräte, Wartung", icon: "home",
		members: []access.Member{{User: "ben", Role: access.Owner}, {User: "lena", Role: access.Viewer}},
		pages:   []pageSeed{{title: "Heizung", body: heatingBody}}},
	{name: "Lenas Ecke", icon: "star",
		members: []access.Member{{User: "lena", Role: access.Owner}},
		pages:   []pageSeed{{title: "Meine Bücher", body: booksBody}}},
}

// Seed writes people, groups, spaces and pages into an empty data directory.
func Seed(dataDir string, repo *store.Repo, us *users.Store, sp *spaces.Service, wk *wiki.Wiki) error {
	if len(sp.List()) > 0 {
		return ErrNotEmpty
	}
	admin, err := seedPeople(us)
	if err != nil {
		return err
	}
	if err := access.SaveGroups(dataDir, groups); err != nil {
		return err
	}
	if err := repo.Commit(admin, "Testgruppen angelegt", access.ConfigFile); err != nil {
		return err
	}
	for _, s := range seedSpaces {
		if err := seedSpace(admin, sp, wk, s); err != nil {
			return err
		}
	}
	return nil
}

// seedPeople creates the profiles and returns the admin as commit author.
func seedPeople(us *users.Store) (store.Author, error) {
	var admin store.Author
	for _, p := range People {
		u, err := us.Upsert(users.User{Sub: p.Sub, Name: p.Name, Email: p.Sub + "@example.org", LastLogin: time.Now()})
		if err != nil {
			return admin, err
		}
		if p.Admin {
			admin = u.Author()
		}
	}
	return admin, nil
}

func seedSpace(a store.Author, sp *spaces.Service, wk *wiki.Wiki, s spaceSeed) error {
	space, err := sp.Create(a, s.name, s.desc)
	if err != nil {
		return err
	}
	space.Icon, space.Members = s.icon, s.members
	if err := sp.Update(a, space); err != nil {
		return err
	}
	return seedPages(a, wk, space.Slug, "", s.pages)
}

func seedPages(a store.Author, wk *wiki.Wiki, space, parentID string, pages []pageSeed) error {
	for _, p := range pages {
		page, err := wk.Create(a, space, parentID, p.title)
		if err != nil {
			return err
		}
		if err := wk.Save(a, page.Meta.ID, p.title, p.body); err != nil {
			return err
		}
		if err := seedPages(a, wk, space, page.Meta.ID, p.children); err != nil {
			return err
		}
	}
	return nil
}
