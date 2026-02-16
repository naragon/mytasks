package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"mytasks/internal/store"
)

// Handlers holds the HTTP handlers and their dependencies.
type Handlers struct {
	store     store.Store
	templates *template.Template
}

// New creates a new Handlers instance.
func New(s store.Store, tmpl *template.Template) *Handlers {
	return &Handlers{
		store:     s,
		templates: tmpl,
	}
}

// parseID extracts and parses an integer ID from URL parameters.
func parseID(r *http.Request, param string) (int64, error) {
	idStr := chi.URLParam(r, param)
	return strconv.ParseInt(idStr, 10, 64)
}

// parseDate parses a date string in YYYY-MM-DD format.
func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

// respondError sends an error response.
func respondError(w http.ResponseWriter, code int, message string) {
	w.WriteHeader(code)
	w.Write([]byte(message))
}

// respondJSON sends a JSON response.
func respondJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// renderTemplate renders a template with the given data.
func (h *Handlers) renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	if h.templates == nil {
		// For testing without templates
		w.WriteHeader(http.StatusOK)
		return
	}
	err := h.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
	}
}

// renderPartial renders a partial template (for htmx responses).
func (h *Handlers) renderPartial(w http.ResponseWriter, name string, data interface{}) {
	if h.templates == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	err := h.templates.ExecuteTemplate(w, name, data)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
	}
}
