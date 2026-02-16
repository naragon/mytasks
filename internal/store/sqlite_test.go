package store

import (
	"context"
	"testing"
	"time"

	"mytasks/internal/models"
)

func setupTestDB(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestCreateProject(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{
		Name:        "Test Project",
		Description: "A test project",
		Type:        "project",
	}

	err := store.CreateProject(ctx, project)
	if err != nil {
		t.Fatalf("CreateProject failed: %v", err)
	}

	if project.ID == 0 {
		t.Error("expected project ID to be set")
	}
	if project.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
	if project.UpdatedAt.IsZero() {
		t.Error("expected updated_at to be set")
	}
}

func TestGetProject(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	targetDate := time.Now().AddDate(0, 1, 0).Truncate(24 * time.Hour)
	project := &models.Project{
		Name:        "Test Project",
		Description: "A test project",
		Type:        "project",
		TargetDate:  &targetDate,
	}
	store.CreateProject(ctx, project)

	got, err := store.GetProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("GetProject failed: %v", err)
	}

	if got.Name != project.Name {
		t.Errorf("expected name %q, got %q", project.Name, got.Name)
	}
	if got.Description != project.Description {
		t.Errorf("expected description %q, got %q", project.Description, got.Description)
	}
	if got.Type != project.Type {
		t.Errorf("expected type %q, got %q", project.Type, got.Type)
	}
	if got.TargetDate == nil {
		t.Error("expected target date to be set")
	}
}

func TestGetProject_NotFound(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	_, err := store.GetProject(ctx, 999)
	if err == nil {
		t.Error("expected error for non-existent project")
	}
}

func TestListProjects_OrderedBySortOrder(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	// Create projects in non-sequential order
	projects := []*models.Project{
		{Name: "Third", Type: "project", SortOrder: 3},
		{Name: "First", Type: "project", SortOrder: 1},
		{Name: "Second", Type: "project", SortOrder: 2},
	}
	for _, p := range projects {
		store.CreateProject(ctx, p)
	}

	got, err := store.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(got) != 3 {
		t.Fatalf("expected 3 projects, got %d", len(got))
	}

	expectedOrder := []string{"First", "Second", "Third"}
	for i, name := range expectedOrder {
		if got[i].Name != name {
			t.Errorf("position %d: expected %q, got %q", i, name, got[i].Name)
		}
	}
}

func TestUpdateProject(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{
		Name: "Original",
		Type: "project",
	}
	store.CreateProject(ctx, project)

	project.Name = "Updated"
	project.Description = "New description"
	err := store.UpdateProject(ctx, project)
	if err != nil {
		t.Fatalf("UpdateProject failed: %v", err)
	}

	got, _ := store.GetProject(ctx, project.ID)
	if got.Name != "Updated" {
		t.Errorf("expected name %q, got %q", "Updated", got.Name)
	}
	if got.Description != "New description" {
		t.Errorf("expected description %q, got %q", "New description", got.Description)
	}
}

func TestDeleteProject_CascadesTasks(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	task := &models.Task{
		ProjectID:   project.ID,
		Description: "Test task",
		Priority:    "medium",
	}
	store.CreateTask(ctx, task)

	err := store.DeleteProject(ctx, project.ID)
	if err != nil {
		t.Fatalf("DeleteProject failed: %v", err)
	}

	_, err = store.GetProject(ctx, project.ID)
	if err == nil {
		t.Error("expected project to be deleted")
	}

	_, err = store.GetTask(ctx, task.ID)
	if err == nil {
		t.Error("expected task to be deleted (cascade)")
	}
}

func TestReorderProjects(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	p1 := &models.Project{Name: "A", Type: "project"}
	p2 := &models.Project{Name: "B", Type: "project"}
	p3 := &models.Project{Name: "C", Type: "project"}
	store.CreateProject(ctx, p1)
	store.CreateProject(ctx, p2)
	store.CreateProject(ctx, p3)

	// Reorder to: C, A, B
	err := store.ReorderProjects(ctx, []int64{p3.ID, p1.ID, p2.ID})
	if err != nil {
		t.Fatalf("ReorderProjects failed: %v", err)
	}

	got, _ := store.ListProjects(ctx)
	expectedOrder := []string{"C", "A", "B"}
	for i, name := range expectedOrder {
		if got[i].Name != name {
			t.Errorf("position %d: expected %q, got %q", i, name, got[i].Name)
		}
	}
}

func TestCreateTask(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	task := &models.Task{
		ProjectID:   project.ID,
		Description: "Test task",
		Priority:    "high",
	}

	err := store.CreateTask(ctx, task)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.ID == 0 {
		t.Error("expected task ID to be set")
	}
	if task.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
}

