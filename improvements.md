# MyTasks — Implementation Evaluation & Suggested Improvements

## What's Done Well

Before the issues: the architecture is genuinely solid. The `Store` interface is clean and enables
proper in-memory testing. Migrations are embedded and versioned with a legacy bootstrap path. The
chi router, layered separation (`models` → `store` → `handlers`), consistent error-wrapping with
`%w`, `defer rows.Close()` everywhere, and transaction usage in reorder operations are all correct.
Test coverage across all three packages is good.

---

## 🔴 High Priority — Correctness & Security

> Update: All high-priority items below have been addressed in code.

### 1. Race condition in sort order assignment
**Status:** ✅ Fixed


Both `CreateProject` and `CreateTask` compute the new `sort_order` by fetching all existing records
and using `len()`:

```go
// internal/handlers/project.go
projects, _ := h.store.ListProjects(ctx)
project.SortOrder = len(projects) + 1
```

Two simultaneous POSTs will read the same count and write the same `sort_order`. The right fix is a
single `INSERT … SELECT MAX(sort_order)+1` or a `MAX` query inside a transaction in the store layer
— not at the handler level at all.

### 2. Silently swallowed errors on that same path
**Status:** ✅ Fixed


```go
projects, _ := h.store.ListProjects(ctx)   // error is thrown away
project.SortOrder = len(projects) + 1
```

If `ListProjects` fails, `projects` is `nil`, `len(nil)` is `0`, and every new project gets
`sort_order = 1`. Same pattern in `CreateTask`. These should return a 500 when the error is
non-nil.

### 3. No CSRF protection
**Status:** ✅ Fixed (origin/referer validation middleware added for unsafe methods)


All state-mutating routes (`POST`, `PUT`, `DELETE`) are registered without any CSRF token. htmx
sends ordinary browser requests, so a malicious page can trigger arbitrary mutations via a
`<form action="/api/projects" method="post">`. The chi ecosystem has `github.com/justinas/nosurf`
or `gorilla/csrf` that integrates easily.

### 4. Internal error details exposed to HTTP clients
**Status:** ✅ Fixed


```go
respondError(w, http.StatusInternalServerError, err.Error())
```

`err.Error()` may include SQL text, file paths, or constraint details. Log the internal error
server-side and return only a generic message to the client:

```go
log.Printf("store error: %v", err)
respondError(w, http.StatusInternalServerError, "internal server error")
```

### 5. External CDN scripts without Subresource Integrity (SRI)
**Status:** ✅ Fixed (moved to local vendored static assets)


```html
<script src="https://unpkg.com/htmx.org@2.0.4"></script>
<script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.6/Sortable.min.js"></script>
```

These load without `integrity=` hashes. A compromised CDN could inject arbitrary JavaScript. Either
add SRI hashes or self-host the files under `static/js/vendor/` (they are already embedded via
`//go:embed static/*`).

---

## 🟠 Medium Priority — Code Quality & Maintainability

> Update: Most medium-priority items are addressed below; remaining deferred items are marked.

### 6. `renderTemplate` and `renderPartial` are identical
**Status:** ✅ Fixed


```go
// internal/handlers/handlers.go  — bodies are byte-for-byte the same
func (h *Handlers) renderTemplate(w http.ResponseWriter, name string, data interface{}) { … }
func (h *Handlers) renderPartial(w http.ResponseWriter, name string, data interface{}) { … }
```

One should delegate to the other, or they should be merged into a single `render` helper. Having
two names that mean the same thing adds confusion about when to use which.

### 7. Massive scan-block duplication in the store
**Status:** ⏸️ Valid concern, deferred (no behavior risk; refactor-only)


The 12-field task scan block (including null-date parsing) is copy-pasted verbatim into `GetTask`,
`ListTasksByProject`, `ListTasksByProjectFiltered`, and `ListTasksByProjectCompletedBetween`. A
`scanTask(rows scanner) (models.Task, error)` helper would cut this to one place. Same applies to
the project scan in `GetProject` vs `ListProjects`.

### 8. Dead code: form handlers registered nowhere
**Status:** ✅ Fixed


`GetTaskForm` and `GetProjectForm` are implemented in the handler files but have no routes in
`main.go`. They cannot be reached. Either register routes for them or delete the functions.

### 9. `respondJSON` is defined but never used
**Status:** ✅ Fixed (removed unused helper)


```go
// internal/handlers/handlers.go
func respondJSON(w http.ResponseWriter, data interface{}) { … }
```

No handler returns JSON. This is dead code and will not compile cleanly if the `encoding/json`
import is ever tidied.

### 10. SQLite not configured for WAL mode or connection limits
**Status:** ✅ Fixed


```go
db, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
```

For any concurrent load (including dev testing), WAL journal mode dramatically reduces lock
contention:

```go
dbPath + "?_foreign_keys=on&_journal_mode=WAL&_busy_timeout=5000"
```

And for SQLite specifically, `db.SetMaxOpenConns(1)` is the safest default when not in WAL mode
(WAL supports multiple readers with one writer).

### 11. LIMIT injected via `fmt.Sprintf` instead of a query parameter
**Status:** ✅ Fixed


```go
query += fmt.Sprintf(" LIMIT %d", limit)
```

`limit` is a typed `int` so there is no actual injection risk, but this pattern is inconsistent
with the parameterized style used everywhere else. Use `query += " LIMIT ?"` and append `limit` to
`args`.

### 12. Extra SELECT before every `UpdateTask`
**Status:** ⏸️ Valid concern, deferred (requires careful SQL rewrite to preserve completion semantics)


