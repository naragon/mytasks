package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"mytasks/internal/models"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

// NewSQLiteStore creates a new SQLite store with the given database path.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	schema := `
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

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateProject creates a new project in the database.
func (s *SQLiteStore) CreateProject(ctx context.Context, project *models.Project) error {
	now := time.Now()
	project.CreatedAt = now
	project.UpdatedAt = now

	var targetDate interface{}
	if project.TargetDate != nil {
		targetDate = project.TargetDate.Format("2006-01-02")
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO projects (name, description, type, target_date, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, project.Name, project.Description, project.Type, targetDate, project.SortOrder, now, now)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	project.ID = id

	return nil
}

// GetProject retrieves a project by ID.
func (s *SQLiteStore) GetProject(ctx context.Context, id int64) (*models.Project, error) {
	project := &models.Project{}
	var targetDate sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, type, target_date, sort_order, created_at, updated_at
		FROM projects WHERE id = ?
	`, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.Type,
		&targetDate,
		&project.SortOrder,
		&project.CreatedAt,
		&project.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("project not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	if targetDate.Valid {
		t, _ := time.Parse("2006-01-02", targetDate.String)
		project.TargetDate = &t
	}

	return project, nil
}

// ListProjects retrieves all projects ordered by sort_order.
func (s *SQLiteStore) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, type, target_date, sort_order, created_at, updated_at
		FROM projects ORDER BY sort_order ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var project models.Project
		var targetDate sql.NullString

		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.Type,
			&targetDate,
			&project.SortOrder,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		if targetDate.Valid {
			t, _ := time.Parse("2006-01-02", targetDate.String)
			project.TargetDate = &t
		}

		projects = append(projects, project)
	}

	return projects, rows.Err()
}

// UpdateProject updates an existing project.
func (s *SQLiteStore) UpdateProject(ctx context.Context, project *models.Project) error {
	project.UpdatedAt = time.Now()

	var targetDate interface{}
	if project.TargetDate != nil {
		targetDate = project.TargetDate.Format("2006-01-02")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE projects
		SET name = ?, description = ?, type = ?, target_date = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, project.Name, project.Description, project.Type, targetDate, project.SortOrder, project.UpdatedAt, project.ID)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// DeleteProject deletes a project and its associated tasks.
func (s *SQLiteStore) DeleteProject(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

// ReorderProjects updates the sort_order of projects based on the given order of IDs.
func (s *SQLiteStore) ReorderProjects(ctx context.Context, ids []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE projects SET sort_order = ? WHERE id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for i, id := range ids {
		_, err := stmt.ExecContext(ctx, i+1, id)
		if err != nil {
			return fmt.Errorf("failed to update sort order: %w", err)
		}
	}

	return tx.Commit()
}

// CreateTask creates a new task in the database.
func (s *SQLiteStore) CreateTask(ctx context.Context, task *models.Task) error {
	now := time.Now()
	task.CreatedAt = now
	task.UpdatedAt = now

	var dueDate interface{}
	if task.DueDate != nil {
		dueDate = task.DueDate.Format("2006-01-02")
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (project_id, description, priority, due_date, completed, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, task.ProjectID, task.Description, task.Priority, dueDate, task.Completed, task.SortOrder, now, now)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	task.ID = id

	return nil
}

// GetTask retrieves a task by ID.
func (s *SQLiteStore) GetTask(ctx context.Context, id int64) (*models.Task, error) {
	task := &models.Task{}
	var dueDate sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, description, priority, due_date, completed, sort_order, created_at, updated_at
		FROM tasks WHERE id = ?
	`, id).Scan(
		&task.ID,
		&task.ProjectID,
		&task.Description,
		&task.Priority,
		&dueDate,
		&task.Completed,
		&task.SortOrder,
		&task.CreatedAt,
		&task.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("task not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if dueDate.Valid {
		t, _ := time.Parse("2006-01-02", dueDate.String)
		task.DueDate = &t
	}

	return task, nil
}

// ListTasksByProject retrieves tasks for a project ordered by sort_order.
// If limit is 0, all tasks are returned.
func (s *SQLiteStore) ListTasksByProject(ctx context.Context, projectID int64, limit int) ([]models.Task, error) {
	query := `
		SELECT id, project_id, description, priority, due_date, completed, sort_order, created_at, updated_at
		FROM tasks WHERE project_id = ? ORDER BY sort_order ASC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var dueDate sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.Description,
			&task.Priority,
			&dueDate,
			&task.Completed,
			&task.SortOrder,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if dueDate.Valid {
			t, _ := time.Parse("2006-01-02", dueDate.String)
			task.DueDate = &t
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// ListTasksByProjectFiltered retrieves tasks for a project filtered by completion status.
// If limit is 0, all matching tasks are returned.
func (s *SQLiteStore) ListTasksByProjectFiltered(ctx context.Context, projectID int64, completed bool, limit int) ([]models.Task, error) {
	query := `
		SELECT id, project_id, description, priority, due_date, completed, sort_order, created_at, updated_at
		FROM tasks WHERE project_id = ? AND completed = ? ORDER BY sort_order ASC
	`
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.db.QueryContext(ctx, query, projectID, completed)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var dueDate sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.Description,
			&task.Priority,
			&dueDate,
			&task.Completed,
			&task.SortOrder,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if dueDate.Valid {
			t, _ := time.Parse("2006-01-02", dueDate.String)
			task.DueDate = &t
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// UpdateTask updates an existing task.
func (s *SQLiteStore) UpdateTask(ctx context.Context, task *models.Task) error {
	task.UpdatedAt = time.Now()

	var dueDate interface{}
	if task.DueDate != nil {
		dueDate = task.DueDate.Format("2006-01-02")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET description = ?, priority = ?, due_date = ?, completed = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, task.Description, task.Priority, dueDate, task.Completed, task.SortOrder, task.UpdatedAt, task.ID)
	if err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}

	return nil
}

// DeleteTask deletes a task by ID.
func (s *SQLiteStore) DeleteTask(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("failed to delete task: %w", err)
	}
	return nil
}

// ToggleTaskComplete toggles the completed status of a task.
func (s *SQLiteStore) ToggleTaskComplete(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks SET completed = NOT completed, updated_at = ? WHERE id = ?
	`, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to toggle task complete: %w", err)
	}
	return nil
}

// ReorderTasks updates the sort_order of tasks within a project.
func (s *SQLiteStore) ReorderTasks(ctx context.Context, projectID int64, ids []int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `UPDATE tasks SET sort_order = ? WHERE id = ? AND project_id = ?`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for i, id := range ids {
		_, err := stmt.ExecContext(ctx, i+1, id, projectID)
		if err != nil {
			return fmt.Errorf("failed to update sort order: %w", err)
		}
	}

	return tx.Commit()
}
