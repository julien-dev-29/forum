# AGENTS.md - Forum Project Guide

## Build & Run

```bash
# Local HTTP (port 8081)
CGO_ENABLED=1 go build -o forum ./cmd/web
CGO_ENABLED=1 go run ./cmd/web

# Local HTTPS dev (port 8443, self-signed cert)
$env:DEV_MODE="true"; CGO_ENABLED=1 go run ./cmd/web

# Docker (HTTPS via autocert or HTTP fallback)
docker compose up --build

# Docker (dev mode with self-signed cert)
$env:DEV_MODE="true"; docker compose up --build
```

## Test & Lint

```bash
CGO_ENABLED=1 go test ./...                              # all tests
CGO_ENABLED=1 go test -v -run TestName ./internal/...    # single test
go vet ./...                                              # vet
staticcheck ./...                                         # if installed
CGO_ENABLED=1 go test -count=1 ./internal/...             # no cache
```

## Project Architecture

```
cmd/web/main.go                 Entry point, routes, HTTPS/HTTP server
internal/handlers/              HTTP handlers (auth, oauth, post, comment, like, notif, activity, admin, modrequest)
internal/database/sqlite/       SQLite data access (one file per domain)
internal/middleware/            Auth middleware (session cookie → context)
internal/models/                Pure data structs, no methods, no JSON tags
internal/session/               Session CRUD + cookie read/write/delete + rotation
internal/oauth/                 OAuth configs, token exchange, user info (Google + GitHub)
internal/tls/                   TLS config, autocert, self-signed certs, security headers, HSTS, HTTPS redirect
internal/ratelimit/             Token bucket rate limiter per client IP
internal/csrf/                  Double-submit cookie CSRF protection for all mutating methods
internal/crypto/                AES-256-GCM field encryption + SHA-256 hashing for PII
ui/html/                        Go html/template files (13 .html files)
ui/static/css/style.css         Single CSS file with :root theme variables
ui/static/js/notifications.js   SSE client for live notification count
```

- **No web framework** — `net/http` with Go 1.22+ method-prefixed routes
- **Session auth** — UUID tokens, HttpOnly cookies, SameSite=Lax, Secure flag (HTTPS), 7d absolute expiry
- **SQLite** — `mattn/go-sqlite3` (CGO_ENABLED=1), `?_foreign_keys=on`
- **bcrypt** — `golang.org/x/crypto` for password hashing
- **Port** — defaults to `8081` (HTTP), `8443` (dev HTTPS), `443` (production autocert)
- **OAuth** — Google + GitHub via env vars
- **Admin** — first user with `ADMIN_EMAIL` becomes admin
- **Session cleanup** — background goroutine runs every 30 min
- **Rate limiting** — per-IP token bucket (configurable per route)
- **CSRF** — double-submit cookie pattern, applied via middleware, token in all forms
- **Database encryption** — AES-256-GCM for email fields (SHA-256 hash for lookups)

## Server Modes

Three start modes controlled by env vars:

| Mode | Env | Port | TLS |
|------|-----|------|-----|
| HTTP (legacy) | (none) | 8081 | No |
| Dev HTTPS | `DEV_MODE=true` | 8443 | Self-signed |
| Production HTTPS | `AUTOCERT_HOST=example.com` | 443 | Let's Encrypt (autocert) |

In `autocert` mode, port 80 runs ACME HTTP-01 challenge handler + redirect to HTTPS.

## Middleware Order (outermost → innermost)

1. `tls.SecurityHeaders` — CSP, X-Frame-Options, X-Content-Type-Options, Referrer-Policy, Permissions-Policy
2. `tls.HSTS` (HTTPS modes only) — Strict-Transport-Security
3. `csrf.Middleware` — validates token for POST/PUT/PATCH/DELETE
4. `middleware.Auth` — injects user_id/username/role into context from session cookie
5. Rate limiters — applied per-route in mux setup (before handler)

## TLS/HTTPS

- **Production**: `golang.org/x/crypto/acme/autocert` with Let's Encrypt
- **Development**: auto-generated self-signed RSA-2048 certificate (localhost/127.0.0.1)
- **Cipher suites** (TLS 1.2+ only): ECDHE-ECDSA/ECDHE-RSA with AES-256-GCM, AES-128-GCM, ChaCha20-Poly1305
- **HSTS**: `max-age=31536000; includeSubDomains` (production only)
- **HTTP→HTTPS redirect**: 301 Moved Permanently, respects X-Forwarded-Proto

## Rate Limiting

Token bucket algorithm (`internal/ratelimit`). Per-IP limits applied in `main.go`:

