package main

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"mytasks/internal/handlers"
	"mytasks/internal/store"
)

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS

func main() {
	// Configuration
	port := getEnv("PORT", "8080")
	dbPath := getEnv("DB_PATH", "./data/mytasks.db")

	// Ensure data directory exists
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		log.Fatalf("Failed to create data directory: %v", err)
	}

	// Initialize store
	s, err := store.NewSQLiteStore(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer s.Close()

	// Parse templates
	tmpl, err := parseTemplates()
	if err != nil {
		log.Fatalf("Failed to parse templates: %v", err)
	}

	// Initialize handlers
	h := handlers.New(s, tmpl)

	// Create router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Compress(5))

	// Static files
	staticSub, _ := fs.Sub(staticFS, "static")
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(staticSub))))

	// Page routes
	r.Get("/", h.Home)
	r.Get("/projects/{id}", h.ProjectDetail)

	// Project API routes
	r.Post("/api/projects", h.CreateProject)
	r.Put("/api/projects/{id}", h.UpdateProject)
	r.Post("/api/projects/{id}/complete", h.CompleteProject)
	r.Post("/api/projects/{id}/reopen", h.ReopenProject)
	r.Delete("/api/projects/{id}", h.DeleteProject)
	r.Post("/api/projects/reorder", h.ReorderProjects)

	// Task API routes
	r.Post("/api/projects/{id}/tasks", h.CreateTask)
	r.Put("/api/tasks/{id}", h.UpdateTask)
	r.Delete("/api/tasks/{id}", h.DeleteTask)
	r.Post("/api/tasks/{id}/toggle", h.ToggleTask)
	r.Post("/api/projects/{id}/tasks/reorder", h.ReorderTasks)

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("Starting server on http://localhost%s", addr)
	if err := http.ListenAndServe(addr, r); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func parseTemplates() (*template.Template, error) {
	// Custom template functions
	funcMap := template.FuncMap{
		"dict": func(values ...interface{}) map[string]interface{} {
			if len(values)%2 != 0 {
				return nil
			}
			dict := make(map[string]interface{}, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				key, ok := values[i].(string)
				if !ok {
					continue
				}
				dict[key] = values[i+1]
			}
			return dict
		},
	}

	tmpl := template.New("").Funcs(funcMap)

	// Parse all templates
	patterns := []string{
		"templates/*.html",
		"templates/partials/*.html",
	}

	for _, pattern := range patterns {
		matches, err := fs.Glob(templatesFS, pattern)
		if err != nil {
			return nil, fmt.Errorf("failed to glob pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			content, err := templatesFS.ReadFile(match)
			if err != nil {
				return nil, fmt.Errorf("failed to read template %s: %w", match, err)
			}

			name := filepath.Base(match)
			_, err = tmpl.New(name).Parse(string(content))
			if err != nil {
				return nil, fmt.Errorf("failed to parse template %s: %w", name, err)
			}
		}
	}

	return tmpl, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
