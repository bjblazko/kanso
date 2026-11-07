// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/bjblazko/kanso/internal/access"
	"github.com/bjblazko/kanso/internal/recipes"
)

// cookItem is an ingredient or step with the key its tick is kept under on the device.
type cookItem struct {
	Tick  string
	Ing   recipes.Ingredient
	Step  string
	Photo string // address of the step's photo; "" = none
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
	cv := cookView{pageCtx: pc, HasServings: pc.Page.Meta.Servings > 0}
	base := max(pc.Page.Meta.Servings, 1)
	cv.Servings = recipes.ClampServings(r.URL.Query().Get("servings"), base)
	v := s.newView(r, pc.Page.Meta.Title, nil)
	cv.fill(recipes.Parse(pc.Page.Body), float64(cv.Servings)/float64(base), v.Lang)
	v.Data = cv
	s.render(w, r, http.StatusOK, "recipe_cook", v)
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
