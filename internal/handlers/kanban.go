package handlers

import (
	"net/http"
	"time"

	"mytasks/internal/models"
)

const donePruneWindowDays = 7

// KanbanData holds data for the Kanban board template.
type KanbanData struct {
	PageData
	Project         *models.Project
	TodoTasks       []models.Task
	InProgressTasks []models.Task
	DoneTasks       []models.Task
}

// KanbanBoard renders the Kanban board for a project.
func (h *Handlers) KanbanBoard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	project, err := h.store.GetProject(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	todoTasks, err := h.store.ListTasksByProjectAndStatus(ctx, id, "todo")
	if err != nil {
		respondServerError(w, err)
		return
	}

	inProgressTasks, err := h.store.ListTasksByProjectAndStatus(ctx, id, "in_progress")
	if err != nil {
		respondServerError(w, err)
		return
	}

	since := time.Now().AddDate(0, 0, -donePruneWindowDays)
	doneTasks, err := h.store.ListRecentDoneTasks(ctx, id, since)
	if err != nil {
		respondServerError(w, err)
		return
	}

	activeProjects, err := h.loadActiveProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	// Set overdue flag on tasks
	for i := range todoTasks {
		todoTasks[i].Overdue = todoTasks[i].IsOverdue()
	}
	for i := range inProgressTasks {
		inProgressTasks[i].Overdue = inProgressTasks[i].IsOverdue()
	}

	data := KanbanData{
		PageData: PageData{
			Title:            project.Name,
			ActiveProjects:   activeProjects,
			CurrentProjectID: id,
			CurrentView:      "kanban",
		},
		Project:         project,
		TodoTasks:       todoTasks,
		InProgressTasks: inProgressTasks,
		DoneTasks:       doneTasks,
	}

	h.renderTemplate(w, "kanban.html", data)
}
