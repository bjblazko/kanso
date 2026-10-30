// SPDX-License-Identifier: Apache-2.0

package devseed

import (
	"time"

	"github.com/bjblazko/kanso/internal/store"
	"github.com/bjblazko/kanso/internal/tasks"
)

// taskSeed is a made-up task and where it stands.
type taskSeed struct {
	by       string
	new      tasks.New
	dueIn    int // days from today; 0 means none unless today is set
	today    bool
	status   tasks.Status
	waiting  string
	claimant string
}

var familyTasks = []taskSeed{
	{by: "anna", new: tasks.New{Title: "Lohnsteuer machen", Important: true}, dueIn: 21},
	{by: "anna", new: tasks.New{Title: "Kind zur Ferienbetreuung anmelden", Urgent: true, Important: true,
		Body: "Per Mail an den Hort, **Formular** liegt im Ordner Schule."}, status: tasks.Waiting, waiting: "Antwort Hort"},
	{by: "ben", new: tasks.New{Title: "Hof fegen", Urgent: true}, today: true, status: tasks.Doing, claimant: "ben"},
	{by: "lena", new: tasks.New{Title: "Fahrrad reparieren"}, status: tasks.Done},
}

var annaTasks = []taskSeed{
	{by: "anna", new: tasks.New{Title: "Zählerstand an die Stadtwerke melden", Urgent: true}, dueIn: 3},
	{by: "anna", new: tasks.New{Title: "Geschenk für Oma Gerda besorgen", Important: true}},
}

// seedTasks makes the family board with tasks in every column and two private tasks for Anna.
func seedTasks(admin store.Author, ts *tasks.Store) error {
	family, err := ts.CreateGroupBoard(admin, "Familie", "familie")
	if err != nil {
		return err
	}
	for _, t := range familyTasks {
		t.new.Board = family.ID
		if err := seedTask(ts, t); err != nil {
			return err
		}
	}
	for _, t := range annaTasks {
		if err := seedTask(ts, t); err != nil {
			return err
		}
	}
	return nil
}

func seedTask(ts *tasks.Store, s taskSeed) error {
	a := tasks.Actor{Sub: s.by, Author: store.Author{Name: s.by, Email: s.by + "@example.org"}, Groups: groups}
	if s.dueIn > 0 || s.today {
		s.new.Due = time.Now().AddDate(0, 0, s.dueIn).Format("2006-01-02")
	}
	t, _, err := ts.Create(a, s.new)
	if err == nil && s.status == tasks.Waiting {
		_, err = ts.Update(a, t.Board, t.Meta.ID, tasks.Edit{Title: t.Meta.Title, Body: t.Body, Status: tasks.Waiting,
			Urgent: t.Meta.Urgent, Important: t.Meta.Important, Due: t.Meta.Due, WaitingFor: s.waiting}, t.Meta.Updated)
	} else if err == nil && s.status != "" {
		_, err = ts.SetStatus(a, t.Board, t.Meta.ID, s.status)
	}
	if err == nil && s.claimant != "" {
		_, err = ts.Claim(a, t.Board, t.Meta.ID, true)
	}
	return err
}
