package handlers

import (
	"fmt"
	"net/http"
)

// Home redirects to the first active project's Kanban board, or shows an empty state.
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	activeProjects, err := h.loadActiveProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	if len(activeProjects) > 0 {
		http.Redirect(w, r, fmt.Sprintf("/projects/%d", activeProjects[0].ID), http.StatusFound)
		return
	}

	// No active projects — show empty state with sidebar
	data := PageData{
		Title:          "My Tasks",
		ActiveProjects: activeProjects,
		CurrentView:    "home",
	}

	h.renderTemplate(w, "empty.html", data)
}
