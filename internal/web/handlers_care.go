// SPDX-License-Identifier: Apache-2.0

package web

import (
	"cmp"
	"errors"
	"math/rand/v2"
	"net/http"
	"strings"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/selfcare"
)

// careTip is a first-aid step in the page language.
type careTip struct {
	ID, Title, Body string
	Own             bool // the person's own idea
}

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
	Meditations []careMeditationLink
	Own         []selfcare.Own
	Practice    *careLink // the exercise or meditation that suits the chosen level
}

// careLink is a way to an exercise or a meditation.
type careLink struct{ Href, Title string }

func practiceFor(l selfcare.Level, lang string) *careLink {
	id, med := l.Practice()
	if med {
		m, ok := selfcare.MeditationByID(id)
		if !ok {
			return nil
		}
		return &careLink{Href: "/care/m/" + id, Title: m.Title.In(lang)}
	}
	e, ok := selfcare.ExerciseByID(id)
	if !ok {
		return nil
	}
	return &careLink{Href: "/care/" + id, Title: e.Title.In(lang)}
}

func tipIn(t selfcare.Tip, lang string) careTip {
	return careTip{ID: t.ID, Title: t.Title.In(lang), Body: t.Body.In(lang), Own: strings.HasPrefix(t.ID, "own-")}
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
	u, _ := auth.UserFrom(r.Context())
	d := carePage{Own: s.SelfCare.Own(u.Sub)}
	pool := selfcare.Tips()
	for _, o := range d.Own {
		pool = append(pool, o.Tip())
	}
	d.Tip = tipIn(selfcare.PickFrom(selfcare.Only(pool, false), rand.IntN, q.Get("after")), v.Lang)
	if l, ok := selfcare.LevelByKey(q.Get("level")); ok {
		d.Level, d.Empty, d.Practice = l.Key, l.Empty(), practiceFor(l, v.Lang)
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
	d.Meditations = meditationLinks(v.Lang)
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

// meditationLine is one spoken line: its text, the pause after it and its
// clip per voice.
type meditationLine struct {
	Text  string
	Pause float64
	Clips []meditationClip
}

type meditationClip struct{ Voice, URL string }

type meditationPage struct {
	ID, Title, Lead string
	Minutes         int
	Bell            float64
	QuietEnd        bool
	Voices          []careChoice
	Lines           []meditationLine
}

func (s *Server) meditationView(v view, m selfcare.Meditation) meditationPage {
	d := meditationPage{ID: m.ID, Title: m.Title.In(v.Lang), Lead: m.Lead.In(v.Lang), Minutes: m.Minutes, Bell: selfcare.Bell, QuietEnd: m.QuietEnd}
	lang := "de"
	if v.Lang == "en" {
		lang = "en"
	}
	for i, vc := range selfcare.Voices() {
		d.Voices = append(d.Voices, careChoice{Key: vc.ID, Label: vc.In(v.Lang), Current: i == 0})
	}
	for i, l := range m.Lines {
		line := meditationLine{Text: l.In(v.Lang), Pause: l.Pause}
		for _, vc := range selfcare.Voices() {
			line.Clips = append(line.Clips, meditationClip{Voice: vc.ID, URL: v.Asset(m.AudioPath(vc.ID, lang, i))})
		}
		d.Lines = append(d.Lines, line)
	}
	return d
}

// careMeditation is a guided meditation: the page plays one clip per line
// with the pauses between them (meditation.js).
func (s *Server) careMeditation(w http.ResponseWriter, r *http.Request) {
	m, ok := selfcare.MeditationByID(r.PathValue("id"))
	if !ok {
		s.notFound(w, r)
		return
	}
	v := s.newView(r, "", nil)
	d := s.meditationView(v, m)
	v.Title, v.Data = d.Title, d
	s.render(w, r, http.StatusOK, "care_meditation", v)
}

// careMeditationLink is a meditation in the Selfcare overview.
type careMeditationLink struct {
	ID, Title, Lead string
	Minutes         int
}

func meditationLinks(lang string) []careMeditationLink {
	var out []careMeditationLink
	for _, m := range selfcare.Meditations() {
		out = append(out, careMeditationLink{ID: m.ID, Title: m.Title.In(lang), Lead: m.Lead.In(lang), Minutes: m.Minutes})
	}
	return out
}

// careOwnAdd keeps one more of the person's own first-aid ideas.
func (s *Server) careOwnAdd(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	err := s.SelfCare.Add(u.Sub, r.PostFormValue("title"), r.PostFormValue("body"), cmp.Or(r.PostFormValue("kind"), "calm"))
	if errors.Is(err, selfcare.ErrInvalid) {
		s.fail(w, r, http.StatusBadRequest, "care.own_invalid")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/care#care-own", http.StatusSeeOther)
}

func (s *Server) careOwnDelete(w http.ResponseWriter, r *http.Request) {
	u, _ := auth.UserFrom(r.Context())
	if err := s.SelfCare.Delete(u.Sub, r.PathValue("id")); err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/care#care-own", http.StatusSeeOther)
}

// careBetween is a short step between two things: close what is done, do
// something small for yourself, then arrive at what comes next.
func (s *Server) careBetween(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	v.Title = v.T("care.between")
	u, _ := auth.UserFrom(r.Context())
	pool := selfcare.Tips()
	for _, o := range s.SelfCare.Own(u.Sub) {
		pool = append(pool, o.Tip())
	}
	var tips []careTip
	for _, t := range selfcare.Only(pool, true) {
		tips = append(tips, tipIn(t, v.Lang))
	}
	first := rand.IntN(len(tips))
	tips[0], tips[first] = tips[first], tips[0]
	v.Data = tips
	s.render(w, r, http.StatusOK, "care_between", v)
}

// careBetweenNote keeps what was done and what comes next as a private note.
func (s *Server) careBetweenNote(w http.ResponseWriter, r *http.Request) {
	v := s.newView(r, "", nil)
	e := careExercise{Title: v.T("care.between"), Steps: []string{v.T("care.between_done"), v.T("care.between_next")}}
	body := careNoteBody(e, r, time.Now())
	if body == "" {
		http.Redirect(w, r, "/care/between", http.StatusSeeOther)
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
