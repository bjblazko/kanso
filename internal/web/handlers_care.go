// SPDX-License-Identifier: Apache-2.0

package web

import (
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/selfcare"
)

// careTip is a first-aid step in the page language.
type careTip struct{ ID, Title, Body string }

type careChoice struct {
	Key, Label string
	Current    bool
}

type careExercise struct {
	ID, Title, Lead, Kind string
	Steps                 []string
}

type carePage struct {
	Tip         careTip
	Levels      []careChoice
	Level       string // chosen battery level, "" before any
	Suggestions []careTip
	Empty       bool
	Exercises   []careExercise
}

func tipIn(t selfcare.Tip, lang string) careTip {
	return careTip{ID: t.ID, Title: t.Title.In(lang), Body: t.Body.In(lang)}
}

func exerciseIn(e selfcare.Exercise, lang string) careExercise {
	out := careExercise{ID: e.ID, Title: e.Title.In(lang), Lead: e.Lead.In(lang), Kind: e.Kind}
	for _, s := range e.Steps {
		out.Steps = append(out.Steps, s.In(lang))
	}
	return out
}

// carePage shows a random first-aid step, the battery check and the
// exercises. Nothing the person picks here is stored.
func (s *Server) carePage(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("care.title")
	q := r.URL.Query()
	d := carePage{Tip: tipIn(selfcare.Pick(rand.IntN, q.Get("after")), v.Lang)}
	if l, ok := selfcare.LevelByKey(q.Get("level")); ok {
		d.Level, d.Empty = l.Key, l.Empty()
		for _, t := range l.Suggestions() {
			d.Suggestions = append(d.Suggestions, tipIn(t, v.Lang))
		}
	}
	for _, l := range selfcare.Levels {
		d.Levels = append(d.Levels, careChoice{Key: l.Key, Label: l.Label.In(v.Lang), Current: l.Key == d.Level})
	}
	for _, e := range selfcare.Exercises() {
		d.Exercises = append(d.Exercises, exerciseIn(e, v.Lang))
	}
	v.Data = d
	s.render(w, r, http.StatusOK, "care", v)
}

func (s *Server) careExercise(w http.ResponseWriter, r *http.Request) {
	e, ok := selfcare.ExerciseByID(r.PathValue("id"))
	if !ok {
		s.notFound(w, r)
		return
	}
	v := s.newView(r, "", nil)
	d := exerciseIn(e, v.Lang)
	v.Title, v.Data = d.Title, d
	s.render(w, r, http.StatusOK, "care_exercise", v)
}

// careNote keeps a check-in's answers as a private note, on request only.
func (s *Server) careNote(w http.ResponseWriter, r *http.Request) {
	e, ok := selfcare.ExerciseByID(r.PathValue("id"))
	if !ok || e.Kind != "note" {
		s.notFound(w, r)
		return
	}
	v := s.newView(r, "", nil)
	body := careNoteBody(exerciseIn(e, v.Lang), r, time.Now())
	if body == "" {
		http.Redirect(w, r, "/care/"+e.ID, http.StatusSeeOther)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	n, _, err := s.Notes.Create(u.Sub, "", body, nil, 0)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/notes/"+n.Meta.ID, http.StatusSeeOther)
}

// careNoteBody writes the answered steps as Markdown under the exercise's
// title and the date; "" when nothing was answered.
func careNoteBody(e careExercise, r *http.Request, now time.Time) string {
	var b strings.Builder
	for i, step := range e.Steps {
		answer := strings.TrimSpace(r.PostFormValue("a" + string(rune('0'+i))))
		if answer != "" {
			b.WriteString("**" + step + "**\n" + answer + "\n\n")
		}
	}
	if b.Len() == 0 {
		return ""
	}
	return e.Title + " – " + now.Format("02.01.2006") + "\n\n" + b.String()
}
