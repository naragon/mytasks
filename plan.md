# My Tasks - Implementation Plan

## Context

**Problem:** Need a simple, efficient todo/task tracking tool for a single user to manage tasks across multiple projects.

**Solution:** A Go web application using htmx for responsive UI interactions and SQLite for data persistence. The application will be a single binary with embedded templates, deployable as a container.

**Key Decisions:**
- Go + htmx + SQLite (single build, efficient, container-ready)
- High/Medium/Low priority levels
- Projects have a `type` field: "project" (with target date) or "category" (ongoing)
- SortableJS for drag-and-drop reordering
- No authentication (local/trusted deployment)

---

## Data Model

### Project
| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER | Primary key, auto-increment |
| name | TEXT | Project name (required) |
| description | TEXT | Project description |
| type | TEXT | "project" or "category" |
| target_date | DATE | Target completion (null for categories) |
| sort_order | INTEGER | Display order on home page |
| created_at | DATETIME | Creation timestamp |
| updated_at | DATETIME | Last update timestamp |

### Task
| Field | Type | Description |
|-------|------|-------------|
| id | INTEGER | Primary key, auto-increment |
| project_id | INTEGER | Foreign key to project |
| description | TEXT | Task description (required) |
| priority | TEXT | "high", "medium", "low" |
| due_date | DATE | Task due date (optional) |
| completed | BOOLEAN | Completion status |
| sort_order | INTEGER | Display order within project |
| created_at | DATETIME | Creation timestamp |
| updated_at | DATETIME | Last update timestamp |

---

## Project Structure

```
mytasks/
├── main.go                 # Application entry point
├── go.mod                  # Go module definition
├── go.sum                  # Dependency checksums
├── Dockerfile              # Container build
├── Makefile                # Build commands
├── spec.md                 # Requirements (existing)
│
├── internal/
│   ├── models/
│   │   ├── project.go      # Project struct and methods
│   │   ├── project_test.go # Project unit tests
│   │   ├── task.go         # Task struct and methods
│   │   └── task_test.go    # Task unit tests
│   │
│   ├── store/
│   │   ├── store.go        # Database interface
│   │   ├── sqlite.go       # SQLite implementation
│   │   └── sqlite_test.go  # Store integration tests
│   │
│   └── handlers/
│       ├── handlers.go     # HTTP handlers
│       ├── handlers_test.go # Handler tests
│       ├── home.go         # Home page handler
│       ├── project.go      # Project CRUD handlers
│       └── task.go         # Task CRUD handlers
│
├── templates/
│   ├── layout.html         # Base layout
│   ├── home.html           # Home page
│   ├── project_detail.html # Project detail page
│   └── partials/
│       ├── project_card.html    # Project card component
│       ├── task_item.html       # Task list item
│       ├── project_form.html    # Project create/edit form
│       └── task_form.html       # Task create/edit form
│
└── static/
    ├── css/
    │   └── styles.css      # Application styles
    └── js/
        └── app.js          # SortableJS initialization
```

---

## API Endpoints

### Pages (HTML)
| Method | Path | Description |
|--------|------|-------------|
| GET | `/` | Home page with all projects |
| GET | `/projects/{id}` | Project detail page |

### Project API (htmx partials)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/projects` | Create project |
| PUT | `/api/projects/{id}` | Update project |
| DELETE | `/api/projects/{id}` | Delete project |
| POST | `/api/projects/reorder` | Update project order |

