# My Tasks

A server-rendered task management app built with Go, Chi, HTMX, and SQLite.

The app is organized around project Kanban boards and includes cross-project upcoming and archive views.

## Features

- Per-project Kanban board with `To Do`, `In Progress`, and `Done` columns
- Drag-and-drop task movement and ordering
- Sidebar project navigation with collapse/expand and resize controls
- Task metadata: priority, due date, notes, status
- Cross-project `Upcoming` view for due tasks
- `Archive` view for completed projects and older completed work
- SQLite persistence with schema migrations
- Embedded templates and static assets (`go:embed`)

## Tech Stack

- Go `1.22`
- Router/middleware: `github.com/go-chi/chi/v5`
- UI interactions: HTMX + SortableJS
- Database: SQLite (`github.com/mattn/go-sqlite3`)

## Quick Start

### Prerequisites

- Go 1.22+
- GNU Make

### Run (development)

```bash
make run-dev
```

This runs on:

- `PORT=3000`
- `DB_PATH=./data/dev.db`

### Run (default)

```bash
make run
```

This runs on:

- `PORT=8080` (default)
- `DB_PATH=./data/mytasks.db` (default)

### Build

```bash
make build
```

## Configuration

Environment variables:

- `PORT` (default: `8080`)
- `DB_PATH` (default: `./data/mytasks.db`)

Example:

```bash
PORT=3000 DB_PATH=./data/dev.db go run .
```

## Common Commands

- Format: `make fmt`
- Lint: `make lint`
- Test (all Go tests): `make test`
- Coverage: `make test-coverage`
- Tidy modules: `make tidy`

## Cleaning Artifacts and Data

- Safe clean (binary + coverage only): `make clean`
- Remove local database files: `make clean-data`
- Remove both build artifacts and DB files: `make clean-all`

## Testing

### Go tests

```bash
go test ./... -v
```

### Playwright E2E tests

Dependencies are in `package.json`.

```bash
npx playwright test e2e/kanban.spec.js --project=chromium
```

If the app is not already running, start it in another terminal:

```bash
make run-dev
```

## Project Structure

- `main.go`: app entrypoint, router setup, middleware, embedded assets
- `internal/models`: domain models and validation
- `internal/store`: `Store` interface and SQLite implementation
- `internal/handlers`: HTTP handlers for pages and API endpoints
- `templates`: full-page templates and partials
- `static`: CSS, JS, and vendor assets
- `e2e`: Playwright end-to-end tests

## Routes (high level)

Page routes:

- `/` (home/redirect)
- `/projects/{id}` (Kanban board)
- `/upcoming`
- `/archive`

API routes (selected):

- `/api/projects/*`
- `/api/tasks/*`

## Project & Task API

The `/api/...` endpoints are primarily designed for HTMX interactions.
Most write endpoints return either:

- `200 OK` with an empty body and an `HX-*` response header, or
- an HTML partial response (not JSON)

Validation errors return `400` with a plain-text message.

### Project Endpoints

| Method | Path | Purpose | Request Body | Response |
|---|---|---|---|---|
| `GET` | `/api/projects/form` | Get blank project form partial | none | HTML partial (`project_form.html`) |
| `GET` | `/api/projects/{id}/form` | Get edit project form partial | none | HTML partial (`project_form.html`) |
| `POST` | `/api/projects` | Create project | form: `name`, `description`, `type`, `target_date` | `200`, sets `HX-Redirect: /projects/{id}` |
| `PUT` | `/api/projects/{id}` | Update project | form: `name`, `description`, `type`, `target_date` | `200`, sets `HX-Refresh: true` |
| `POST` | `/api/projects/{id}/complete` | Mark project complete | none | `200`, sets `HX-Redirect: /archive` |
| `POST` | `/api/projects/{id}/reopen` | Reopen project | none | `200`, sets `HX-Redirect: /projects/{id}` |
| `DELETE` | `/api/projects/{id}` | Delete project | none | `200` |
| `POST` | `/api/projects/reorder` | Reorder sidebar projects | JSON: `{ \"ids\": [1,2,3] }` | `200` |

