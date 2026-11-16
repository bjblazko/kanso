// SPDX-License-Identifier: Apache-2.0

package wiki

// SpaceStats says what a space holds, for the question before deleting it.
type SpaceStats struct {
	IDs                                 []string // every page id of the space
	Pages, Recipes, Attachments, Shared int      // Shared: pages with their own members
}

// SpaceStats counts the pages of space.
func (w *Wiki) SpaceStats(space string) SpaceStats {
	var st SpaceStats
	for _, loc := range w.locationsIn(space) {
		st.IDs = append(st.IDs, loc.meta.ID)
		st.Pages++
		if loc.meta.IsRecipe() {
			st.Recipes++
		}
		if len(loc.meta.Members) > 0 {
			st.Shared++
		}
	}
	for _, id := range st.IDs {
		st.Attachments += len(w.ListAttachments(id))
	}
	return st
}

// ForgetSpace drops the pages of space from the index, for a space whose
// folder is gone, and reports each as removed.
func (w *Wiki) ForgetSpace(space string) {
	w.mu.Lock()
	var gone []Page
	for id, loc := range w.index {
		if loc.space == space {
			gone = append(gone, Page{Meta: loc.meta, Space: loc.space, Dir: loc.dir})
			delete(w.index, id)
		}
	}
	w.mu.Unlock()
	for _, p := range gone {
		w.notify(p, true)
	}
}

func (w *Wiki) locationsIn(space string) []location {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var locs []location
	for _, loc := range w.index {
		if loc.space == space {
			locs = append(locs, loc)
		}
	}
	return locs
}
