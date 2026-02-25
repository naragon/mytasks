package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

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

	task := &models.Task{
		ProjectID:   projectID,
		Description: r.FormValue("description"),
		Notes:       r.FormValue("notes"),
		Priority:    r.FormValue("priority"),
		DueDate:     parseDate(r.FormValue("due_date")),
	}

	if err := task.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Set sort order to be at the end
	tasks, _ := h.store.ListTasksByProject(ctx, projectID, 0)
	task.SortOrder = len(tasks) + 1

	if err := h.store.CreateTask(ctx, task); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
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

	if r.FormValue("completed") == "true" {
		task.Completed = true
	} else {
		task.Completed = false
	}

	if err := task.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.UpdateTask(ctx, task); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
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
		respondError(w, http.StatusInternalServerError, err.Error())
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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Return the updated task
	task, err := h.store.GetTask(ctx, id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.renderPartial(w, "task_item.html", task)
}

// ReorderTasks updates the order of tasks within a project.
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

	if err := h.store.ReorderTasks(ctx, projectID, payload.IDs); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
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