Notes:

- `type` currently uses `"project"` in UI forms.
- `target_date` format is `YYYY-MM-DD`.

### Task Endpoints

| Method | Path | Purpose | Request Body | Response |
|---|---|---|---|---|
| `GET` | `/api/projects/{project_id}/tasks/form` | Get blank task form partial | none | HTML partial (`task_form.html`) |
| `GET` | `/api/tasks/{id}/form` | Get edit task form partial | none | HTML partial (`task_form.html`) |
| `POST` | `/api/projects/{id}/tasks` | Create task in project | form: `description`, `notes`, `priority`, `status`, `due_date` | HTML partial (`task_item.html`) |
| `PUT` | `/api/tasks/{id}` | Update task | form: `description`, `notes`, `priority`, `status`, `due_date`, optional `project_id` | HTML partial (`task_item.html`) |
| `DELETE` | `/api/tasks/{id}` | Delete task | none | `200` |
| `POST` | `/api/tasks/{id}/toggle` | Toggle task complete/done | none | HTML partial (`task_item.html`) |
| `POST` | `/api/tasks/{id}/move` | Move task between Kanban columns | JSON: `{ \"status\": \"todo|in_progress|done\", \"sort_order\": 1 }` | `200` |
| `POST` | `/api/projects/{id}/tasks/reorder` | Reorder tasks within project or status | JSON: `{ \"ids\": [10,11,12] }`, optional query `?status=todo|in_progress|done` | `200` |

Notes:

- `priority` values: `high`, `medium`, `low`.
- `status` values: `todo`, `in_progress`, `done`.
- `due_date` format is `YYYY-MM-DD`.

### CSRF/Origin Behavior

For non-GET requests, middleware requires same-host `Origin` or `Referer`.
Requests without either header (or with a different host) are rejected with `403`.

### Example `curl` Requests

Set a base URL once:

```bash
BASE="http://localhost:3000"
```

Create a project:

```bash
curl -i -X POST "$BASE/api/projects" \
  -H "Origin: $BASE" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "name=Inbox&description=General work&type=project&target_date=2030-01-31"
```

Update a project:

```bash
curl -i -X PUT "$BASE/api/projects/1" \
  -H "Origin: $BASE" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "name=Inbox&description=Updated&type=project&target_date=2030-02-15"
```

Reorder projects:

```bash
curl -i -X POST "$BASE/api/projects/reorder" \
  -H "Origin: $BASE" \
  -H "Content-Type: application/json" \
  --data '{"ids":[2,1,3]}'
```

Create a task in a project:

```bash
curl -i -X POST "$BASE/api/projects/1/tasks" \
  -H "Origin: $BASE" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "description=Write docs&notes=README updates&priority=medium&status=todo&due_date=2030-01-15"
```

Update a task (including moving to another project):

```bash
curl -i -X PUT "$BASE/api/tasks/10" \
  -H "Origin: $BASE" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data "description=Write docs&notes=Done soon&priority=high&status=in_progress&due_date=2030-01-20&project_id=2"
```

Move a task between Kanban columns:

```bash
curl -i -X POST "$BASE/api/tasks/10/move" \
  -H "Origin: $BASE" \
  -H "Content-Type: application/json" \
  --data '{"status":"done","sort_order":1}'
```

Reorder tasks within a status column:

```bash
curl -i -X POST "$BASE/api/projects/1/tasks/reorder?status=todo" \
  -H "Origin: $BASE" \
  -H "Content-Type: application/json" \
  --data '{"ids":[12,10,11]}'
```

Delete a task:

```bash
curl -i -X DELETE "$BASE/api/tasks/10" \
  -H "Origin: $BASE"
```

Delete a project:

```bash
curl -i -X DELETE "$BASE/api/projects/1" \
  -H "Origin: $BASE"
```

## Docker

- Build image: `make docker-build`
- Run container: `make docker-run`

## License

This project is licensed under the terms in [LICENSE](./LICENSE).
