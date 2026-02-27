package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"mytasks/internal/models"
)

// SQLiteStore implements the Store interface using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

var sqliteDateLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05-07:00",
	"2006-01-02 15:04:05.999999999-07:00",
	"2006-01-02T15:04:05.999999999-07:00",
}

func parseSQLiteDate(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	for _, layout := range sqliteDateLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return &t, nil
		}
	}

	return nil, fmt.Errorf("invalid date format: %q", value)
}

// NewSQLiteStore creates a new SQLite store with the given database path.
func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dsn := dbPath + "?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &SQLiteStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return store, nil
}

func (s *SQLiteStore) migrate() error {
	return runMigrations(s.db)
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

	sortOrder := project.SortOrder
	if sortOrder <= 0 {
		sortOrder = -1
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO projects (name, description, type, target_date, completed, completed_at, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?,
			CASE WHEN ? > 0 THEN ? ELSE COALESCE((SELECT MAX(sort_order) + 1 FROM projects), 1) END,
			?, ?)
	`, project.Name, project.Description, project.Type, targetDate, false, nil, sortOrder, sortOrder, now, now)
	if err != nil {
		return fmt.Errorf("failed to create project: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	project.ID = id

	if err := s.db.QueryRowContext(ctx, `SELECT sort_order FROM projects WHERE id = ?`, id).Scan(&project.SortOrder); err != nil {
		return fmt.Errorf("failed to load project sort order: %w", err)
	}

	return nil
}

// GetProject retrieves a project by ID.
func (s *SQLiteStore) GetProject(ctx context.Context, id int64) (*models.Project, error) {
	project := &models.Project{}
	var targetDate sql.NullString
	var completedAt sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, description, type, target_date, completed, completed_at, sort_order, created_at, updated_at
		FROM projects WHERE id = ?
	`, id).Scan(
		&project.ID,
		&project.Name,
		&project.Description,
		&project.Type,
		&targetDate,
		&project.Completed,
		&completedAt,
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
		parsedDate, err := parseSQLiteDate(targetDate.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse project target_date: %w", err)
		}
		project.TargetDate = parsedDate
	}

	if completedAt.Valid {
		parsedDate, err := parseSQLiteDate(completedAt.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse project completed_at: %w", err)
		}
		project.CompletedAt = parsedDate
	}

	return project, nil
}

