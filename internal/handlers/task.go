package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"mytasks/internal/models"
)

// CreateTask creates a new task for a project.
func (h *Handlers) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	projectID, err := parseID(r, "id")
	if err != nil {
		projectID, err = strconv.ParseInt(r.FormValue("project_id"), 10, 64)
		if err != nil || projectID <= 0 {
			respondError(w, http.StatusBadRequest, "invalid project id")
			return
		}
	}

	status := r.FormValue("status")
	if status == "" {
		status = "todo"
	}

	task := &models.Task{
		ProjectID:   projectID,
		Description: r.FormValue("description"),
		Notes:       r.FormValue("notes"),
		Priority:    r.FormValue("priority"),
		Status:      status,
		DueDate:     parseDate(r.FormValue("due_date")),
	}

	if err := task.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.CreateTask(ctx, task); err != nil {
		respondServerError(w, err)
		return
	}

	h.renderPartial(w, "task_item.html", task)
}

// UpdateTask updates an existing task.
func (h *Handlers) UpdateTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	task, err := h.store.GetTask(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}

	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	task.Description = r.FormValue("description")
	task.Notes = r.FormValue("notes")
	task.Priority = r.FormValue("priority")
	task.DueDate = parseDate(r.FormValue("due_date"))

	if status := r.FormValue("status"); status != "" {
		task.Status = status
	}

	// Support legacy completed checkbox — sync to status
	if r.FormValue("completed") == "true" {
		task.Status = "done"
	}

	// Move to another project if project_id supplied and differs from current.
	if rawProjectID := r.FormValue("project_id"); rawProjectID != "" {
		destID, err := strconv.ParseInt(rawProjectID, 10, 64)
		if err != nil {
			respondError(w, http.StatusBadRequest, "invalid project_id")
			return
		}
		if destID != task.ProjectID {
			dest, err := h.store.GetProject(ctx, destID)
			if err != nil || dest.Completed {
				respondError(w, http.StatusBadRequest, "invalid destination project")
				return
			}
			task.ProjectID = destID
		}
	}

	if err := task.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.UpdateTask(ctx, task); err != nil {
		respondServerError(w, err)
		return
	}

	h.renderPartial(w, "task_item.html", task)
}

// DeleteTask deletes a task.
func (h *Handlers) DeleteTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.store.DeleteTask(ctx, id); err != nil {
		respondServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ToggleTask toggles the completion status of a task.
func (h *Handlers) ToggleTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	if err := h.store.ToggleTaskComplete(ctx, id); err != nil {
		respondServerError(w, err)
		return
	}

	// Return the updated task
	task, err := h.store.GetTask(ctx, id)
	if err != nil {
		respondServerError(w, err)
		return
	}

	h.renderPartial(w, "task_item.html", task)
}

// MoveTask changes a task's status (Kanban column move).
func (h *Handlers) MoveTask(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task id")
		return
	}

	var payload struct {
		Status    string `json:"status"`
		SortOrder int    `json:"sort_order"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if payload.Status != "todo" && payload.Status != "in_progress" && payload.Status != "done" {
		respondError(w, http.StatusBadRequest, "invalid status")
		return
	}

	if err := h.store.MoveTaskToStatus(ctx, id, payload.Status, payload.SortOrder); err != nil {
		respondServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ReorderTasks updates the order of tasks within a project.
// Accepts an optional "status" query parameter to scope the reorder.
func (h *Handlers) ReorderTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	projectID, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	var payload struct {
		IDs []int64 `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}

	status := r.URL.Query().Get("status")
	if status != "" {
		if err := h.store.ReorderTasksInStatus(ctx, projectID, status, payload.IDs); err != nil {
			respondServerError(w, err)
			return
		}
	} else {
		if err := h.store.ReorderTasks(ctx, projectID, payload.IDs); err != nil {
			respondServerError(w, err)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// GetTaskForm returns the task form for editing.
func (h *Handlers) GetTaskForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		// New task form - need project ID from URL
		projectID, _ := parseID(r, "project_id")
		h.renderPartial(w, "task_form.html", map[string]interface{}{
			"ProjectID": projectID,
		})
		return
	}

	task, err := h.store.GetTask(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "task not found")
		return
	}

	h.renderPartial(w, "task_form.html", task)
}

// ListTasks returns all tasks, optionally filtered by completion window.
// Query params:
//   - completed_within_days: optional non-negative integer; when set, only done tasks completed within the last N days are returned.
func (h *Handlers) ListTasks(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var completedSince *time.Time
	if rawDays := r.URL.Query().Get("completed_within_days"); rawDays != "" {
		days, err := strconv.Atoi(rawDays)
		if err != nil || days < 0 {
			respondError(w, http.StatusBadRequest, "invalid completed_within_days")
			return
		}

		since := time.Now().AddDate(0, 0, -days)
		completedSince = &since
	}

	tasks, err := h.store.ListTasks(ctx, completedSince)
	if err != nil {
		respondServerError(w, err)
		return
	}
	if tasks == nil {
		tasks = []models.Task{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(tasks); err != nil {
		respondServerError(w, err)
		return
	}
}
