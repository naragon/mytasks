package store

import (
	"context"
	"time"

	"mytasks/internal/models"
)

// Store defines the interface for data persistence operations.
type Store interface {
	// Project operations
	CreateProject(ctx context.Context, project *models.Project) error
	GetProject(ctx context.Context, id int64) (*models.Project, error)
	ListProjects(ctx context.Context) ([]models.Project, error)
	ListActiveProjects(ctx context.Context) ([]models.Project, error)
	ListCompletedProjects(ctx context.Context) ([]models.Project, error)
	UpdateProject(ctx context.Context, project *models.Project) error
	MarkProjectComplete(ctx context.Context, id int64) error
	MarkProjectIncomplete(ctx context.Context, id int64) error
	DeleteProject(ctx context.Context, id int64) error
	ReorderProjects(ctx context.Context, ids []int64) error

	// Task operations
	CreateTask(ctx context.Context, task *models.Task) error
	GetTask(ctx context.Context, id int64) (*models.Task, error)
	ListTasksByProject(ctx context.Context, projectID int64, limit int) ([]models.Task, error)
	ListTasksByProjectFiltered(ctx context.Context, projectID int64, completed bool, limit int) ([]models.Task, error)
	ListTasksByProjectCompletedBetween(ctx context.Context, projectID int64, from, to *time.Time, limit int) ([]models.Task, error)
	ListTasksByProjectAndStatus(ctx context.Context, projectID int64, status string) ([]models.Task, error)
	ListRecentDoneTasks(ctx context.Context, projectID int64, since time.Time) ([]models.Task, error)
	ListOldDoneTasks(ctx context.Context, projectID int64, before time.Time) ([]models.Task, error)
	ListActiveProjectsWithOldDoneTasks(ctx context.Context, before time.Time) ([]models.Project, error)
	ListUpcomingTasks(ctx context.Context, days int) ([]models.Task, error)
	UpdateTask(ctx context.Context, task *models.Task) error
	DeleteTask(ctx context.Context, id int64) error
	ToggleTaskComplete(ctx context.Context, id int64) error
	MoveTaskToStatus(ctx context.Context, taskID int64, newStatus string, newSortOrder int) error
	ReorderTasks(ctx context.Context, projectID int64, ids []int64) error
	ReorderTasksInStatus(ctx context.Context, projectID int64, status string, ids []int64) error

	// Lifecycle
	Close() error
}