// ListProjects retrieves all projects ordered by sort_order.
func (s *SQLiteStore) ListProjects(ctx context.Context) ([]models.Project, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, description, type, target_date, completed, completed_at, sort_order, created_at, updated_at
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
		var completedAt sql.NullString

		err := rows.Scan(
			&project.ID,
			&project.Name,
			&project.Description,
			&project.Type,
			&targetDate,
			&project.Completed,
			&completedAt,
			&project.SortOrder,
			&project.CreatedAt,
			&project.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan project: %w", err)
		}

		if targetDate.Valid {
			parsedDate, err := parseSQLiteDate(targetDate.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse project target_date: %w", err)
			}
			project.TargetDate = parsedDate
		}

		if completedAt.Valid {
			parsedDate, err := parseSQLiteDate(completedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse project completed_at: %w", err)
			}
			project.CompletedAt = parsedDate
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

	var completedAt interface{}
	if project.Completed {
		if project.CompletedAt == nil {
			now := time.Now()
			project.CompletedAt = &now
		}
		completedAt = project.CompletedAt.Format("2006-01-02")
	}

	_, err := s.db.ExecContext(ctx, `
		UPDATE projects
		SET name = ?, description = ?, type = ?, target_date = ?, completed = ?, completed_at = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, project.Name, project.Description, project.Type, targetDate, project.Completed, completedAt, project.SortOrder, project.UpdatedAt, project.ID)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}

	return nil
}

// MarkProjectComplete marks a project as completed and records the completion date.
func (s *SQLiteStore) MarkProjectComplete(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE projects
		SET completed = TRUE,
		    completed_at = ?,
		    updated_at = ?
		WHERE id = ?
	`, now.Format("2006-01-02"), now, id)
	if err != nil {
		return fmt.Errorf("failed to mark project complete: %w", err)
	}

	return nil
}

// MarkProjectIncomplete marks a project as incomplete and clears completion date.
func (s *SQLiteStore) MarkProjectIncomplete(ctx context.Context, id int64) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE projects
		SET completed = FALSE,
		    completed_at = NULL,
		    updated_at = ?
		WHERE id = ?
	`, now, id)
	if err != nil {
		return fmt.Errorf("failed to mark project incomplete: %w", err)
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

	var completedAt interface{}
	if task.Completed {
		if task.CompletedAt == nil {
			t := now
			task.CompletedAt = &t
		}
		completedAt = task.CompletedAt.Format("2006-01-02")
	}

	sortOrder := task.SortOrder
	if sortOrder <= 0 {
		sortOrder = -1
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (project_id, description, notes, priority, due_date, completed, completed_at, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?,
			CASE WHEN ? > 0 THEN ? ELSE COALESCE((SELECT MAX(sort_order) + 1 FROM tasks WHERE project_id = ?), 1) END,
			?, ?)
	`, task.ProjectID, task.Description, task.Notes, task.Priority, dueDate, task.Completed, completedAt, sortOrder, sortOrder, task.ProjectID, now, now)
	if err != nil {
		return fmt.Errorf("failed to create task: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	task.ID = id

	if err := s.db.QueryRowContext(ctx, `SELECT sort_order FROM tasks WHERE id = ?`, id).Scan(&task.SortOrder); err != nil {
		return fmt.Errorf("failed to load task sort order: %w", err)
	}

	return nil
}

// GetTask retrieves a task by ID.
func (s *SQLiteStore) GetTask(ctx context.Context, id int64) (*models.Task, error) {
	task := &models.Task{}
	var dueDate sql.NullString
	var completedAt sql.NullString

	err := s.db.QueryRowContext(ctx, `
		SELECT id, project_id, description, notes, priority, due_date, completed, completed_at, sort_order, created_at, updated_at
		FROM tasks WHERE id = ?
	`, id).Scan(
		&task.ID,
		&task.ProjectID,
		&task.Description,
		&task.Notes,
		&task.Priority,
		&dueDate,
		&task.Completed,
		&completedAt,
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
		parsedDate, err := parseSQLiteDate(dueDate.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse task due_date: %w", err)
		}
		task.DueDate = parsedDate
	}

	if completedAt.Valid {
		parsedDate, err := parseSQLiteDate(completedAt.String)
		if err != nil {
			return nil, fmt.Errorf("failed to parse task completed_at: %w", err)
		}
		task.CompletedAt = parsedDate
	}

	return task, nil
}

// ListTasksByProject retrieves tasks for a project ordered by sort_order.
// If limit is 0, all tasks are returned.
func (s *SQLiteStore) ListTasksByProject(ctx context.Context, projectID int64, limit int) ([]models.Task, error) {
	query := `
		SELECT id, project_id, description, notes, priority, due_date, completed, completed_at, sort_order, created_at, updated_at
		FROM tasks WHERE project_id = ? ORDER BY sort_order ASC
	`
	args := []interface{}{projectID}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var dueDate sql.NullString
		var completedAt sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.Description,
			&task.Notes,
			&task.Priority,
			&dueDate,
			&task.Completed,
			&completedAt,
			&task.SortOrder,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if dueDate.Valid {
			parsedDate, err := parseSQLiteDate(dueDate.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse task due_date: %w", err)
			}
			task.DueDate = parsedDate
		}

		if completedAt.Valid {
			parsedDate, err := parseSQLiteDate(completedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse task completed_at: %w", err)
			}
			task.CompletedAt = parsedDate
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// ListTasksByProjectFiltered retrieves tasks for a project filtered by completion status.
// If limit is 0, all matching tasks are returned.
func (s *SQLiteStore) ListTasksByProjectFiltered(ctx context.Context, projectID int64, completed bool, limit int) ([]models.Task, error) {
	query := `
		SELECT id, project_id, description, notes, priority, due_date, completed, completed_at, sort_order, created_at, updated_at
		FROM tasks WHERE project_id = ? AND completed = ? ORDER BY sort_order ASC
	`
	args := []interface{}{projectID, completed}
	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var dueDate sql.NullString
		var completedAt sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.Description,
			&task.Notes,
			&task.Priority,
			&dueDate,
			&task.Completed,
			&completedAt,
			&task.SortOrder,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan task: %w", err)
		}

		if dueDate.Valid {
			parsedDate, err := parseSQLiteDate(dueDate.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse task due_date: %w", err)
			}
			task.DueDate = parsedDate
		}

		if completedAt.Valid {
			parsedDate, err := parseSQLiteDate(completedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse task completed_at: %w", err)
			}
			task.CompletedAt = parsedDate
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// ListTasksByProjectCompletedBetween retrieves completed tasks for a project within a completion date range.
// When from/to are nil they are not applied as filters. If limit is 0, all matching tasks are returned.
func (s *SQLiteStore) ListTasksByProjectCompletedBetween(ctx context.Context, projectID int64, from, to *time.Time, limit int) ([]models.Task, error) {
	query := `
		SELECT id, project_id, description, notes, priority, due_date, completed, completed_at, sort_order, created_at, updated_at
		FROM tasks WHERE project_id = ? AND completed = TRUE AND completed_at IS NOT NULL
	`
	args := []interface{}{projectID}

	if from != nil {
		query += ` AND completed_at >= ?`
		args = append(args, from.Format("2006-01-02"))
	}

	if to != nil {
		query += ` AND completed_at <= ?`
		args = append(args, to.Format("2006-01-02"))
	}

	query += ` ORDER BY completed_at DESC, sort_order ASC`

	if limit > 0 {
		query += " LIMIT ?"
		args = append(args, limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list completed tasks by range: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var task models.Task
		var dueDate sql.NullString
		var completedAt sql.NullString

		err := rows.Scan(
			&task.ID,
			&task.ProjectID,
			&task.Description,
			&task.Notes,
			&task.Priority,
			&dueDate,
			&task.Completed,
			&completedAt,
			&task.SortOrder,
			&task.CreatedAt,
			&task.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan completed task: %w", err)
		}

		if dueDate.Valid {
			parsedDate, err := parseSQLiteDate(dueDate.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse task due_date: %w", err)
			}
			task.DueDate = parsedDate
		}

		if completedAt.Valid {
			parsedDate, err := parseSQLiteDate(completedAt.String)
			if err != nil {
				return nil, fmt.Errorf("failed to parse task completed_at: %w", err)
			}
			task.CompletedAt = parsedDate
		}

		tasks = append(tasks, task)
	}

	return tasks, rows.Err()
}

// UpdateTask updates an existing task.
func (s *SQLiteStore) UpdateTask(ctx context.Context, task *models.Task) error {
	task.UpdatedAt = time.Now()

	var wasCompleted bool
	var existingCompletedAt sql.NullString
	err := s.db.QueryRowContext(ctx, `SELECT completed, completed_at FROM tasks WHERE id = ?`, task.ID).Scan(&wasCompleted, &existingCompletedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("task not found: %d", task.ID)
		}
		return fmt.Errorf("failed to load task completion state: %w", err)
	}

	var dueDate interface{}
	if task.DueDate != nil {
		dueDate = task.DueDate.Format("2006-01-02")
	}

	var completedAt interface{}
	if task.Completed {
		switch {
		case !wasCompleted:
			now := time.Now()
			task.CompletedAt = &now
			completedAt = now.Format("2006-01-02")
		case task.CompletedAt != nil:
			completedAt = task.CompletedAt.Format("2006-01-02")
		case existingCompletedAt.Valid:
			completedAt = existingCompletedAt.String
		}
	} else {
		task.CompletedAt = nil
	}

	_, err = s.db.ExecContext(ctx, `
		UPDATE tasks
		SET description = ?, notes = ?, priority = ?, due_date = ?, completed = ?, completed_at = ?, sort_order = ?, updated_at = ?
		WHERE id = ?
	`, task.Description, task.Notes, task.Priority, dueDate, task.Completed, completedAt, task.SortOrder, task.UpdatedAt, task.ID)
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
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE tasks
		SET completed = NOT completed,
		    completed_at = CASE
		        WHEN completed = 0 THEN ?
		        ELSE NULL
		    END,
		    updated_at = ?
		WHERE id = ?
	`, now.Format("2006-01-02"), now, id)
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
