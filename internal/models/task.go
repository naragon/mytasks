package models

import (
	"errors"
	"strings"
	"time"
)

// Task represents a single task within a project.
type Task struct {
	ID          int64      `json:"id"`
	ProjectID   int64      `json:"project_id"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"` // "high", "medium", "low"
	DueDate     *time.Time `json:"due_date,omitempty"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// Validate checks that the task has valid field values.
func (t *Task) Validate() error {
	if strings.TrimSpace(t.Description) == "" {
		return errors.New("description is required")
	}

	if t.ProjectID == 0 {
		return errors.New("project_id is required")
	}

	if t.Priority != "high" && t.Priority != "medium" && t.Priority != "low" {
		return errors.New("priority must be 'high', 'medium', or 'low'")
	}

	return nil
}

// IsOverdue returns true if the task has a due date that has passed and is not completed.
func (t *Task) IsOverdue() bool {
	if t.Completed || t.DueDate == nil {
		return false
	}
	return t.DueDate.Before(time.Now())
}

// PriorityOrder returns a numeric value for sorting by priority.
// Lower numbers indicate higher priority.
func (t *Task) PriorityOrder() int {
	switch t.Priority {
	case "high":
		return 1
	case "medium":
		return 2
	case "low":
		return 3
	default:
		return 99
	}
}