func TestGetTask(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	dueDate := time.Now().AddDate(0, 0, 7).Truncate(24 * time.Hour)
	task := &models.Task{
		ProjectID:   project.ID,
		Description: "Test task",
		Priority:    "high",
		DueDate:     &dueDate,
	}
	store.CreateTask(ctx, task)

	got, err := store.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if got.Description != task.Description {
		t.Errorf("expected description %q, got %q", task.Description, got.Description)
	}
	if got.Priority != task.Priority {
		t.Errorf("expected priority %q, got %q", task.Priority, got.Priority)
	}
	if got.DueDate == nil {
		t.Error("expected due date to be set")
	}
}

func TestListTasksByProject_OrderedBySortOrder(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	tasks := []*models.Task{
		{ProjectID: project.ID, Description: "Third", Priority: "low", SortOrder: 3},
		{ProjectID: project.ID, Description: "First", Priority: "high", SortOrder: 1},
		{ProjectID: project.ID, Description: "Second", Priority: "medium", SortOrder: 2},
	}
	for _, task := range tasks {
		store.CreateTask(ctx, task)
	}

	got, err := store.ListTasksByProject(ctx, project.ID, 0)
	if err != nil {
		t.Fatalf("ListTasksByProject failed: %v", err)
	}

	expectedOrder := []string{"First", "Second", "Third"}
	for i, desc := range expectedOrder {
		if got[i].Description != desc {
			t.Errorf("position %d: expected %q, got %q", i, desc, got[i].Description)
		}
	}
}

func TestListTasksByProject_LimitThree(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	for i := 1; i <= 5; i++ {
		task := &models.Task{
			ProjectID:   project.ID,
			Description: "Task",
			Priority:    "medium",
			SortOrder:   i,
		}
		store.CreateTask(ctx, task)
	}

	got, err := store.ListTasksByProject(ctx, project.ID, 3)
	if err != nil {
		t.Fatalf("ListTasksByProject failed: %v", err)
	}

	if len(got) != 3 {
		t.Errorf("expected 3 tasks, got %d", len(got))
	}
}

func TestUpdateTask(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	task := &models.Task{
		ProjectID:   project.ID,
		Description: "Original",
		Priority:    "low",
	}
	store.CreateTask(ctx, task)

	task.Description = "Updated"
	task.Priority = "high"
	err := store.UpdateTask(ctx, task)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	got, _ := store.GetTask(ctx, task.ID)
	if got.Description != "Updated" {
		t.Errorf("expected description %q, got %q", "Updated", got.Description)
	}
	if got.Priority != "high" {
		t.Errorf("expected priority %q, got %q", "high", got.Priority)
	}
}

func TestDeleteTask(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	task := &models.Task{
		ProjectID:   project.ID,
		Description: "Test",
		Priority:    "medium",
	}
	store.CreateTask(ctx, task)

	err := store.DeleteTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	_, err = store.GetTask(ctx, task.ID)
	if err == nil {
		t.Error("expected task to be deleted")
	}
}

func TestToggleTaskComplete(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	task := &models.Task{
		ProjectID:   project.ID,
		Description: "Test",
		Priority:    "medium",
		Completed:   false,
	}
	store.CreateTask(ctx, task)

	// Toggle to complete
	err := store.ToggleTaskComplete(ctx, task.ID)
	if err != nil {
		t.Fatalf("ToggleTaskComplete failed: %v", err)
	}

	got, _ := store.GetTask(ctx, task.ID)
	if !got.Completed {
		t.Error("expected task to be completed")
	}

	// Toggle back to incomplete
	err = store.ToggleTaskComplete(ctx, task.ID)
	if err != nil {
		t.Fatalf("ToggleTaskComplete failed: %v", err)
	}

	got, _ = store.GetTask(ctx, task.ID)
	if got.Completed {
		t.Error("expected task to be incomplete")
	}
}

func TestReorderTasks(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	t1 := &models.Task{ProjectID: project.ID, Description: "A", Priority: "medium"}
	t2 := &models.Task{ProjectID: project.ID, Description: "B", Priority: "medium"}
	t3 := &models.Task{ProjectID: project.ID, Description: "C", Priority: "medium"}
	store.CreateTask(ctx, t1)
	store.CreateTask(ctx, t2)
	store.CreateTask(ctx, t3)

	// Reorder to: C, A, B
	err := store.ReorderTasks(ctx, project.ID, []int64{t3.ID, t1.ID, t2.ID})
	if err != nil {
		t.Fatalf("ReorderTasks failed: %v", err)
	}

	got, _ := store.ListTasksByProject(ctx, project.ID, 0)
	expectedOrder := []string{"C", "A", "B"}
	for i, desc := range expectedOrder {
		if got[i].Description != desc {
			t.Errorf("position %d: expected %q, got %q", i, desc, got[i].Description)
		}
	}
}
