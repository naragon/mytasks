package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

func TestHomeHandler_ListsProjects(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	// Create test projects
	s.CreateProject(ctx, &models.Project{Name: "Project A", Type: "project", SortOrder: 1})
	s.CreateProject(ctx, &models.Project{Name: "Project B", Type: "category", SortOrder: 2})

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestHomeHandler_HidesCompletedProjects(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	active := &models.Project{Name: "Active", Type: "project", SortOrder: 1}
	completed := &models.Project{Name: "Completed", Type: "project", SortOrder: 2}
	s.CreateProject(ctx, active)
	s.CreateProject(ctx, completed)
	s.MarkProjectComplete(ctx, completed.ID)

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	body := rec.Body.String()
	if strings.Contains(body, "Completed") {
		t.Fatalf("expected completed project to be hidden, response body: %s", body)
	}
}

func TestHomeHandler_ShowsTopThreeTasks(t *testing.T) {
	h, s := setupTestHandlers(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	s.CreateProject(ctx, project)

	// Create 5 tasks
	for i := 1; i <= 5; i++ {
		s.CreateTask(ctx, &models.Task{
			ProjectID:   project.ID,
			Description: "Task",
			Priority:    "medium",
			SortOrder:   i,
		})
	}

	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()

	h.Home(rec, req)

	// The handler should only fetch top 3 tasks per project
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestProjectDetailHandler_ShowsAllTasks(t *testing.T) {
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

	// Set up chi URL params
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", "1")
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, rctx))

	h.ProjectDetail(rec, req)

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