`UpdateTask` issues a `SELECT completed, completed_at … WHERE id = ?` before the `UPDATE` to
preserve `completed_at`. The same result can be achieved in a single SQL statement using a `CASE`
expression (analogous to `ToggleTaskComplete`), eliminating a database round-trip per task edit.

---

## 🟡 Low Priority — Testing

> Update: Low-priority items were validated; fixes/deferred decisions are noted per item.

### 13. Home handler tests pass trivially because templates are `nil`
**Status:** ✅ Fixed


`TestHomeHandler_HidesCompletedProjects` asserts that the response body does not contain
`"Completed"`. But `h.templates` is `nil`, so `renderTemplate` returns a blank 200 body — the
assertion always passes regardless of filtering logic. Handler-layer tests that verify rendering
logic need either a minimal real template or a different assertion strategy (e.g., inspect the
`HomeData` struct before it is passed to the template).

### 14. Tests use hardcoded ID `"1"` instead of the created entity's ID
**Status:** ℹ️ Valid but not changed (current tests use isolated in-memory DBs where first insert is deterministic)


```go
// handlers_test.go
rctx.URLParams.Add("id", "1")
```

If autoincrement behaviour ever changes (or tests run in a different order), these break silently.
Use `strconv.FormatInt(project.ID, 10)` from the actual created entity.

### 15. `sqlite_test.go` accesses `store.db` directly
**Status:** ℹ️ Reviewed; acceptable in same-package persistence tests (no production impact)


Two tests (`TestListTasksByProjectCompletedBetween`,
`TestNewSQLiteStore_MigratesLegacyDatabaseAndPreservesData`) use `store.db.ExecContext` and
`store.db.Query` to manipulate data around the public API. This breaks encapsulation and makes
tests tightly coupled to implementation. Add test-support helpers or expose narrow internal hooks
for controlled test setup.

### 16. Missing negative-path tests for handlers
**Status:** ✅ Fixed


There are no handler tests for: updating/deleting a non-existent task, a non-existent project,
invalid JSON on reorder endpoints, or `start_date > end_date` on the home completed tab. Each of
these paths branches to a `respondError` but is untested.

---

## 🔵 Operational / UX

### 17. No graceful shutdown

```go
if err := http.ListenAndServe(addr, r); err != nil {
    log.Fatalf("Server failed: %v", err)
}
```

In-flight requests are dropped immediately on SIGTERM. Go's `http.Server.Shutdown(ctx)` with a
signal handler is a straightforward fix and important in containerised deployments.

### 18. No cache headers on static assets

Static files served via `http.FileServer` get no `Cache-Control` headers. Adding a short max-age
(e.g., `"public, max-age=3600"`) in a middleware wrapper avoids re-fetching CSS/JS on every page
load.

### 19. Full `window.location.reload()` on task toggle from home page

```javascript
hx-on::after-request="if(event.detail.successful) window.location.reload()"
```

This is used pervasively on the home and upcoming pages after toggling a task. It refetches and
re-renders the entire page for a single checkbox click. The `task_item.html` partial already returns
the updated element — htmx's `hx-target` and `hx-swap="outerHTML"` could be used directly, the
same way the project detail page does.

### 20. Fetch errors silently ignored in `app.js`

```javascript
fetch('/api/projects/reorder', {
    method: 'POST',
    …
    body: JSON.stringify({ ids: ids }),
});
// no .then()/.catch()
```

A failed reorder (network error, server restart) is invisible to the user. At minimum, a
`.catch(err => console.error('reorder failed', err))` should be present; ideally, a brief
toast/banner for the user.

---

## Summary Table

| # | Area | Issue | Severity | Status |
|---|------|-------|----------|--------|
| 1 | Correctness | Race condition in sort order | 🔴 High | ✅ Fixed |
| 2 | Correctness | Silently swallowed store errors | 🔴 High | ✅ Fixed |
| 3 | Security | No CSRF protection | 🔴 High | ✅ Fixed |
| 4 | Security | Internal errors leaked to clients | 🔴 High | ✅ Fixed |
| 5 | Security | CDN scripts without SRI hashes | 🔴 High | ✅ Fixed |
| 6 | Code quality | Duplicate `renderTemplate`/`renderPartial` | 🟠 Medium | ✅ Fixed |
| 7 | Code quality | 4× duplicated task scan block | 🟠 Medium | ⏸️ Deferred |
| 8 | Code quality | Dead form handler code, unregistered routes | 🟠 Medium | ✅ Fixed |
| 9 | Code quality | Unused `respondJSON` | 🟠 Medium | ✅ Fixed |
| 10 | Performance | SQLite WAL mode not configured | 🟠 Medium | ✅ Fixed |
| 11 | Code quality | LIMIT via `fmt.Sprintf` instead of `?` | 🟠 Medium | ✅ Fixed |
| 12 | Performance | Extra SELECT per `UpdateTask` | 🟠 Medium | ⏸️ Deferred |
| 13 | Testing | Handler tests pass trivially with nil templates | 🟡 Low | ✅ Fixed |
| 14 | Testing | Hardcoded ID `"1"` in handler tests | 🟡 Low | ℹ️ Accepted |
| 15 | Testing | Direct `store.db` access in tests | 🟡 Low | ℹ️ Accepted |
| 16 | Testing | No negative-path handler tests | 🟡 Low | ✅ Fixed |
| 17 | Ops | No graceful shutdown | 🔵 Ops | ⏳ Open |
| 18 | Ops | No static asset caching | 🔵 Ops | ⏳ Open |
| 19 | UX | Full page reload on task toggle | 🔵 UX | ⏳ Open |
| 20 | UX | Silent fetch errors on reorder | 🔵 UX | ⏳ Open |
