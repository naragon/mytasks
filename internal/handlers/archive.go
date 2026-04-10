package handlers

import (
	"net/http"
	"time"

	"mytasks/internal/models"
)

// ArchivedProjectEntry combines a project with its tasks grouped by status.
type ArchivedProjectEntry struct {
	models.Project
	IsProjectCompleted bool
	TodoTasks          []models.Task
	InProgressTasks    []models.Task
	DoneTasks          []models.Task
}

// ArchiveData holds data for archive templates.
type ArchiveData struct {
	PageData
	ArchivedProjects []ArchivedProjectEntry
}

// Archive redirects to the completed tasks view.
func (h *Handlers) Archive(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/archive/tasks", http.StatusFound)
}

// CompletedProjects renders completed projects and all of their tasks.
func (h *Handlers) CompletedProjects(w http.ResponseWriter, r *http.Request) {
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
			Project:            p,
			IsProjectCompleted: true,
			TodoTasks:          todo,
			InProgressTasks:    inProgress,
			DoneTasks:          done,
		})
	}

	activeProjects, err := h.loadActiveProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	data := ArchiveData{
		PageData: PageData{
			Title:          "Completed Projects",
			ActiveProjects: activeProjects,
			CurrentView:    "completed_projects",
		},
		ArchivedProjects: entries,
	}

	h.renderTemplate(w, "archive_projects.html", data)
}

// CompletedTasks renders old completed tasks for active projects, grouped by project.
func (h *Handlers) CompletedTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	before := time.Now().AddDate(0, 0, -donePruneWindowDays)

	activeWithOld, err := h.store.ListActiveProjectsWithOldDoneTasks(ctx, before)
	if err != nil {
		respondServerError(w, err)
		return
	}

	entries := make([]ArchivedProjectEntry, 0, len(activeWithOld))
	for _, p := range activeWithOld {
		oldDone, err := h.store.ListOldDoneTasks(ctx, p.ID, before)
		if err != nil {
			respondServerError(w, err)
			return
		}
		entries = append(entries, ArchivedProjectEntry{
			Project:   p,
			DoneTasks: oldDone,
		})
	}

	activeProjects, err := h.loadActiveProjects(ctx)
	if err != nil {
		respondServerError(w, err)
		return
	}

	data := ArchiveData{
		PageData: PageData{
			Title:          "Completed Tasks",
			ActiveProjects: activeProjects,
			CurrentView:    "completed_tasks",
		},
		ArchivedProjects: entries,
	}

	h.renderTemplate(w, "archive_tasks.html", data)
}
