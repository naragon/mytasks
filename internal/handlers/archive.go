package handlers

import (
	"net/http"

	"mytasks/internal/models"
)

// ArchivedProjectEntry combines a project with its tasks grouped by status.
type ArchivedProjectEntry struct {
	models.Project
	TodoTasks       []models.Task
	InProgressTasks []models.Task
	DoneTasks       []models.Task
}

// ArchiveData holds data for the Archive template.
type ArchiveData struct {
	PageData
	ArchivedProjects []ArchivedProjectEntry
}

// Archive renders the completed/archived projects view.
func (h *Handlers) Archive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	completedProjects, err := h.store.ListCompletedProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	entries := make([]ArchivedProjectEntry, 0, len(completedProjects))
	for _, p := range completedProjects {
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
		done, err := h.store.ListTasksByProjectAndStatus(ctx, p.ID, "done")
		if err != nil {
			respondServerError(w, err)
			return
		}
		entries = append(entries, ArchivedProjectEntry{
			Project:         p,
			TodoTasks:       todo,
			InProgressTasks: inProgress,
			DoneTasks:       done,
		})
	}

	activeProjects, err := h.loadActiveProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	data := ArchiveData{
		PageData: PageData{
			Title:          "Archive",
			ActiveProjects: activeProjects,
			CurrentView:    "archive",
		},
		ArchivedProjects: entries,
	}

	h.renderTemplate(w, "archive.html", data)
}
