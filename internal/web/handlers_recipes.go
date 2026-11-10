// SPDX-License-Identifier: Apache-2.0

package web

import (
	"slices"
	"time"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/wiki"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/markdown"
	"github.com/bjblazko/kanso/internal/recipes"
)

// cookItem is an ingredient or step with the key its tick is kept under on the device.
type cookItem struct {
	Tick  string
	Ing   recipes.Ingredient
	Step  string
	Photo string // address of the step's photo; "" = none
	// Text and Note are Step or the ingredient's name and note with their
	// inline Markdown (bold, italics, links) rendered and sanitised.
	Text, Note template.HTML
}

type cookGroup struct {
	Name  string
	Items []cookItem
}

type cookView struct {
	pageCtx
	Groups      []cookGroup
	Steps       []cookItem
	Servings    int
	HasServings bool
	CookedToday bool
}

// Fewer and More are the links of the servings stepper; "" at the limits.
func (c cookView) Fewer() string { return c.servingsLink(c.Servings - 1) }

// More is the link to one serving more.
func (c cookView) More() string { return c.servingsLink(c.Servings + 1) }

func (c cookView) servingsLink(n int) string {
	if n < 1 || n > recipes.MaxServings {
		return ""
	}
	return "/recipes/" + c.Page.Meta.ID + "/cook?servings=" + strconv.Itoa(n)
}

// cookRecipe shows a recipe for the stove: scaled ingredients and steps to tick off.
func (s *Server) cookRecipe(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Read)
	if !ok {
		return
	}
	if !pc.Page.Meta.IsRecipe() {
		s.notFound(w, r)
		return
	}
	cv := cookView{pageCtx: pc, HasServings: pc.Page.Meta.Servings > 0,
		CookedToday: slices.Contains(pc.Page.Meta.Cooked, time.Now().Format(dayLayout))}
	base := max(pc.Page.Meta.Servings, 1)
	cv.Servings = recipes.ClampServings(r.URL.Query().Get("servings"), base)
	v := s.newView(r, pc.Page.Meta.Title, nil)
	cv.fill(recipes.Parse(pc.Page.Body), float64(cv.Servings)/float64(base), v.Lang)
	s.renderInline(&cv)
	v.Data = cv
	s.render(w, r, http.StatusOK, "recipe_cook", v)
}

// markCooked notes today as a day the recipe was cooked; once per day.
func (s *Server) markCooked(w http.ResponseWriter, r *http.Request) {
	pc, ok := s.pageFor(w, r, access.Write)
	if !ok {
		return
	}
	if !pc.Page.Meta.IsRecipe() {
		s.notFound(w, r)
		return
	}
	u, _ := auth.UserFrom(r.Context())
	today := time.Now().Format(dayLayout)
	err := s.Wiki.SaveWith(u.Author(), pc.Page.Meta.ID, pc.Page.Meta.Title, pc.Page.Body, func(m *wiki.PageMeta) {
		if !slices.Contains(m.Cooked, today) {
			m.Cooked = append(m.Cooked, today)
			slices.Sort(m.Cooked)
		}
	})
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	http.Redirect(w, r, "/recipes/"+pc.Page.Meta.ID+"/cook", http.StatusSeeOther)
}

// renderInline turns the Markdown inside steps and ingredients into HTML.
func (s *Server) renderInline(cv *cookView) {
	base := markdown.PageCtx{Base: "/p/" + cv.Page.Meta.ID}
	for g := range cv.Groups {
		for i := range cv.Groups[g].Items {
			it := &cv.Groups[g].Items[i]
			it.Text, it.Note = s.inline(it.Ing.Name, base), s.inline(it.Ing.Note, base)
		}
	}
	for i := range cv.Steps {
		cv.Steps[i].Text = s.inline(cv.Steps[i].Step, base)
	}
}

// inline renders one line of Markdown without its paragraph; raw HTML stays text.
func (s *Server) inline(md string, base markdown.PageCtx) template.HTML {
	if md == "" {
		return ""
	}
	out, _, err := s.MD.Render([]byte(md), base)
	if err != nil {
		return template.HTML(template.HTMLEscapeString(md)) // #nosec G203 -- escaped
	}
	h := strings.TrimSpace(string(out))
	if strings.HasPrefix(h, "<p>") && strings.HasSuffix(h, "</p>") && strings.Count(h, "<p>") == 1 {
		h = h[3 : len(h)-4]
	}
	return template.HTML(h) // #nosec G203 -- sanitised by the Markdown renderer
}

// fill scales the recipe by factor and numbers every tick.
func (c *cookView) fill(rec recipes.Recipe, factor float64, lang string) {
	n := 0
	for _, g := range rec.Groups {
		cg := cookGroup{Name: g.Name}
		for _, ing := range g.Ingredients {
			cg.Items = append(cg.Items, cookItem{Tick: "i" + strconv.Itoa(n), Ing: ing.Scale(factor, lang)})
			n++
		}
		c.Groups = append(c.Groups, cg)
	}
	for i, step := range rec.Steps {
		item := cookItem{Tick: "s" + strconv.Itoa(i), Step: step}
		if img := rec.StepImages[i]; strings.HasPrefix(img, "attachments/") {
			item.Photo = "/p/" + c.Page.Meta.ID + "/" + img
		}
		c.Steps = append(c.Steps, item)
	}
}
