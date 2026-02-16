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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
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

	// Set sort order to be at the end
	projects, _ := h.store.ListProjects(ctx)
	project.SortOrder = len(projects) + 1

	if err := h.store.CreateProject(ctx, project); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
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

	if err := project.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.store.UpdateProject(ctx, project); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Load tasks for the response
	tasks, _ := h.store.ListTasksByProject(ctx, id, 3)
	project.Tasks = tasks

	h.renderPartial(w, "project_card.html", project)
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
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

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
		respondError(w, http.StatusInternalServerError, err.Error())
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
