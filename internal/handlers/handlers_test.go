package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"mytasks/internal/models"
	"mytasks/internal/store"
)

func setupTestHandlers(t *testing.T) (*Handlers, *store.SQLiteStore) {
	t.Helper()
	s, err := store.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	h := New(s, nil) // nil templates for API tests
	return h, s
}

func setupTestHandlersWithTemplates(t *testing.T) (*Handlers, *store.SQLiteStore) {
	t.Helper()
	h, s := setupTestHandlers(t)

	funcMap := template.FuncMap{
		"add": func(a, b int) int { return a + b },
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
	files, err := filepath.Glob("../../templates/*.html")
	if err != nil {
		t.Fatalf("failed to glob page templates: %v", err)
	}
	partialFiles, err := filepath.Glob("../../templates/partials/*.html")
	if err != nil {
		t.Fatalf("failed to glob partial templates: %v", err)
	}
	files = append(files, partialFiles...)

	tmpl, err = tmpl.ParseFiles(files...)
	if err != nil {
		t.Fatalf("failed to parse templates: %v", err)
	}

	h.templates = tmpl
	return h, s
}

func TestHomeHandler_RedirectsToFirstProject(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	s.CreateProject(ctx, &models.Project{Name: "Project A", Type: "project", SortOrder: 1})
	s.CreateProject(ctx, &models.Project{Name: "Project B", Type: "project", SortOrder: 2})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	if rec.Code != http.StatusFound {
		t.Errorf("expected status %d, got %d", http.StatusFound, rec.Code)
	}
	location := rec.Header().Get("Location")
	if location != "/projects/1" {
		t.Errorf("expected redirect to /projects/1, got %s", location)
	}
}

func TestHomeHandler_EmptyStateWhenNoProjects(t *testing.T) {
	h, _ := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	// With nil templates, renders as 200
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestKanbanBoardHandler_ShowsAllTasks(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)

	for i := 1; i <= 5; i++ {
		s.CreateTask(ctx, &models.Task{
			ProjectID:   project.ID,
			Description: "Task",
			Priority:    "medium",
			SortOrder:   i,
		})
	}

	req := httptest.NewRequest("GET", "/projects/1", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.KanbanBoard(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestCreateProjectHandler_Success(t *testing.T) {
	h, _ := setupTestHandlers(t)

	form := url.Values{}
	form.Set("name", "New Project")
	form.Set("type", "project")
	form.Set("description", "A new project")

	req := httptest.NewRequest("POST", "/api/projects", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestCreateProjectHandler_ValidationError(t *testing.T) {
	h, _ := setupTestHandlers(t)

	form := url.Values{}
	form.Set("name", "")
	form.Set("type", "project")

	req := httptest.NewRequest("POST", "/api/projects", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.CreateProject(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateProjectHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Original", Type: "project"}
	s.CreateProject(ctx, project)

	form := url.Values{}
	form.Set("name", "Updated")
	form.Set("type", "project")
	form.Set("description", "Updated description")

	req := httptest.NewRequest("PUT", "/api/projects/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestUpdateProjectHandler_CanChangeToCategoryAndSetDescription(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	targetDate := "2026-03-01"
	project := &models.Project{Name: "Original", Type: "project", TargetDate: parseDate(targetDate)}
	s.CreateProject(ctx, project)

	form := url.Values{}
	form.Set("name", "Updated Name")
	form.Set("type", "category")
	form.Set("description", "Updated description")
	form.Set("target_date", targetDate)

	req := httptest.NewRequest("PUT", "/api/projects/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	updated, err := s.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if updated.Type != "category" {
		t.Fatalf("expected type category, got %s", updated.Type)
	}
	if updated.Description != "Updated description" {
		t.Fatalf("expected description to persist, got %q", updated.Description)
	}
	if updated.TargetDate != nil {
		t.Fatalf("expected target date to be nil for category, got %v", updated.TargetDate)
	}
}

func TestDeleteProjectHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)

	req := httptest.NewRequest("DELETE", "/api/projects/1", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DeleteProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	_, err := s.GetProject(ctx, 1)
	if err == nil {
		t.Error("expected project to be deleted")
	}
}

func TestCompleteProjectHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)

	req := httptest.NewRequest("POST", "/api/projects/1/complete", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.CompleteProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	updated, err := s.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if !updated.Completed {
		t.Fatal("expected project to be marked completed")
	}
}

func TestReopenProjectHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)
	s.MarkProjectComplete(ctx, project.ID)

	req := httptest.NewRequest("POST", "/api/projects/1/reopen", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ReopenProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	updated, err := s.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}
	if updated.Completed {
		t.Fatal("expected project to be reopened")
	}
}

func TestReorderProjectsHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	p1 := &models.Project{Name: "A", Type: "project"}
	p2 := &models.Project{Name: "B", Type: "project"}
	s.CreateProject(ctx, p1)
	s.CreateProject(ctx, p2)

	body, _ := json.Marshal(map[string][]int64{"ids": {2, 1}})
	req := httptest.NewRequest("POST", "/api/projects/reorder", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ReorderProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	projects, _ := s.ListProjects(ctx)
	if projects[0].Name != "B" {
		t.Errorf("expected first project to be B, got %s", projects[0].Name)
	}
}

func TestCreateTaskHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)

	form := url.Values{}
	form.Set("description", "New Task")
	form.Set("priority", "high")

	req := httptest.NewRequest("POST", "/api/projects/1/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.CreateTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestCreateTaskHandler_SuccessWithProjectFromForm(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)

	form := url.Values{}
	form.Set("project_id", "1")
	form.Set("description", "New Task")
	form.Set("priority", "high")

	req := httptest.NewRequest("POST", "/api/tasks", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	h.CreateTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestUpdateTaskHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)
	task := &models.Task{ProjectID: project.ID, Description: "Original", Priority: "low"}
	s.CreateTask(ctx, task)

	form := url.Values{}
	form.Set("description", "Updated")
	form.Set("priority", "high")

	req := httptest.NewRequest("PUT", "/api/tasks/1", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}
}

func TestDeleteTaskHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)
	task := &models.Task{ProjectID: project.ID, Description: "Test", Priority: "medium"}
	s.CreateTask(ctx, task)

	req := httptest.NewRequest("DELETE", "/api/tasks/1", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DeleteTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	_, err := s.GetTask(ctx, 1)
	if err == nil {
		t.Error("expected task to be deleted")
	}
}

func TestToggleTaskHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)
	task := &models.Task{ProjectID: project.ID, Description: "Test", Priority: "medium", Completed: false}
	s.CreateTask(ctx, task)

	req := httptest.NewRequest("POST", "/api/tasks/1/toggle", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ToggleTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	updated, _ := s.GetTask(ctx, 1)
	if !updated.Completed {
		t.Error("expected task to be completed")
	}
}

func TestReorderTasksHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)

	t1 := &models.Task{ProjectID: project.ID, Description: "A", Priority: "medium"}
	t2 := &models.Task{ProjectID: project.ID, Description: "B", Priority: "medium"}
	s.CreateTask(ctx, t1)
	s.CreateTask(ctx, t2)

	body, _ := json.Marshal(map[string][]int64{"ids": {2, 1}})
	req := httptest.NewRequest("POST", "/api/projects/1/tasks/reorder", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ReorderTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	tasks, _ := s.ListTasksByProject(ctx, project.ID, 0)
	if tasks[0].Description != "B" {
		t.Errorf("expected first task to be B, got %s", tasks[0].Description)
	}
}

func TestReorderProjectsHandler_InvalidJSON(t *testing.T) {
	h, _ := setupTestHandlers(t)

	req := httptest.NewRequest("POST", "/api/projects/reorder", strings.NewReader("{"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ReorderProjects(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestUpdateTaskHandler_NotFound(t *testing.T) {
	h, _ := setupTestHandlers(t)

	form := url.Values{}
	form.Set("description", "Updated")
	form.Set("priority", "high")

	req := httptest.NewRequest("PUT", "/api/tasks/999", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateTask(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, rec.Code)
	}
}

func TestDeleteProjectHandler_NotFound(t *testing.T) {
	h, _ := setupTestHandlers(t)

	req := httptest.NewRequest("DELETE", "/api/projects/999", nil)
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "999")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.DeleteProject(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d for idempotent delete, got %d", http.StatusOK, rec.Code)
	}
}

func TestMoveTaskHandler_Success(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)
	task := &models.Task{ProjectID: project.ID, Description: "Test", Priority: "medium"}
	s.CreateTask(ctx, task)

	body, _ := json.Marshal(map[string]interface{}{"status": "in_progress", "sort_order": 1})
	req := httptest.NewRequest("POST", "/api/tasks/1/move", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.MoveTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	updated, _ := s.GetTask(ctx, 1)
	if updated.Status != "in_progress" {
		t.Fatalf("expected status in_progress, got %s", updated.Status)
	}
	if updated.Completed {
		t.Fatal("expected task not to be completed")
	}
}

func TestMoveTaskHandler_ToDone(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)
	task := &models.Task{ProjectID: project.ID, Description: "Test", Priority: "medium"}
	s.CreateTask(ctx, task)

	body, _ := json.Marshal(map[string]interface{}{"status": "done", "sort_order": 1})
	req := httptest.NewRequest("POST", "/api/tasks/1/move", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.MoveTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	updated, _ := s.GetTask(ctx, 1)
	if updated.Status != "done" {
		t.Fatalf("expected status done, got %s", updated.Status)
	}
	if !updated.Completed {
		t.Fatal("expected task to be completed")
	}
	if updated.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
}

func TestArchiveHandler_RedirectsToCompletedTasks(t *testing.T) {
	h, _ := setupTestHandlers(t)

	req := httptest.NewRequest("GET", "/archive", nil)
	rec := httptest.NewRecorder()

	h.Archive(rec, req)

	if rec.Code != http.StatusFound {
		t.Fatalf("expected %d, got %d", http.StatusFound, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/archive/tasks" {
		t.Fatalf("expected redirect to /archive/tasks, got %q", got)
	}
}

func TestCompletedTasksHandler_IncludesOnlyActiveProjectsWithOldDoneTasks(t *testing.T) {
	h, s := setupTestHandlersWithTemplates(t)
	ctx := context.Background()

	oldProject := &models.Project{Name: "Admin", Type: "project"}
	if err := s.CreateProject(ctx, oldProject); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	oldTask := &models.Task{ProjectID: oldProject.ID, Description: "Old work", Priority: "medium", Status: "todo"}
	if err := s.CreateTask(ctx, oldTask); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := s.MoveTaskToStatus(ctx, oldTask.ID, "done", 0); err != nil {
		t.Fatalf("MoveTaskToStatus: %v", err)
	}
	oldDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	if _, err := s.DB().ExecContext(ctx, `UPDATE tasks SET completed_at = ? WHERE id = ?`, oldDate, oldTask.ID); err != nil {
		t.Fatalf("set completed_at: %v", err)
	}

	recentProject := &models.Project{Name: "Sprint", Type: "project"}
	if err := s.CreateProject(ctx, recentProject); err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	recentTask := &models.Task{ProjectID: recentProject.ID, Description: "Recent work", Priority: "low", Status: "todo"}
	if err := s.CreateTask(ctx, recentTask); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if err := s.MoveTaskToStatus(ctx, recentTask.ID, "done", 0); err != nil {
		t.Fatalf("MoveTaskToStatus: %v", err)
	}

	completedProject := &models.Project{Name: "Completed", Type: "project"}
	if err := s.CreateProject(ctx, completedProject); err != nil {
		t.Fatalf("CreateProject completed: %v", err)
	}
	if err := s.MarkProjectComplete(ctx, completedProject.ID); err != nil {
		t.Fatalf("MarkProjectComplete: %v", err)
	}

	req := httptest.NewRequest("GET", "/archive/tasks", nil)
	rec := httptest.NewRecorder()
	h.CompletedTasks(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, fmt.Sprintf(`id="project-%d"`, oldProject.ID)) {
		t.Fatalf("expected completed tasks cards to include old active project %q", oldProject.Name)
	}
	if strings.Contains(body, fmt.Sprintf(`id="project-%d"`, recentProject.ID)) {
		t.Fatalf("did not expect recent project %q in completed tasks cards", recentProject.Name)
	}
	if strings.Contains(body, fmt.Sprintf(`id="project-%d"`, completedProject.ID)) {
		t.Fatalf("did not expect completed project %q in completed tasks cards", completedProject.Name)
	}
}

func TestCompletedProjectsHandler_ShowsOnlyCompletedProjects(t *testing.T) {
	h, s := setupTestHandlersWithTemplates(t)
	ctx := context.Background()

	activeProject := &models.Project{Name: "Active", Type: "project"}
	if err := s.CreateProject(ctx, activeProject); err != nil {
		t.Fatalf("CreateProject active: %v", err)
	}
	activeTask := &models.Task{ProjectID: activeProject.ID, Description: "Old active task", Priority: "medium", Status: "todo"}
	if err := s.CreateTask(ctx, activeTask); err != nil {
		t.Fatalf("CreateTask active: %v", err)
	}
	if err := s.MoveTaskToStatus(ctx, activeTask.ID, "done", 0); err != nil {
		t.Fatalf("MoveTaskToStatus active: %v", err)
	}
	oldDate := time.Now().AddDate(0, 0, -10).Format("2006-01-02")
	if _, err := s.DB().ExecContext(ctx, `UPDATE tasks SET completed_at = ? WHERE id = ?`, oldDate, activeTask.ID); err != nil {
		t.Fatalf("set active completed_at: %v", err)
	}

	completedProject := &models.Project{Name: "Shipped", Type: "project"}
	if err := s.CreateProject(ctx, completedProject); err != nil {
		t.Fatalf("CreateProject completed: %v", err)
	}
	completedTask := &models.Task{ProjectID: completedProject.ID, Description: "Shipped task", Priority: "high", Status: "todo"}
	if err := s.CreateTask(ctx, completedTask); err != nil {
		t.Fatalf("CreateTask completed: %v", err)
	}
	if err := s.MarkProjectComplete(ctx, completedProject.ID); err != nil {
		t.Fatalf("MarkProjectComplete: %v", err)
	}

	req := httptest.NewRequest("GET", "/archive/projects", nil)
	rec := httptest.NewRecorder()
	h.CompletedProjects(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if !strings.Contains(body, fmt.Sprintf(`id="project-%d"`, completedProject.ID)) {
		t.Fatalf("expected completed projects cards to include %q", completedProject.Name)
	}
	if strings.Contains(body, fmt.Sprintf(`id="project-%d"`, activeProject.ID)) {
		t.Fatalf("did not expect active project %q in completed projects cards", activeProject.Name)
	}
}

func TestUpdateTaskHandler_MovesToAnotherProject(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	p1 := &models.Project{Name: "Source", Type: "project"}
	p2 := &models.Project{Name: "Dest", Type: "project"}
	if err := s.CreateProject(ctx, p1); err != nil {
		t.Fatalf("CreateProject p1: %v", err)
	}
	if err := s.CreateProject(ctx, p2); err != nil {
		t.Fatalf("CreateProject p2: %v", err)
	}
	task := &models.Task{ProjectID: p1.ID, Description: "Moveable", Priority: "medium", Status: "todo"}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	form := url.Values{
		"description": {"Moveable"},
		"priority":    {"medium"},
		"status":      {"todo"},
		"project_id":  {strconv.FormatInt(p2.ID, 10)},
	}
	req := httptest.NewRequest("PUT", "/api/tasks/"+strconv.FormatInt(task.ID, 10), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(task.ID, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateTask(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	updated, _ := s.GetTask(ctx, task.ID)
	if updated.ProjectID != p2.ID {
		t.Errorf("expected ProjectID %d, got %d", p2.ID, updated.ProjectID)
	}
}

func TestUpdateTaskHandler_RejectsCompletedProject(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	active := &models.Project{Name: "Active", Type: "project"}
	completed := &models.Project{Name: "Completed", Type: "project"}
	if err := s.CreateProject(ctx, active); err != nil {
		t.Fatalf("CreateProject active: %v", err)
	}
	if err := s.CreateProject(ctx, completed); err != nil {
		t.Fatalf("CreateProject completed: %v", err)
	}
	if err := s.MarkProjectComplete(ctx, completed.ID); err != nil {
		t.Fatalf("MarkProjectComplete: %v", err)
	}
	task := &models.Task{ProjectID: active.ID, Description: "Task", Priority: "medium", Status: "todo"}
	if err := s.CreateTask(ctx, task); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	form := url.Values{
		"description": {"Task"},
		"priority":    {"medium"},
		"status":      {"todo"},
		"project_id":  {strconv.FormatInt(completed.ID, 10)},
	}
	req := httptest.NewRequest("PUT", "/api/tasks/"+strconv.FormatInt(task.ID, 10), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", strconv.FormatInt(task.ID, 10))
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.UpdateTask(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
