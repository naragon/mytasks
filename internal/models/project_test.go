package models

import (
	"testing"
	"time"
)

func TestProjectValidation_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		project Project
		wantErr bool
		errMsg  string
	}{
		{
			name:    "empty name should fail",
			project: Project{Name: "", Type: "project"},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "whitespace name should fail",
			project: Project{Name: "   ", Type: "project"},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name:    "valid name should pass",
			project: Project{Name: "Test Project", Type: "project"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
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

func TestProjectValidation_TypeValues(t *testing.T) {
	tests := []struct {
		name    string
		project Project
		wantErr bool
		errMsg  string
	}{
		{
			name:    "type project is valid",
			project: Project{Name: "Test", Type: "project"},
			wantErr: false,
		},
		{
			name:    "type category is valid",
			project: Project{Name: "Test", Type: "category"},
			wantErr: false,
		},
		{
			name:    "empty type should fail",
			project: Project{Name: "Test", Type: ""},
			wantErr: true,
			errMsg:  "type must be 'project' or 'category'",
		},
		{
			name:    "invalid type should fail",
			project: Project{Name: "Test", Type: "invalid"},
			wantErr: true,
			errMsg:  "type must be 'project' or 'category'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
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

func TestProjectValidation_CategoryNoTargetDate(t *testing.T) {
	targetDate := time.Now().AddDate(0, 1, 0)
	tests := []struct {
		name    string
		project Project
		wantErr bool
		errMsg  string
	}{
		{
			name:    "category with target date should fail",
			project: Project{Name: "Test", Type: "category", TargetDate: &targetDate},
			wantErr: true,
			errMsg:  "category cannot have a target date",
		},
		{
			name:    "category without target date should pass",
			project: Project{Name: "Test", Type: "category", TargetDate: nil},
			wantErr: false,
		},
		{
			name:    "project with target date should pass",
			project: Project{Name: "Test", Type: "project", TargetDate: &targetDate},
			wantErr: false,
		},
		{
			name:    "project without target date should pass",
			project: Project{Name: "Test", Type: "project", TargetDate: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
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

func TestProject_IsCategory(t *testing.T) {
	tests := []struct {
		name     string
		project  Project
		expected bool
	}{
		{
			name:     "type category returns true",
			project:  Project{Type: "category"},
			expected: true,
		},
		{
			name:     "type project returns false",
			project:  Project{Type: "project"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.project.IsCategory()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestProject_IsOverdue(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	tomorrow := time.Now().AddDate(0, 0, 1)

	tests := []struct {
		name     string
		project  Project
		expected bool
	}{
		{
			name:     "past target date is overdue",
			project:  Project{Type: "project", TargetDate: &yesterday},
			expected: true,
		},
		{
			name:     "future target date is not overdue",
			project:  Project{Type: "project", TargetDate: &tomorrow},
			expected: false,
		},
		{
			name:     "no target date is not overdue",
			project:  Project{Type: "project", TargetDate: nil},
			expected: false,
		},
		{
			name:     "category is never overdue",
			project:  Project{Type: "category", TargetDate: nil},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.project.IsOverdue()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
