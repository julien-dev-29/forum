# AGENTS.md - Forum Project Guide

## Build & Run

```bash
# Build (CGO required for sqlite3)
CGO_ENABLED=1 go build -o forum ./cmd/web

# Run (default port 8081, or set PORT env var)
go run ./cmd/web
# or
CGO_ENABLED=1 go run ./cmd/web
```

## Testing

```bash
# Run all tests
CGO_ENABLED=1 go test ./...

# Run all tests with verbose output
CGO_ENABLED=1 go test -v ./...

# Run tests in a specific package
CGO_ENABLED=1 go test -v ./internal/handlers/...

# Run a single test by name (replace TestName)
CGO_ENABLED=1 go test -v -run TestName ./internal/handlers/...

# Run a single test file
CGO_ENABLED=1 go test -v ./internal/database/sqlite/ -run TestCreateUser
```

## Lint & Vet

```bash
# Standard vet
go vet ./...

# Run staticcheck if available
staticcheck ./...
```

## Docker

```bash
docker compose up --build
```

## Project Architecture

```
cmd/web/main.go            - Entry point, routes, HTTP server
internal/handlers/         - HTTP handlers (auth, post, comment, like)
internal/database/sqlite/  - SQLite data access layer
internal/middleware/        - Auth middleware (session cookie)
internal/models/            - Pure data structs (User, Post, Comment, Category, Like)
internal/session/           - Session CRUD and cookie management
ui/html/                    - Go html/template files (server-side rendered)
ui/static/css/              - Static CSS (no JavaScript)
```

- **No external web framework** - uses `net/http` with Go 1.22+ route patterns
- **Session-based auth** with UUID tokens, HttpOnly cookies, SameSite=Lax
- **SQLite** via `mattn/go-sqlite3` (requires CGO_ENABLED=1)
- **bcrypt** for password hashing via `golang.org/x/crypto`
- **Port**: defaults to `8081` in code, `8080` in Docker

## Code Style Guidelines

### Imports
- Group: stdlib first, blank line, third-party packages
- Internal imports use module path `forum/...`
- Blank import `_ "github.com/mattn/go-sqlite3"` for driver registration

### Formatting
- Standard `gofmt` formatting (tabs for indentation)
- No comments in production code
- No trailing commas or semicolons

### Naming
- Exported types/functions: PascalCase (`User`, `CreateUser`, `GetAllPosts`)
- Unexported: camelCase (`authHandler`, `getPostCategories`, `parseTime`)
- Acronyms: all-caps (`UserID`, not `UserId`)
- Models: pure structs, exported fields, no JSON tags, no methods
- Handler structs: unexported, `*sql.DB` field named `db`
- Database functions: package-level, take `*sql.DB` as first arg
- Template data keys: PascalCase (`"Authenticated"`, `"UserID"`, `"Error"`, `"Posts"`)

### Error Handling
- Startup errors: `log.Fatal(err)`
- Template rendering errors: `log.Printf` (non-fatal)
- Database errors: `fmt.Errorf("context message: %w", err)`
- HTTP handler errors: `renderError(w, statusCode)` or re-render template with `"Error"` key
- `defer tx.Rollback()` after `db.Begin()` (Commit sets tx to nil)
- `defer rows.Close()` after `db.Query()`

### Database Layer
- Named `?` placeholders (not `$1`, `$2`)
- `rows.Next()` loop then `rows.Err()` check after loop
- `if posts == nil { posts = []models.Post{} }` to return empty slice, not nil
- `sql.ErrNoRows` treated as "not found" (return zero value, not error)
- Public functions: `GetAllX`, `GetXByID`, `CreateX`
- Unexported helpers for shared scan/enrich logic

### HTTP Handlers
- Methods: lowercase (e.g., `registerPost`, `home`, `viewPost`)
- Exported via `Handler` struct delegation (`h.Post.home(w, r)`)
- `renderTemplate(w, "name.html", map[string]any{...})` for success
- `renderError(w, http.StatusXxx)` for errors
- `http.Redirect(w, r, "/path", http.StatusSeeOther)` for redirects
- Always check `isAuthenticated(r)` before protected actions
- `getUserID(r)` returns `*int64` (nil if not authenticated)
- Form values: `strings.TrimSpace(r.FormValue("key"))`
- URL query params: `r.URL.Query().Get("key")`

### Router Pattern
- Go 1.22+ syntax: `"GET /{$}"`, `"POST /post/new"`, `"GET /post?id="`
- Static files: `mux.Handle("GET /static/", http.StripPrefix("/static/", fs))`

### Middleware
- Signature: `func Auth(db *sql.DB, next http.Handler) http.Handler`
- Context keys: custom unexported `contextKey` string type
- Inject `user_id` into `r.Context()` when valid session exists
- Delete cookie silently when session invalid/expired

### Models
```go
type User struct {
    ID        int64
    Email     string
    Username  string
    Password  string
    CreatedAt time.Time
}
```
- `int64` for IDs, `int` for counts/like types
- `*int64` for nullable FK fields (e.g., `Like.PostID`)
- `time.Time` for timestamps
- LIKE type: `1` = like, `-1` = dislike, `0` = none

### Template Helpers
- `getUsername(r)` returns string from middleware context
- `getUserIDInt(r)` returns `int64` (0 if not authenticated) for template comparisons
- `getUnreadCount(db, r)` returns unread notification count for the header badge

### Advanced Features (Notifications, Activity, Edit/Delete)

**Notifications** — `internal/database/sqlite/notifications.go`
- `notifications` table: user_id (recipient), actor_id, type, post_id, comment_id, is_read
- Created on like/dislike/comment actions (skips self-notification)
- Deleted when like is removed (stays in sync); upserted when type changes
- Routes: `GET /notifications`, `POST /notifications/read`
- **Real-time badge** via SSE (`GET /api/notifications/stream`): server pushes `{"count": N}` every 2 seconds when count changes; `ui/static/js/notifications.js` updates `.notif-count` spans in header
- Unread count shown as badge next to username in header

**Activity Page** — `internal/database/sqlite/activity.go`
- Three sections: user's posts, comments (with post title), likes/dislikes
- Route: `GET /activity`
- Helper types: `models.UserComment`, `models.UserLike`

**Edit/Delete Posts** — routes: `GET /post/edit`, `POST /post/edit`, `POST /post/delete`
- Author-only via `user_id` check in SQL WHERE clause
- `UpdatePost` uses a transaction (updates post + replaces categories)
- `DeletePost` uses CASCADE deletes for comments/likes/post_categories

**Edit/Delete Comments** — routes: `GET /comment/edit`, `POST /comment/edit`, `POST /comment/delete`
- Author-only via `user_id` check
- Redirects back to parent post after edit/delete

### Dependencies
- `github.com/google/uuid` - session tokens (v4)
- `github.com/mattn/go-sqlite3` - SQLite driver
- `golang.org/x/crypto` - bcrypt
