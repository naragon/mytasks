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

func TestProjectValidation_TypeDefaults(t *testing.T) {
	tests := []struct {
		name    string
		project Project
		wantErr bool
	}{
		{
			name:    "type project is valid",
			project: Project{Name: "Test", Type: "project"},
			wantErr: false,
		},
		{
			name:    "empty type defaults to project",
			project: Project{Name: "Test", Type: ""},
			wantErr: false,
		},
		{
			name:    "legacy category type still accepted",
			project: Project{Name: "Test", Type: "category"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.project.Validate()
			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
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
