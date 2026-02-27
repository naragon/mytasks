package handlers

import (
	"encoding/json"
	"net/http"

	"mytasks/internal/models"
)

// ProjectDetailData holds data for the project detail page.
type ProjectDetailData struct {
	Title   string
	Project *models.Project
}

// ProjectDetail renders the project detail page with active (not completed) tasks.
func (h *Handlers) ProjectDetail(w http.ResponseWriter, r *http.Request) {
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

	// Load active tasks only (no limit)
	tasks, err := h.store.ListTasksByProjectFiltered(ctx, id, false, 0)
	if err != nil {
		respondServerError(w, err)
		return
	}
	for i := range tasks {
		tasks[i].InlineEdit = true
	}
	project.Tasks = tasks

	data := ProjectDetailData{
		Title:   project.Name,
		Project: project,
	}

	h.renderTemplate(w, "project_detail.html", data)
}

// CreateProject creates a new project.
func (h *Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	project := &models.Project{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Type:        r.FormValue("type"),
		TargetDate:  parseDate(r.FormValue("target_date")),
	}

	if err := project.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.CreateProject(ctx, project); err != nil {
		respondServerError(w, err)
		return
	}

	h.renderPartial(w, "project_card.html", project)
}

// UpdateProject updates an existing project.
func (h *Handlers) UpdateProject(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseForm(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid form data")
		return
	}

	project.Name = r.FormValue("name")
	project.Description = r.FormValue("description")
	project.Type = r.FormValue("type")
	project.TargetDate = parseDate(r.FormValue("target_date"))
	if project.Type == "category" {
		project.TargetDate = nil
	}

	if err := project.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.UpdateProject(ctx, project); err != nil {
		respondServerError(w, err)
		return
	}

	w.Header().Set("HX-Refresh", "true")
	w.WriteHeader(http.StatusOK)
}

// DeleteProject deletes a project.
func (h *Handlers) DeleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.store.DeleteProject(ctx, id); err != nil {
		respondServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// CompleteProject marks a project as completed.
func (h *Handlers) CompleteProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.store.MarkProjectComplete(ctx, id); err != nil {
		respondServerError(w, err)
		return
	}

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// ReopenProject marks a completed project as incomplete.
func (h *Handlers) ReopenProject(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid project id")
		return
	}

	if err := h.store.MarkProjectIncomplete(ctx, id); err != nil {
		respondServerError(w, err)
		return
	}

	w.Header().Set("HX-Redirect", "/")
	w.WriteHeader(http.StatusOK)
}

// ReorderProjects updates the order of projects.
func (h *Handlers) ReorderProjects(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var payload struct {
		IDs []int64 `json:"ids"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.store.ReorderProjects(ctx, payload.IDs); err != nil {
		respondServerError(w, err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// GetProjectForm returns the project form for editing.
func (h *Handlers) GetProjectForm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := parseID(r, "id")
	if err != nil {
		// New project form
		h.renderPartial(w, "project_form.html", nil)
		return
	}

	project, err := h.store.GetProject(ctx, id)
	if err != nil {
		respondError(w, http.StatusNotFound, "project not found")
		return
	}

	h.renderPartial(w, "project_form.html", project)
}
