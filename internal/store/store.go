package store

import (
	"context"

	"mytasks/internal/models"
)

// Store defines the interface for data persistence operations.
type Store interface {
	// Project operations
	CreateProject(ctx context.Context, project *models.Project) error
	GetProject(ctx context.Context, id int64) (*models.Project, error)
	ListProjects(ctx context.Context) ([]models.Project, error)
	UpdateProject(ctx context.Context, project *models.Project) error
	DeleteProject(ctx context.Context, id int64) error
	ReorderProjects(ctx context.Context, ids []int64) error

	// Task operations
	CreateTask(ctx context.Context, task *models.Task) error
	GetTask(ctx context.Context, id int64) (*models.Task, error)
	ListTasksByProject(ctx context.Context, projectID int64, limit int) ([]models.Task, error)
	ListTasksByProjectFiltered(ctx context.Context, projectID int64, completed bool, limit int) ([]models.Task, error)
	UpdateTask(ctx context.Context, task *models.Task) error
	DeleteTask(ctx context.Context, id int64) error
	ToggleTaskComplete(ctx context.Context, id int64) error
	ReorderTasks(ctx context.Context, projectID int64, ids []int64) error

	// Lifecycle
	Close() error
}
