package models

import (
	"strings"
	"testing"
	"time"
)

func TestTaskValidation_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty description should fail",
			task:    Task{Description: "", ProjectID: 1, Priority: "medium"},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name:    "whitespace description should fail",
			task:    Task{Description: "   ", ProjectID: 1, Priority: "medium"},
			wantErr: true,
			errMsg:  "description is required",
		},
		{
			name:    "zero project ID should fail",
			task:    Task{Description: "Test task", ProjectID: 0, Priority: "medium"},
			wantErr: true,
			errMsg:  "project_id is required",
		},
		{
			name:    "valid task should pass",
			task:    Task{Description: "Test task", ProjectID: 1, Priority: "medium"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestTaskValidation_PriorityValues(t *testing.T) {
	tests := []struct {
		name    string
		task    Task
		wantErr bool
		errMsg  string
	}{
		{
			name:    "high priority is valid",
			task:    Task{Description: "Test", ProjectID: 1, Priority: "high"},
			wantErr: false,
		},
		{
			name:    "medium priority is valid",
			task:    Task{Description: "Test", ProjectID: 1, Priority: "medium"},
			wantErr: false,
		},
		{
			name:    "low priority is valid",
			task:    Task{Description: "Test", ProjectID: 1, Priority: "low"},
			wantErr: false,
		},
		{
			name:    "empty priority should fail",
			task:    Task{Description: "Test", ProjectID: 1, Priority: ""},
			wantErr: true,
			errMsg:  "priority must be 'high', 'medium', or 'low'",
		},
		{
			name:    "invalid priority should fail",
			task:    Task{Description: "Test", ProjectID: 1, Priority: "urgent"},
			wantErr: true,
			errMsg:  "priority must be 'high', 'medium', or 'low'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.task.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				} else if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestTaskValidation_NotesLength(t *testing.T) {
	validNotes := strings.Repeat("a", 255)
	invalidNotes := strings.Repeat("a", 256)

	valid := Task{Description: "Task", ProjectID: 1, Priority: "medium", Notes: validNotes}
	if err := valid.Validate(); err != nil {
		t.Fatalf("expected 255-char notes to be valid, got error: %v", err)
	}

	invalid := Task{Description: "Task", ProjectID: 1, Priority: "medium", Notes: invalidNotes}
	err := invalid.Validate()
	if err == nil {
		t.Fatal("expected validation error for notes longer than 255")
	}
	if err.Error() != "notes must be 255 characters or fewer" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestTask_IsOverdue(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	tomorrow := time.Now().AddDate(0, 0, 1)

	tests := []struct {
		name     string
		task     Task
		expected bool
	}{
		{
			name:     "past due date and not completed is overdue",
			task:     Task{DueDate: &yesterday, Completed: false},
			expected: true,
		},
		{
			name:     "past due date but completed is not overdue",
			task:     Task{DueDate: &yesterday, Completed: true},
			expected: false,
		},
		{
			name:     "future due date is not overdue",
			task:     Task{DueDate: &tomorrow, Completed: false},
			expected: false,
		},
		{
			name:     "no due date is not overdue",
			task:     Task{DueDate: nil, Completed: false},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.IsOverdue()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTask_PriorityOrder(t *testing.T) {
	tests := []struct {
		name     string
		task     Task
		expected int
	}{
		{
			name:     "high priority returns 1",
			task:     Task{Priority: "high"},
			expected: 1,
		},
		{
			name:     "medium priority returns 2",
			task:     Task{Priority: "medium"},
			expected: 2,
		},
		{
			name:     "low priority returns 3",
			task:     Task{Priority: "low"},
			expected: 3,
		},
		{
			name:     "unknown priority returns 99",
			task:     Task{Priority: "unknown"},
			expected: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.task.PriorityOrder()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
