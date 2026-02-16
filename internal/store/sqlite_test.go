package store

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mytasks/internal/models"

	_ "github.com/mattn/go-sqlite3"
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
	if got.CompletedAt != nil {
		t.Error("expected completed_at to be cleared when task is incomplete")
	}
}

func TestToggleTaskComplete_SetsCompletedAt(t *testing.T) {
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

	err := store.ToggleTaskComplete(ctx, task.ID)
	if err != nil {
		t.Fatalf("ToggleTaskComplete failed: %v", err)
	}

	got, _ := store.GetTask(ctx, task.ID)
	if !got.Completed {
		t.Fatal("expected task to be completed")
	}
	if got.CompletedAt == nil {
		t.Fatal("expected completed_at to be set")
	}
	if got.CompletedAt.Format("2006-01-02") != time.Now().Format("2006-01-02") {
		t.Fatalf("expected completed_at to be today, got %s", got.CompletedAt.Format("2006-01-02"))
	}
}

func TestListTasksByProjectCompletedBetween(t *testing.T) {
	store := setupTestDB(t)
	ctx := context.Background()

	project := &models.Project{Name: "Test", Type: "project"}
	store.CreateProject(ctx, project)

	first := &models.Task{ProjectID: project.ID, Description: "First", Priority: "high"}
	second := &models.Task{ProjectID: project.ID, Description: "Second", Priority: "medium"}
	store.CreateTask(ctx, first)
	store.CreateTask(ctx, second)

	if err := store.ToggleTaskComplete(ctx, first.ID); err != nil {
		t.Fatalf("ToggleTaskComplete(first) failed: %v", err)
	}
	if err := store.ToggleTaskComplete(ctx, second.ID); err != nil {
		t.Fatalf("ToggleTaskComplete(second) failed: %v", err)
	}

	if _, err := store.db.ExecContext(ctx, `UPDATE tasks SET completed_at = ? WHERE id = ?`, "2025-01-10", first.ID); err != nil {
		t.Fatalf("failed to set first completed_at: %v", err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE tasks SET completed_at = ? WHERE id = ?`, "2025-02-05", second.ID); err != nil {
		t.Fatalf("failed to set second completed_at: %v", err)
	}

	from := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)

	tasks, err := store.ListTasksByProjectCompletedBetween(ctx, project.ID, &from, &to, 0)
	if err != nil {
		t.Fatalf("ListTasksByProjectCompletedBetween failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Fatalf("expected 1 task in range, got %d", len(tasks))
	}
	if tasks[0].Description != "Second" {
		t.Fatalf("expected task Second, got %s", tasks[0].Description)
	}
	if tasks[0].CompletedAt == nil || tasks[0].CompletedAt.Format("2006-01-02") != "2025-02-05" {
		t.Fatalf("expected completed_at 2025-02-05, got %#v", tasks[0].CompletedAt)
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

func TestNewSQLiteStore_MigratesLegacyDatabaseAndPreservesData(t *testing.T) {
	tempDir := t.TempDir()
	dbPath := filepath.Join(tempDir, "legacy.db")

	legacyDB, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("failed to open legacy db: %v", err)
	}
	t.Cleanup(func() { legacyDB.Close() })

	legacySchema := `
	CREATE TABLE IF NOT EXISTS projects (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT DEFAULT '',
		type TEXT NOT NULL CHECK(type IN ('project', 'category')),
		target_date DATE,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tasks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_id INTEGER NOT NULL,
		description TEXT NOT NULL,
		priority TEXT NOT NULL CHECK(priority IN ('high', 'medium', 'low')),
		due_date DATE,
		completed BOOLEAN DEFAULT FALSE,
		sort_order INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_tasks_project_id ON tasks(project_id);
	CREATE INDEX IF NOT EXISTS idx_projects_sort_order ON projects(sort_order);
	CREATE INDEX IF NOT EXISTS idx_tasks_sort_order ON tasks(sort_order);
	`

	if _, err := legacyDB.Exec(legacySchema); err != nil {
		t.Fatalf("failed to create legacy schema: %v", err)
	}

	if _, err := legacyDB.Exec(`INSERT INTO projects (id, name, type, sort_order) VALUES (1, 'Legacy Project', 'project', 1)`); err != nil {
		t.Fatalf("failed to seed legacy project: %v", err)
	}
	if _, err := legacyDB.Exec(`INSERT INTO tasks (id, project_id, description, priority, completed, sort_order) VALUES (1, 1, 'Legacy Task', 'medium', 0, 1)`); err != nil {
		t.Fatalf("failed to seed legacy task: %v", err)
	}

	if err := legacyDB.Close(); err != nil {
		t.Fatalf("failed to close legacy db before reopening: %v", err)
	}

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open sqlite store with migrations: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	ctx := context.Background()
	project, err := store.GetProject(ctx, 1)
	if err != nil {
		t.Fatalf("expected legacy project to persist: %v", err)
	}
	if project.Name != "Legacy Project" {
		t.Fatalf("expected Legacy Project, got %s", project.Name)
	}

	task, err := store.GetTask(ctx, 1)
	if err != nil {
		t.Fatalf("expected legacy task to persist: %v", err)
	}
	if task.Description != "Legacy Task" {
		t.Fatalf("expected Legacy Task, got %s", task.Description)
	}

	var hasCompletedAt bool
	rows, err := store.db.Query(`PRAGMA table_info(tasks)`)
	if err != nil {
		t.Fatalf("failed to inspect tasks table: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			typeName   string
			notNull    int
			defaultVal interface{}
			pk         int
		)
		if err := rows.Scan(&cid, &name, &typeName, &notNull, &defaultVal, &pk); err != nil {
			t.Fatalf("failed to scan pragma row: %v", err)
		}
		if name == "completed_at" {
			hasCompletedAt = true
		}
	}

	if !hasCompletedAt {
		t.Fatal("expected completed_at column to exist after migrations")
	}

	var migrationCount int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("failed to count schema migrations: %v", err)
	}
	if migrationCount < 2 {
		t.Fatalf("expected at least 2 applied migrations, got %d", migrationCount)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
}
