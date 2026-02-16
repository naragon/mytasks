package handlers

import (
	"net/http"
	"time"

	"mytasks/internal/models"
)

// HomeData holds data for the home page template.
type HomeData struct {
	Title              string
	Tab                string // "active" or "completed"
	Projects           []models.Project
	CompletedStartDate string
	CompletedEndDate   string
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

	now := time.Now()
	completedEnd := now
	completedStart := now.AddDate(0, 0, -30)

	if showCompleted {
		if v := r.URL.Query().Get("start_date"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid start_date")
				return
			}
			completedStart = t
		}

		if v := r.URL.Query().Get("end_date"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid end_date")
				return
			}
			completedEnd = t
		}

		if completedStart.After(completedEnd) {
			respondError(w, http.StatusBadRequest, "start_date cannot be after end_date")
			return
		}
	}

	// Load top 3 tasks for each project, filtered by completion status
	filteredProjects := make([]models.Project, 0, len(projects))
	for i := range projects {
		if projects[i].Completed {
			continue
		}

		var (
			tasks []models.Task
			err   error
		)

		if showCompleted {
			tasks, err = h.store.ListTasksByProjectCompletedBetween(ctx, projects[i].ID, &completedStart, &completedEnd, 0)
		} else {
			tasks, err = h.store.ListTasksByProjectFiltered(ctx, projects[i].ID, false, 3)
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		projects[i].ViewTab = tab
		projects[i].Tasks = tasks

		if !showCompleted || len(tasks) > 0 {
			filteredProjects = append(filteredProjects, projects[i])
		}
	}

	data := HomeData{
		Title:              "My Tasks",
		Tab:                tab,
		Projects:           filteredProjects,
		CompletedStartDate: completedStart.Format("2006-01-02"),
		CompletedEndDate:   completedEnd.Format("2006-01-02"),
	}

	h.renderTemplate(w, "home.html", data)
}
