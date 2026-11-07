// SPDX-License-Identifier: Apache-2.0

package web

import (
	"net/http"
	"slices"

	"github.com/bjblazko/kanso/internal/auth"
	"github.com/bjblazko/kanso/internal/mealplan"
)

type adminPlansPage struct {
	Plans   []mealplan.Plan
	Groups  []string
	Confirm *mealplan.Plan // plan whose deletion waits for confirmation
	Error   string
}

func (s *Server) mealplansAdmin(w http.ResponseWriter, r *http.Request) {
	if s.requireAdmin(w, r) {
		s.renderAdminPlans(w, r, http.StatusOK, adminPlansPage{})
	}
}

func (s *Server) renderAdminPlans(w http.ResponseWriter, r *http.Request, status int, data adminPlansPage) {
	v := s.newView(r, "", nil)
	data.Plans = s.MealPlans.All()
	for name := range s.groups() {
		data.Groups = append(data.Groups, name)
	}
	slices.Sort(data.Groups)
	v.Title, v.Data = v.T("plan.admin"), data
	s.render(w, r, status, "admin_mealplans", v)
}

// updateMealplans creates a plan for a group or (after confirmation) deletes one.
func (s *Server) updateMealplans(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	u, _ := auth.UserFrom(r.Context())
	id := r.PostFormValue("id")
	var err error
	switch r.PostFormValue("action") {
	case "create":
		if _, known := s.groups()[r.PostFormValue("group")]; !known {
			err = mealplan.ErrInvalid
		} else {
			_, err = s.MealPlans.Create(u.Author(), r.PostFormValue("name"), r.PostFormValue("group"))
		}
	case "delete":
		s.confirmPlanDelete(w, r, id)
		return
	case "delete_confirmed":
		err = s.MealPlans.Delete(u.Author(), id)
	}
	if err != nil {
		v := s.newView(r, "", nil)
		s.renderAdminPlans(w, r, http.StatusBadRequest, adminPlansPage{Error: v.T("plan.invalid")})
		return
	}
	http.Redirect(w, r, "/admin/mealplans", http.StatusSeeOther)
}

func (s *Server) confirmPlanDelete(w http.ResponseWriter, r *http.Request, id string) {
	i := slices.IndexFunc(s.MealPlans.All(), func(p mealplan.Plan) bool { return p.ID == id })
	if i < 0 {
		s.notFound(w, r)
		return
	}
	p := s.MealPlans.All()[i]
	s.renderAdminPlans(w, r, http.StatusOK, adminPlansPage{Confirm: &p})
}