| Route | Rate |
|-------|------|
| POST /register | 3/min |
| POST /login | 5/min |
| OAuth endpoints | 10/min |
| POST /post/new, /post/edit | 30/min |
| POST /comment, /like/* | 30/min |
| Admin POST routes | 10/min |
| GET /api/* | 60/min |

Returns `429 Too Many Requests` with `Retry-After` header. Stale buckets cleaned every 5 min.

## CSRF Protection

Double-submit cookie pattern (`internal/csrf`):
- Token stored in HttpOnly, SameSite=Strict cookie
- Must be echoed back via `X-CSRF-Token` header or `csrf_token` form field
- Applied to all POST/PUT/PATCH/DELETE routes via middleware
- All 13 `.html` templates include `{{.CSRFToken}}` in hidden inputs
- Helper: `getCSRFToken(w, r)` in `internal/handlers/helpers.go` generates cookie on first GET

## Session Security

- UUID v4 tokens, stored server-side in SQLite
- 7-day absolute expiry (not just idle)
- Session rotation on re-login (`session.Rotate`)
- `Secure` flag enabled when HTTPS active
- HttpOnly + SameSite=Lax
- Old sessions deleted on new login (single-session-per-user)

## Database Encryption

- **Enabled by** `DB_ENCRYPTION_KEY` env var (32 bytes, hex or base64)
- **Algorithm**: AES-256-GCM (from `crypto/aes` + `crypto/cipher`)
- **Scope**: email field encrypted at rest, SHA-256 hash used for lookup
- **Migration**: existing emails backfilled on startup via `ALTER TABLE` + batch UPDATE
- **Fallback**: if key is unset, operates in plaintext mode (backward compatible)

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
- **renderError**: sets Content-Type before WriteHeader, executes error template directly

## Handler Patterns

```go
func (h *postHandler) home(w http.ResponseWriter, r *http.Request) {
    if !isAuthenticated(r) { http.Redirect(w, r, "/login", http.StatusSeeOther); return }
    if err := r.ParseForm(); err != nil { renderError(w, http.StatusBadRequest); return }
    title := strings.TrimSpace(r.FormValue("title"))
    catID := r.URL.Query().Get("category")
    renderTemplate(w, "index.html", map[string]any{
        "Authenticated": isAuthenticated(r),
        "CurrentUserID": getUserIDInt(r),
        "Username":      getUsername(r),
        "Role":          getRole(r),
        "Posts":         posts,
        "CSRFToken":     getCSRFToken(w, r),
    })
    http.Redirect(w, r, "/", http.StatusSeeOther)
}
```

Helpers: `isAuthenticated(r)`, `getUserID(r)` returns `*int64`, `getUserIDInt(r)` returns `int64` (0 if not auth), `getUsername(r)`, `getRole(r)`, `isAdmin(r)`, `isModOrAdmin(r)`, `getUnreadCount(db, r)`, `getCSRFToken(w, r)`.

Form values: always `strings.TrimSpace(r.FormValue("key"))`. URL query: `r.URL.Query().Get("key")`.

## Naming Conventions

| Scope | Case | Examples |
|-------|------|----------|
| Types, functions, methods (exported) | PascalCase | `User`, `CreateUser`, `GetAllPosts`, `LoginGet` |
| Unexported functions, methods | camelCase | `authHandler`, `renderTemplate`, `getUserID` |
| Handler sub-structs | camelCase + `Handler` | `authHandler`, `oauthHandler`, `postHandler` |
| File names | snake_case | `auth.go`, `users.go`, `notifications.go` |
| Template files | kebab-case | `create-post.html`, `view-post.html` |
| Acronyms | ALL CAPS | `UserID`, `OAuthProvider`, `AuthURL`, `HTTP`, `API` |
| Template data keys | PascalCase | `"Authenticated"`, `"CurrentUserID"`, `"CSRFToken"` |
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
    Role          string
    CreatedAt     time.Time
}
```

- `int64` for IDs, `int` for counts/like types
- `*int64` for nullable FK fields (e.g., `Like.PostID`, `Like.CommentID`)
- `string` for text, `time.Time` for timestamps
- `bool` for flags (e.g., `Notification.IsRead`)
- Like type: `1` = like, `-1` = dislike, `0` = none

## Database Layer

- `?` placeholders (NOT `$1`, `$2`)
- `rows.Next()` loop then `rows.Err()` check after loop
- `defer rows.Close()` immediately after error check
- `defer tx.Rollback()` after `db.Begin()` (safe no-op after Commit)
- `if results == nil { results = []T{} }` to return empty slice, not nil
- `COALESCE(SUM(CASE WHEN type = 1 THEN 1 ELSE 0 END), 0)` for like counts
- Public funcs: `GetAllX`, `GetXByID`, `CreateX`, `UpdateX`, `DeleteX`
- Private helpers for shared scan/enrich logic: `scanPosts`, `getPostCategories`
- Migrations: `ALTER TABLE ... ADD COLUMN ...` in `InitSchema` (ignores errors if column exists)
- Unique indexes: `CREATE UNIQUE INDEX IF NOT EXISTS idx_likes_post ON likes(user_id, post_id) WHERE post_id IS NOT NULL`
- When `DB_ENCRYPTION_KEY` is set: uses `email_hash` for lookups, `email_encrypted` for display, separate `scanUserEncrypted` function

## Router (Go 1.22+)

```go
mux.HandleFunc("GET /{$}", h.Home)              // home
mux.HandleFunc("GET /post?id=", h.ViewPost)      // query params
mux.HandleFunc("POST /like/post", h.LikePost)    // method prefix
// Rate-limited routes:
mux.Handle("POST /login", loginLimiter.Middleware()(http.HandlerFunc(h.LoginPost)))
fs := http.FileServer(http.Dir("ui/static"))
mux.Handle("GET /static/", http.StripPrefix("/static/", fs))
```

Use `mux.Handle` (not `HandleFunc`) when wrapping with rate limiter middleware.

## Middleware (Auth)

```go
func Auth(db *sql.DB, next http.Handler) http.Handler
```

Context keys: `ContextKeyUserID`, `ContextKeyUsername`, `ContextKeyRole` (type `contextKey = string`). Injects `user_id`, `username`, and `role` into `r.Context()` when valid session cookie exists. Deletes cookie silently when session invalid/expired.

## Templates

13 `.html` files. Two `{{define}}` blocks in `footer.html` (`"header"` and `"footer"`). Glob loaded via `filepath.Join("ui", "html", "*.html")` with a `nowYear` FuncMap. All templates receive `.` (the map). Conditionals: `{{if .Authenticated}}`. Ranges: `{{range .Posts}}`.

Every form includes `<input type="hidden" name="csrf_token" value="{{.CSRFToken}}">`. For forms inside a range loop, use `{{$.CSRFToken}}`.

## OAuth

Env vars: `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`, `GITHUB_CLIENT_ID`, `GITHUB_CLIENT_SECRET`.

Account linking: (1) lookup by `(provider, oauth_id)`, (2) lookup by email (updates OAuth on existing account), (3) create new user. State cookie (`oauth_state`, 10min, Path=/auth/, Secure when HTTPS) for CSRF.

Redirect URL builder: `scheme://r.Host + path` (respects TLS and X-Forwarded-Proto).

## CSS Theming

All colors and values in `:root` with `--color-*`, `--radius-*`, `--font-family`, `--shadow-focus`, `--max-width` variables. Use `var(--variable)` throughout. Convention: `--color-{role}`, `--color-{role}-{variant}`.

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `DB_PATH` | `forum.db` | SQLite file path |
| `PORT` | `8081` | HTTP/HTTPS listen port |
| `AUTOCERT_HOST` | — | Domain for Let's Encrypt (comma-separated, enables autocert) |
| `AUTOCERT_CACHE_DIR` | `certs` | Certificate cache directory |
| `DEV_MODE` | — | Enable self-signed HTTPS for local development |
| `SESSION_SECURE` | `false` | Set Secure flag on session cookies |
| `DB_ENCRYPTION_KEY` | — | AES-256-GCM key (32 bytes hex/base64, enables field encryption) |
| `GOOGLE_CLIENT_ID` | — | Google OAuth client ID |
| `GOOGLE_CLIENT_SECRET` | — | Google OAuth client secret |
| `GITHUB_CLIENT_ID` | — | GitHub OAuth client ID |
| `GITHUB_CLIENT_SECRET` | — | GitHub OAuth client secret |
| `ADMIN_EMAIL` | — | Email of first user to register as admin |

## Dependencies

- `github.com/google/uuid` — session + CSRF token generation (v4)
- `github.com/mattn/go-sqlite3` — SQLite driver (blank import)
- `golang.org/x/crypto` — bcrypt password hashing, autocert TLS, ChaCha20-Poly1305

## Docker Notes

- Multi-stage build: `golang:1.22-alpine` builder → `alpine:3.19` runtime
- Runtime installs `ca-certificates` + `sqlite-libs`
- Exposes `443` and `80` for HTTPS/ACME support
- `PORT=443` in production, `AUTOCERT_HOST` for Let's Encrypt
- Volume `forum_data:/data` persists SQLite DB, `certs:/certs` for TLS certificates
- OAuth env vars passed from host `.env` file through compose
