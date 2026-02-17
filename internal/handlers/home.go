package handlers

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"mytasks/internal/models"
)

// HomeData holds data for the home page template.
type HomeData struct {
	Title              string
	Tab                string // "active", "completed", or "upcoming"
	Projects           []models.Project
	UpcomingTasks      []models.Task
	UpcomingDays       int
	CompletedStartDate string
	CompletedEndDate   string
}

// Home renders the home page with all projects and their tasks filtered by tab.
func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get the tab from query parameter, default to "active"
	tab := r.URL.Query().Get("tab")
	if tab != "completed" && tab != "upcoming" {
		tab = "active"
	}

	projects, err := h.store.ListProjects(ctx)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Filter tasks based on tab (completed = true for completed tab, false for active)
	showCompleted := tab == "completed"
	showUpcoming := tab == "upcoming"

	upcomingDays := 30
	if showUpcoming {
		if v := r.URL.Query().Get("days"); v != "" {
			days, err := strconv.Atoi(v)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid days")
				return
			}
			if days != 7 && days != 14 && days != 30 {
				respondError(w, http.StatusBadRequest, "days must be 7, 14, or 30")
				return
			}
			upcomingDays = days
		}
	}

	now := time.Now()
	completedEnd := now
	completedStart := now.AddDate(0, 0, -30)

	if showCompleted {
		if v := r.URL.Query().Get("start_date"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid start_date")
				return
			}
			completedStart = t
		}

		if v := r.URL.Query().Get("end_date"); v != "" {
			t, err := time.Parse("2006-01-02", v)
			if err != nil {
				respondError(w, http.StatusBadRequest, "invalid end_date")
				return
			}
			completedEnd = t
		}

		if completedStart.After(completedEnd) {
			respondError(w, http.StatusBadRequest, "start_date cannot be after end_date")
			return
		}
	}

	today := now.Format("2006-01-02")
	upcomingEnd := now.AddDate(0, 0, upcomingDays)
	upcomingEndDate := upcomingEnd.Format("2006-01-02")

	// Load projects/tasks based on selected tab.
	filteredProjects := make([]models.Project, 0, len(projects))
	upcomingTasks := make([]models.Task, 0)
	for i := range projects {
		if projects[i].Completed {
			continue
		}

		var (
			tasks []models.Task
			err   error
		)

		if showCompleted {
			tasks, err = h.store.ListTasksByProjectCompletedBetween(ctx, projects[i].ID, &completedStart, &completedEnd, 0)
		} else if showUpcoming {
			tasks, err = h.store.ListTasksByProjectFiltered(ctx, projects[i].ID, false, 0)
		} else {
			tasks, err = h.store.ListTasksByProjectFiltered(ctx, projects[i].ID, false, 3)
		}
		if err != nil {
			respondError(w, http.StatusInternalServerError, err.Error())
			return
		}

		projects[i].ViewTab = tab
		projects[i].Tasks = tasks

		if showUpcoming {
			for _, task := range tasks {
				if task.DueDate == nil {
					continue
				}
				due := task.DueDate.Format("2006-01-02")
				if due > upcomingEndDate {
					continue
				}
				task.Overdue = due < today
				task.ProjectName = projects[i].Name
				upcomingTasks = append(upcomingTasks, task)
			}
			continue
		}

		if !showCompleted || len(tasks) > 0 {
			filteredProjects = append(filteredProjects, projects[i])
		}
	}

	if showUpcoming {
		sort.Slice(upcomingTasks, func(i, j int) bool {
			if upcomingTasks[i].Overdue != upcomingTasks[j].Overdue {
				return upcomingTasks[i].Overdue
			}
			leftDue := upcomingTasks[i].DueDate.Format("2006-01-02")
			rightDue := upcomingTasks[j].DueDate.Format("2006-01-02")
			if leftDue != rightDue {
				return leftDue < rightDue
			}
			return upcomingTasks[i].PriorityOrder() < upcomingTasks[j].PriorityOrder()
		})
	}

	data := HomeData{
		Title:              "My Tasks",
		Tab:                tab,
		Projects:           filteredProjects,
		UpcomingTasks:      upcomingTasks,
		UpcomingDays:       upcomingDays,
		CompletedStartDate: completedStart.Format("2006-01-02"),
		CompletedEndDate:   completedEnd.Format("2006-01-02"),
	}

	h.renderTemplate(w, "home.html", data)
}
