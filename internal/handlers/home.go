package handlers

import (
	"net/http"

	"mytasks/internal/models"
)

// HomeData holds data for the home page template.
type HomeData struct {
	Title    string
	Tab      string // "active" or "completed"
	Projects []models.Project
}

// Home renders the home page with all projects and their tasks filtered by tab.
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the tab from query parameter, default to "active"
	tab := r.URL.Query().Get("tab")
	if tab != "completed" {
		tab = "active"
	}

	projects, err := h.store.ListProjects(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter tasks based on tab (completed = true for completed tab, false for active)
	showCompleted := tab == "completed"

	// Load top 3 tasks for each project, filtered by completion status
	filteredProjects := make([]models.Project, 0)
	for i := range projects {
		tasks, err := h.store.ListTasksByProjectFiltered(ctx, projects[i].ID, showCompleted, 3)
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}
		projects[i].Tasks = tasks
		// Only include projects that have tasks matching the filter
		if len(tasks) > 0 {
			filteredProjects = append(filteredProjects, projects[i])
		}
	}

	data := HomeData{
		Title:    "My Tasks",
		Tab:      tab,
		Projects: filteredProjects,
	}

	h.renderTemplate(w, "home.html", data)
}
