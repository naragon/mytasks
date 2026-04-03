package handlers

import (
	"net/http"
	"strconv"

	"mytasks/internal/models"
)

// UpcomingData holds data for the Upcoming tasks template.
type UpcomingData struct {
	PageData
	UpcomingTasks []models.Task
	UpcomingDays  int
}

// Upcoming renders the cross-project upcoming tasks view.
func (h *Handlers) Upcoming(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	days := 30
	if v := r.URL.Query().Get("days"); v != "" {
		d, err := strconv.Atoi(v)
		if err == nil && (d == 7 || d == 14 || d == 30) {
			days = d
		}
	}

	tasks, err := h.store.ListUpcomingTasks(ctx, days)
	if err != nil {
		respondServerError(w, err)
		return
	}

	activeProjects, err := h.loadActiveProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	data := UpcomingData{
		PageData: PageData{
			Title:          "Upcoming",
			ActiveProjects: activeProjects,
			CurrentView:    "upcoming",
		},
		UpcomingTasks: tasks,
		UpcomingDays:  days,
	}

	h.renderTemplate(w, "upcoming.html", data)
}