### Task API (htmx partials)
| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/projects/{id}/tasks` | Create task |
| PUT | `/api/tasks/{id}` | Update task |
| DELETE | `/api/tasks/{id}` | Delete task |
| POST | `/api/tasks/{id}/toggle` | Toggle completion |
| POST | `/api/projects/{id}/tasks/reorder` | Update task order |

---

## TDD Implementation Phases

### Phase 1: Core Models & Database (Tests First)

- [x] **1.1 Write Project Model Tests**
  - File: `internal/models/project_test.go`
  - Tests:
    - `TestProjectValidation_RequiredFields`
    - `TestProjectValidation_TypeValues`
    - `TestProjectValidation_CategoryNoTargetDate`
    - `TestProject_IsCategory`
    - `TestProject_IsOverdue`

- [x] **1.2 Implement Project Model**
  - File: `internal/models/project.go`
  - Make all tests pass

- [x] **1.3 Write Task Model Tests**
  - File: `internal/models/task_test.go`
  - Tests:
    - `TestTaskValidation_RequiredFields`
    - `TestTaskValidation_PriorityValues`
    - `TestTask_IsOverdue`
    - `TestTask_PriorityOrder`

- [x] **1.4 Implement Task Model**
  - File: `internal/models/task.go`
  - Make all tests pass

- [x] **1.5 Write Store Interface & SQLite Tests**
  - File: `internal/store/sqlite_test.go`
  - Tests:
    - `TestCreateProject`
    - `TestGetProject`
    - `TestListProjects_OrderedBySortOrder`
    - `TestUpdateProject`
    - `TestDeleteProject_CascadesTasks`
    - `TestReorderProjects`
    - `TestCreateTask`
    - `TestGetTask`
    - `TestListTasksByProject_OrderedBySortOrder`
    - `TestListTasksByProject_LimitThree`
    - `TestUpdateTask`
    - `TestDeleteTask`
    - `TestToggleTaskComplete`
    - `TestReorderTasks`

- [x] **1.6 Implement SQLite Store**
  - Files: `internal/store/store.go`, `internal/store/sqlite.go`
  - Make all tests pass

### Phase 2: HTTP Handlers (Tests First)

- [x] **2.1 Write Handler Tests**
  - File: `internal/handlers/handlers_test.go`
  - Tests:
    - `TestHomeHandler_ListsProjects`
    - `TestHomeHandler_ShowsTopThreeTasks`
    - `TestProjectDetailHandler_ShowsAllTasks`
    - `TestCreateProjectHandler_Success`
    - `TestCreateProjectHandler_ValidationError`
    - `TestUpdateProjectHandler_Success`
    - `TestDeleteProjectHandler_Success`
    - `TestReorderProjectsHandler_Success`
    - `TestCreateTaskHandler_Success`
    - `TestUpdateTaskHandler_Success`
    - `TestDeleteTaskHandler_Success`
    - `TestToggleTaskHandler_Success`
    - `TestReorderTasksHandler_Success`

- [x] **2.2 Implement Handlers**
  - Files: `internal/handlers/*.go`
  - Make all tests pass

### Phase 3: Templates & Static Assets

- [x] **3.1 Create Base Layout**
  - File: `templates/layout.html`
  - Include htmx, SortableJS, and styles

- [x] **3.2 Create Home Page Template**
  - File: `templates/home.html`
  - Show all projects with top 3 tasks each

- [x] **3.3 Create Project Detail Template**
  - File: `templates/project_detail.html`
  - Show all tasks for a project

- [x] **3.4 Create Partial Templates**
  - Files: `templates/partials/*.html`
  - Reusable components for htmx swaps

- [x] **3.5 Create Styles**
  - File: `static/css/styles.css`
  - Clean, responsive design

- [x] **3.6 Create JavaScript**
  - File: `static/js/app.js`
  - SortableJS initialization with htmx integration

### Phase 4: Application Wiring

- [x] **4.1 Write Main Integration Tests**
  - Test full request/response cycle
  - Test template rendering

- [x] **4.2 Implement Main Entry Point**
  - File: `main.go`
  - Wire up routes, templates, and database

### Phase 5: Containerization

- [x] **5.1 Create Dockerfile**
  - Multi-stage build
  - Embed templates and static files

- [x] **5.2 Create Makefile**
  - Build, test, run, docker targets

---

## Verification Plan

### Unit Tests
```bash
go test ./internal/models/... -v
go test ./internal/store/... -v
go test ./internal/handlers/... -v
```

### Integration Tests
```bash
go test ./... -v -tags=integration
```

### Manual Testing Checklist
- [x] Create a new project (type: project with target date)
- [x] Create a new category (type: category, no target date)
- [x] Add tasks to a project
- [x] View home page - verify only 3 tasks shown per project
- [x] View project detail - verify all tasks shown
- [ ] Drag-drop reorder projects on home page
- [ ] Drag-drop reorder tasks within a project
- [x] Mark task as complete/incomplete
- [ ] Edit project details
- [ ] Edit task details
- [ ] Delete a task
- [ ] Delete a project (verify tasks cascade delete)

### Container Testing
```bash
make docker-build
docker run -p 8080:8080 -v ./data:/data mytasks
# Verify app runs and data persists across restarts
```

---

## Dependencies

```go
// go.mod
module mytasks

go 1.22

require (
    github.com/go-chi/chi/v5 v5.0.12  // Router
    github.com/mattn/go-sqlite3 v1.14.22  // SQLite driver
)
```

## External Libraries (CDN)
- htmx 2.0.x - For AJAX interactions
- SortableJS 1.15.x - For drag-and-drop

---

## Files Created

| File | Status | Description |
|------|--------|-------------|
| `go.mod` | [x] | Module definition |
| `main.go` | [x] | Entry point |
| `Makefile` | [x] | Build automation |
| `Dockerfile` | [x] | Container build |
| `.gitignore` | [x] | Git ignore rules |
| `internal/models/project.go` | [x] | Project model |
| `internal/models/project_test.go` | [x] | Project tests |
| `internal/models/task.go` | [x] | Task model |
| `internal/models/task_test.go` | [x] | Task tests |
| `internal/store/store.go` | [x] | Store interface |
| `internal/store/sqlite.go` | [x] | SQLite implementation |
| `internal/store/sqlite_test.go` | [x] | Store tests |
| `internal/handlers/handlers.go` | [x] | Common handler utilities |
| `internal/handlers/handlers_test.go` | [x] | Handler tests |
| `internal/handlers/home.go` | [x] | Home page handler |
| `internal/handlers/project.go` | [x] | Project handlers |
| `internal/handlers/task.go` | [x] | Task handlers |
| `templates/layout.html` | [x] | Base layout |
| `templates/home.html` | [x] | Home page |
| `templates/project_detail.html` | [x] | Project detail |
| `templates/partials/project_card.html` | [x] | Project component |
| `templates/partials/task_item.html` | [x] | Task component |
| `templates/partials/project_form.html` | [x] | Project form |
| `templates/partials/task_form.html` | [x] | Task form |
| `static/css/styles.css` | [x] | Styles |
| `static/js/app.js` | [x] | JavaScript |

---

## Test Results

**37 tests passing:**
- Models: 9 tests
- Store: 15 tests
- Handlers: 13 tests
