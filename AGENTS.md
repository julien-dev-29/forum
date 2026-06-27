# AGENTS.md - Forum Project Guide

## Build & Run

```bash
CGO_ENABLED=1 go build -o forum ./cmd/web
CGO_ENABLED=1 go run ./cmd/web                                 # port 8081 (or $PORT)
CGO_ENABLED=1 go build -o forum && ./forum                      # build + run
```

## Test & Lint

```bash
CGO_ENABLED=1 go test ./...                                     # all tests
CGO_ENABLED=1 go test -v -run TestName ./internal/handlers/...  # single test
go vet ./...                                                     # vet
staticcheck ./...                                                # if installed
```

## Project Architecture

```
cmd/web/main.go                 Entry point, routes, HTTP server
internal/handlers/              HTTP handlers (auth, oauth, post, comment, like, notif, activity)
internal/database/sqlite/       SQLite data access layer (one file per domain)
internal/middleware/            Auth middleware (session cookie context injection)
internal/models/                Pure data structs, no methods, no JSON tags
internal/session/               Session CRUD + cookie read/write/delete
internal/oauth/                 OAuth configs, token exchange, user info (Google + GitHub)
ui/html/                        Go html/template files (11 .html files)
ui/static/css/style.css         Single CSS file with :root theme variables
ui/static/js/notifications.js   SSE client for live notification count
```

- **No web framework** — `net/http` with Go 1.22+ method-prefixed routes
- **Session auth** — UUID tokens, HttpOnly cookies, SameSite=Lax, 24h expiry
- **SQLite** — `mattn/go-sqlite3` (CGO_ENABLED=1), `?_foreign_keys=on`
- **bcrypt** — `golang.org/x/crypto` for password hashing
- **Port** — defaults to `8081` in code, `8080` in Docker

## Naming

| Scope | Case | Examples |
|-------|------|---------|
| Types, functions, methods (exported) | PascalCase | `User`, `CreateUser`, `GetAllPosts`, `LoginGet` |
| Unexported functions, methods | camelCase | `authHandler`, `renderTemplate`, `getUserID`, `getPostCategories` |
| Handler sub-structs | camelCase + `Handler` | `authHandler`, `oauthHandler`, `postHandler` |
| File names | snake_case | `auth.go`, `users.go`, `notifications.go` |
| Template files | kebab-case | `create-post.html`, `view-post.html` |
| Acronyms | ALL CAPS | `UserID`, `OAuthProvider`, `AuthURL`, `HTTP`, `API` |
| Template data keys | PascalCase | `"Authenticated"`, `"CurrentUserID"`, `"UnreadCount"` |
| Handler params | `w` (ResponseWriter), `r` (*Request) | Always `(w, r)` |
| DB func params | `db *sql.DB` as first arg | `func CreateUser(db *sql.DB, ...)` |

## Imports

Two groups: stdlib first (sorted), then blank line, then external/internal (sorted). Internal imports use `forum/...`.

```go
import (
    "database/sql"
    "fmt"
    "net/http"

    "forum/internal/models"

    "github.com/mattn/go-sqlite3"
    "golang.org/x/crypto/bcrypt"
)
```

Blank driver import: `_ "github.com/mattn/go-sqlite3"`

## Models

Pure exported structs, no methods, no JSON tags, exported fields. Field order: ID, FKs, data, CreatedAt last.

```go
type User struct {
    ID            int64
    Email         string
    Username      string
    Password      string
    OAuthProvider string
    OAuthID       string
    CreatedAt     time.Time
}
```

- `int64` for IDs, `int` for counts/like types
- `*int64` for nullable FK fields (e.g., `Like.PostID`, `Like.CommentID`)
- `string` for text, `time.Time` for timestamps
- `bool` for flags (e.g., `Notification.IsRead`)
- Like type: `1` = like, `-1` = dislike, `0` = none

## Handler Patterns

