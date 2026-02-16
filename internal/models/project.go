package models

import (
	"errors"
	"strings"
	"time"
)

// Project represents a project or category for organizing tasks.
type Project struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Type        string     `json:"type"` // "project" or "category"
	TargetDate  *time.Time `json:"target_date,omitempty"`
	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	SortOrder   int        `json:"sort_order"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ViewTab     string     `json:"-"`

	// Tasks holds the tasks for this project (populated by queries)
	Tasks []Task `json:"tasks,omitempty"`
}

// Validate checks that the project has valid field values.
func (p *Project) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("name is required")
	}

	if p.Type != "project" && p.Type != "category" {
		return errors.New("type must be 'project' or 'category'")
	}

	if p.Type == "category" && p.TargetDate != nil {
		return errors.New("category cannot have a target date")
	}

	return nil
}

// IsCategory returns true if this project is a category (ongoing, no target date).
func (p *Project) IsCategory() bool {
	return p.Type == "category"
}

// IsOverdue returns true if the project has a target date that has passed.
func (p *Project) IsOverdue() bool {
	if p.TargetDate == nil {
		return false
	}
	return p.TargetDate.Before(time.Now())
}
