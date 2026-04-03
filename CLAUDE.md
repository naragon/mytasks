# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
# Build
make build              # Build binary
go build -o mytasks .   # Alternative direct build

# Test
make test               # Run all tests
go test ./... -v        # Run all tests verbose
go test ./internal/models/... -v    # Test specific package
go test -run TestCreateProject ./internal/store/...  # Run single test

# Run
make run-dev            # Development mode (port 3000, ./data/dev.db)
make run                # Production mode (port 8080, ./data/mytasks.db)

# Docker
make docker-build       # Build container
make docker-run         # Run container with volume mount
```

## Architecture

This is a Go web application using htmx for UI interactions and SQLite for persistence. Templates and static files are embedded in the binary via `//go:embed`.

### Layers

```
main.go                 → Entry point, routing (chi), template parsing
internal/handlers/      → HTTP handlers, render templates/partials
internal/store/         → Data persistence (Store interface + SQLite impl)
internal/models/        → Domain types (Project, Task) with validation
templates/              → HTML templates (embedded)
static/                 → CSS/JS assets (embedded)
```

### Request Flow

1. **Page requests** (`/`, `/projects/{id}`) → Handler fetches data via Store → Renders full HTML template
2. **API requests** (`/api/...`) → Handler performs CRUD via Store → Returns HTML partial for htmx swap

### Key Patterns

- **Store interface** (`internal/store/store.go`): All database operations go through this interface. SQLite implementation in `sqlite.go`. Tests use `:memory:` database.
- **Template structure**: Page templates (`home.html`, `project_detail.html`) are self-contained. Partials in `templates/partials/` are reused for htmx responses.
- **Handler tests**: Pass `nil` for templates when testing API logic only.

### Data Model

- **Project**: Has `type` field ("project" with optional target_date, or "category" without)
- **Task**: Belongs to Project, has priority (high/medium/low), optional due_date, completed flag
- Both have `sort_order` for drag-drop reordering

### Environment Variables

- `PORT` - Server port (default: 8080)
- `DB_PATH` - SQLite database path (default: ./data/mytasks.db)


<!-- BEGIN BEADS INTEGRATION v:1 profile:minimal hash:ca08a54f -->
## Beads Issue Tracker

This project uses **bd (beads)** for issue tracking. Run `bd prime` to see full workflow context and commands.

### Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --claim  # Claim work
bd close <id>         # Complete work
```

### Rules

- Use `bd` for ALL task tracking — do NOT use TodoWrite, TaskCreate, or markdown TODO lists
- Run `bd prime` for detailed command reference and session close protocol
- Use `bd remember` for persistent knowledge — do NOT use MEMORY.md files

## Session Completion

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd dolt push
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
<!-- END BEADS INTEGRATION -->
Use 'bd' for task tracking