```go
func (h *postHandler) home(w http.ResponseWriter, r *http.Request) {
    // auth guard
    if !isAuthenticated(r) { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
    // form parsing
    if err := r.ParseForm(); err != nil { renderError(w, http.StatusBadRequest); return }
    // form values
    title := strings.TrimSpace(r.FormValue("title"))
    // URL query
    catID := r.URL.Query().Get("category")
    // template rendering
    renderTemplate(w, "index.html", map[string]any{
        "Authenticated": isAuthenticated(r),
        "CurrentUserID": getUserIDInt(r),
        "Username":      getUsername(r),
        "Posts":         posts,
    })
    // redirect
    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

Helper functions: `isAuthenticated(r)`, `getUserID(r)` returns `*int64` (nil if not auth), `getUserIDInt(r)` returns `int64` (0 if not auth), `getUsername(r)`, `getUnreadCount(db, r)`.

Form values: always `strings.TrimSpace(r.FormValue("key"))`. URL query: `r.URL.Query().Get("key")`.

## Error Handling

- **Startup**: `log.Fatal(err)`
- **Template parse**: `log.Fatalf("parse templates: %v", err)`
- **Template render**: `log.Printf("render template %s: %v", name, err)` then `http.Error(w, "...", 500)`
- **Database**: `fmt.Errorf("lowercase context: %w", err)`
- **sql.ErrNoRows**: return zero value, NOT an error
- **Handler errors**: `renderError(w, http.StatusXxx)` or re-render template with `"Error"` key
- **Ownership errors**: plain string `"not found or not owned"` (not wrapped)
- **Sentinel errors**: `fmt.Errorf("invalid credentials")` (not wrapped)
- **Notification side effects**: `_ = UpsertNotification(db, ...)`

## Database Layer

- `?` placeholders (NOT `$1`, `$2`)
- `rows.Next()` loop then `rows.Err()` check after loop
- `defer rows.Close()` immediately after error check
- `defer tx.Rollback()` after `db.Begin()` (safe no-op after Commit)
- `if results == nil { results = []T{} }` to return empty slice, not nil
- `COALESCE(SUM(CASE WHEN type = 1 THEN 1 ELSE 0 END), 0)` for like counts
- Public funcs: `GetAllX`, `GetXByID`, `CreateX`, `UpdateX`, `DeleteX`
- Private helpers for shared scan/enrich logic: `scanPosts`, `getPostCategories`

## Router (Go 1.22+)

```go
mux.HandleFunc("GET /{$}", h.Home)              // home
mux.HandleFunc("GET /post?id=", h.ViewPost)      // query params
mux.HandleFunc("POST /like/post", h.LikePost)    // method prefix
fs := http.FileServer(http.Dir("ui/static"))
mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
```

## Middleware (Auth)

```go
func Auth(db *sql.DB, next http.Handler) http.Handler
```
Context keys: `ContextKeyUserID`, `ContextKeyUsername` (type `contextKey = string`). Injects `user_id` and `username` into `r.Context()` when valid session cookie exists. Deletes cookie silently when session invalid/expired.

## Templates

11 `.html` files. Two `{{define}}` blocks in `footer.html` (`"header"` and `"footer"`). Glob loaded via `filepath.Join("ui", "html", "*.html")` with a `nowYear` FuncMap. All templates receive `.` (the map). Conditionals: `{{if .Authenticated}}`. Ranges: `{{range .Posts}}`.

## OAuth

Env vars: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`.

Account linking: (1) lookup by `(provider, oauth_id)`, (2) lookup by email (updates OAuth on existing account), (3) create new user. State cookie (`oauth_state`, 10min, Path=/auth/) for CSRF.

Redirect URL builder: `scheme://r.Host + path` (respects TLS and X-Forwarded-Proto).

## CSS Theming

All colors and values in `:root` with `--color-*`, `--radius-*`, `--font-family`, `--shadow-focus`, `--max-width` variables. Use `var(--variable)` throughout. Convention: `--color-{role}`, `--color-{role}-{variant}`.

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `DB_PATH` | `forum.db` | SQLite file path |
| `PORT` | `8081` | HTTP listen port |
| `GOOGLE_CLIENT_ID` | — | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | — | Google OAuth client secret |
| `GITHUB_CLIENT_ID` | — | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | — | GitHub OAuth client secret |

## Dependencies

- `github.com/google/uuid` — session token generation (v4)
- `github.com/mattn/go-sqlite3` — SQLite driver (blank import)
- `golang.org/x/crypto` — bcrypt password hashing
