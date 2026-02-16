# AGENTS.md

Guidance for coding agents working in this repository.

## Agent Quick Start

Use this default loop for most code changes:

1. `go fmt ./...`
2. `go test -run '^TestName$' ./path/to/package -v` (targeted)
3. `go test ./path/to/package -v` (broader package scope)
4. `go vet ./...`
5. `go test ./... -v` (full suite for risky/wide changes)

Prefer `make` targets when available (`make fmt`, `make lint`, `make test`).

## Project Snapshot

- Language: Go (`go 1.22`)
- App type: server-rendered web app (chi router + htmx)
- Persistence: SQLite (`github.com/mattn/go-sqlite3`)
- Router/middleware: `github.com/go-chi/chi/v5`
- Entry point: `main.go`
- Key layers:
  - `internal/models` for domain data + validation
  - `internal/store` for persistence interface + SQLite implementation
  - `internal/handlers` for HTTP handlers and template rendering
  - `templates` for full pages and partials
  - `static` for CSS/JS assets

## Source of Truth for Commands

Use `Makefile` first when possible.

## Build, Run, Format, Lint

- Build binary: `make build`
- Build directly: `go build -o mytasks .`
- Run prod-like mode: `make run` (defaults to port `8080`)
- Run dev mode: `make run-dev` (port `3000`, DB `./data/dev.db`)
- Format code: `make fmt` or `go fmt ./...`
- Lint/static checks: `make lint` or `go vet ./...`
- Tidy dependencies: `make tidy` or `go mod tidy`

## Test Commands (General)

- All tests (verbose): `make test`
- All tests directly: `go test ./... -v`
- Coverage run: `make test-coverage`
- Package-only tests:
  - `go test ./internal/models/... -v`
  - `go test ./internal/store/... -v`
  - `go test ./internal/handlers/... -v`

## Test Commands (Single Test Focus)

Prefer `-run` with an anchored regex when running one test.

- Single test in a package:
  - `go test -run '^TestCreateProject$' ./internal/store -v`
  - `go test -run '^TestTaskValidation_PriorityValues$' ./internal/models -v`
- Single handler test:
  - `go test -run '^TestUpdateTaskHandler_Success$' ./internal/handlers -v`
- Subtest selection:
  - `go test -run 'TestProjectValidation_TypeValues/invalid type should fail' ./internal/models -v`
- Multiple related tests by prefix:
  - `go test -run '^TestReorder' ./internal/... -v`
- Disable test cache while iterating:
  - `go test -count=1 -run '^TestCreateTask$' ./internal/store -v`

## Architecture + Flow Rules

- Keep data access behind `store.Store` interface (`internal/store/store.go`).
- `SQLiteStore` should satisfy `store.Store`; preserve method signatures.
- Handlers should call store methods via `h.store`, not direct SQL.
- Page routes render full templates; `/api/...` routes return partial HTML or status.
- Continue using embedded assets (`//go:embed templates/*`, `//go:embed static/*`).
- Keep request flow simple: parse -> validate -> store call -> render/response.

## Code Style: Imports and Formatting

- Always run `go fmt ./...` after edits.
- Use standard Go import grouping:
  1. standard library
  2. third-party modules
  3. local module imports (`mytasks/...`)
- Keep import lists gofmt-sorted; do not hand-tune order.
- Keep line length readable; prefer clarity over dense one-liners.
- Use tabs/formatting produced by gofmt; do not align manually.

## Code Style: Types and Data Modeling

- Use concrete structs for domain entities (`Project`, `Task`) in `internal/models`.
- Use pointers for optional dates (`*time.Time`) and nullable form fields.
- Keep JSON tags snake_case as established (`project_id`, `target_date`, etc.).
- Preserve current enum-like string values:
  - Project type: `project`, `category`
  - Task priority: `high`, `medium`, `low`
- Add validation rules in model `Validate()` methods when domain constraints change.

## Code Style: Naming Conventions

- Exported identifiers: `CamelCase` with clear domain meaning.
- Unexported helpers: short, descriptive lowerCamelCase (`parseID`, `respondError`).
- Test names: `Test<Subject>_<Behavior>` or `Test<Function>`.
- Table tests: use `tests := []struct{...}` with `name` field + `t.Run`.
- Avoid abbreviations except common Go usage (`ctx`, `err`, `id`).

## Error Handling Conventions

- Return early on errors; avoid deep nesting.
- Wrap lower-level errors with context using `%w`:
  - `fmt.Errorf("failed to create task: %w", err)`
- For not-found in store, return a clear message (`"project not found: %d"`).
- In handlers, convert parsing/validation issues to `400`.
- In handlers, convert missing resources to `404` where appropriate.
- In handlers, convert unexpected store/render failures to `500`.
- Keep HTTP error body simple text unless endpoint contract requires JSON.

## Handler Implementation Notes

- Parse route params via shared helper `parseID`.
- Parse date inputs via `parseDate("2006-01-02")` pattern.
- Call `r.ParseForm()` before reading form values.
- For JSON payloads, decode into explicit local structs.
- Use `h.renderTemplate` for page views, `h.renderPartial` for htmx responses.
- In tests that do not need rendering, initialize handlers with `nil` templates.

## Store/SQLite Implementation Notes

- Keep SQL statements parameterized with `?` placeholders.
- Use transactions for reorder operations and multi-step updates.
- Always `defer rows.Close()` and check `rows.Err()`.
- Maintain `sort_order` semantics (1-based ordering in reorder loops).
- Preserve current date serialization format for DB DATE fields: `2006-01-02`.
- Respect foreign key behavior (`?_foreign_keys=on`, cascade delete).

## Testing Guidelines

- Prefer in-memory DB (`:memory:`) in store/handler tests.
- Use `t.Helper()` for setup helpers.
- Use `t.Cleanup` for closing stores/resources.
- Validate both success path and failure path for new behavior.
- Assert status code first for handler tests, then response/body/state.
- Keep tests deterministic; avoid relying on wall clock beyond simple overdue checks.

## Configuration and Environment

- `PORT` defaults to `8080`.
- `DB_PATH` defaults to `./data/mytasks.db`.
- Dev convention: `PORT=3000`, `DB_PATH=./data/dev.db`.

## Rule Files Check

No repository-specific Cursor or Copilot instruction files were found:

- No `.cursorrules`
- No `.cursor/rules/`
- No `.github/copilot-instructions.md`

If any of these files are added later, treat them as higher-priority agent instructions and update this document.

## Agent Work Checklist

- Read `CLAUDE.md` and this file before making non-trivial changes.
- Match existing layer boundaries (`models` vs `store` vs `handlers`).
- Run `go fmt ./...`.
- Run targeted tests first (`go test -run ...`), then broader package tests.
- For risky changes, run full suite: `go test ./... -v`.
