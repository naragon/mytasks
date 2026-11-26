package handlers

import (
	"net/http"
	"time"

	"mytasks/internal/models"
)

// DashboardProject holds project information and its tasks grouped by status.
type DashboardProject struct {
	models.Project
	TodoTasks       []models.Task
	InProgressTasks []models.Task
	DoneTasks       []models.Task
}

// Home renders the landing page showing a single view of all tasks across projects.
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	activeProjects, err := h.loadActiveProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	var dashboardProjects []DashboardProject
	since := time.Now().AddDate(0, 0, -7)

	for _, p := range activeProjects {
		todo, err := h.store.ListTasksByProjectAndStatus(ctx, p.ID, "todo")
		if err != nil {
			respondServerError(w, err)
			return
		}
		inProgress, err := h.store.ListTasksByProjectAndStatus(ctx, p.ID, "in_progress")
		if err != nil {
			respondServerError(w, err)
			return
		}
		done, err := h.store.ListRecentDoneTasks(ctx, p.ID, since)
		if err != nil {
			respondServerError(w, err)
			return
		}

		// Set overdue flag
		for i := range todo {
			todo[i].Overdue = todo[i].IsOverdue()
		}
		for i := range inProgress {
			inProgress[i].Overdue = inProgress[i].IsOverdue()
		}

		dashboardProjects = append(dashboardProjects, DashboardProject{
			Project:         p,
			TodoTasks:       todo,
			InProgressTasks: inProgress,
			DoneTasks:       done,
		})
	}

	data := struct {
		PageData
		Projects []DashboardProject
	}{
		PageData: PageData{
			Title:          "My Tasks",
			ActiveProjects: activeProjects,
			CurrentView:    "home",
		},
		Projects: dashboardProjects,
	}

	h.renderTemplate(w, "home.html", data)
}
