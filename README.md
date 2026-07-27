# Strata-prototype

Building Strata — the frontier ERP-CRM Hybrid, direct competitor to Odoo.

---

## Project structure

```
cmd/
├── api/main.go          # Entry point. Wires config, database, and starts the server.
├── cli/                 # CLI commands (future — admin tasks, data imports, etc.)
└── migrate/             # Database migrations (future)

internal/
├── api/
│   ├── server.go        # App struct, Serve() method, HTTP server config
│   ├── routes.go        # All route registration and middleware wiring
│   ├── health.go        # GET /health endpoint
│   ├── response.go      # JSON response helpers (writeJSON, writeErr)
│   └── middleware.go    # Request logging, panic recovery, CORS
├── config/config.go     # App configuration loaded from env vars
├── database/database.go # Placeholder DB type (swap with pgx / sqlx when ready)
└── env/env.go           # Safe environment variable helpers (GetString, GetInt, GetBool)
```

---

## Adding a new endpoint

### 1. Create a handler file in `internal/api/`

```go
// internal/api/users.go
package api

import "net/http"

func (a *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Business logic goes here.
	data := envelope{
		"users": []string{},
	}
	writeJSON(w, http.StatusOK, data)
}
```

### 2. Register the route in `internal/api/routes.go`

```go
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.healthHandler)
	mux.HandleFunc("GET /users", a.listUsersHandler)   // <-- add here

	// ...
}
```

> **Naming convention:** `HandleFunc("METHOD /path", a.resourceActionHandler)`

### 3. Group routes by resource when the list grows

When a resource has many endpoints (e.g. CRUD for users), extract a helper method:

```go
// internal/api/routes.go
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.healthHandler)
	a.registerUserRoutes(mux)   // groups GET/POST/PUT/DELETE /users & /users/{id}

	return a.withMiddleware(mux)
}

func (a *App) registerUserRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /users", a.listUsersHandler)
	mux.HandleFunc("POST /users", a.createUserHandler)
	mux.HandleFunc("GET /users/{id}", a.getUserHandler)
	mux.HandleFunc("PUT /users/{id}", a.updateUserHandler)
	mux.HandleFunc("DELETE /users/{id}", a.deleteUserHandler)
}
```

---

## Adding a new feature (service layer)

For non-trivial business logic, add a service package under `internal/`:

```
internal/
├── api/
├── auth/        # Authentication & authorization
├── billing/     # Invoicing, subscriptions
├── crm/         # Leads, deals, pipeline logic
├── erp/         # Inventory, orders, MRP
└── users/       # User management & profiles
```

Services are plain Go structs called from handlers:

```go
// internal/users/service.go
package users

type Service struct {
	repo *Repository
}

func (s *Service) List(ctx context.Context) ([]User, error) { ... }
func (s *Service) GetByID(ctx context.Context, id string) (*User, error) { ... }

// internal/api/users.go
func (a *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	users, err := a.Users.List(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not list users")
		return
	}
	writeJSON(w, http.StatusOK, envelope{"users": users})
}
```

Wire the service into `App`:

```go
// internal/api/server.go
type App struct {
	Config config.Config
	DB     *database.DB
	Users  *users.Service   // <-- add field
}
```

This keeps handlers thin — they parse input, call a service, write output. Business logic stays testable without HTTP.

---

## Adding a utility or helper

### Internal helpers (used across the app)

Put shared utilities inside a package under `internal/utils`:

```
internal/utils/
├── validate.go     # Input validation helpers
├── paginate.go      # Pagination helpers (page, limit, offset)
├── filter.go        # Query filtering / sorting helpers
├── respond.gp       # Extended response helpers (pagination meta, errors)
└── types.go        # Shared domain types
```

Example — pagination helper:

```go
// internal/utils/paginate.go
package paginate

type Params struct {
	Page   int
	Limit  int
	Offset int
}

func FromRequest(r *http.Request) Params { ... }
func (p Params) SQL() string             { return fmt.Sprintf("LIMIT %d OFFSET %d", p.Limit, p.Offset) }
```

### Handlers should stay thin — delegate to helpers

```go
func (a *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	p := paginate.FromRequest(r)
	users, total, err := a.Users.List(r.Context(), p)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "could not list users")
		return
	}
	writeJSON(w, http.StatusOK, envelope{
		"users":      users,
		"pagination": envelope{"page": p.Page, "limit": p.Limit, "total": total},
	})
}
```

---

## Adding middleware

### 1. Write the middleware function in `internal/api/middleware.go`

```go
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rate-limit logic…
		next.ServeHTTP(w, r)
	})
}
```

### 2. Wire it in `routes.go`

```go
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", a.healthHandler)
	// ...

	handler := http.Handler(mux)
	handler = corsMiddleware(handler)
	handler = rateLimitMiddleware(handler)   // <-- add here (order matters)
	handler = loggingMiddleware(handler)
	handler = recoveryMiddleware(handler)
	return handler
}
```

**Middleware runs outside-in.** The first wrapped middleware runs outermost (last to run).

---

## Adding a new database table

### 1. Add an `internal/<resource>/repository.go`

```go
// internal/users/repository.go
package users

import "github.com/patiHash1/Strata-prototype/internal/database"

type Repository struct {
	db *database.DB
}

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindAll(ctx context.Context) ([]User, error) { ... }
func (r *Repository) FindByID(ctx context.Context, id string) (*User, error) { ... }
```

### 2. Wire repository → service → handler

```go
// internal/users/service.go
type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}
```

```go
// cmd/api/main.go
db   := database.New(cfg.DatabaseURL)
userRepo := users.NewRepository(db)
userSvc  := users.NewService(userRepo)

app := api.App{
	Config: cfg,
	DB:     db,
	Users:  userSvc,
}
```

---

## Configuration conventions

All configuration lives in `internal/config/config.go` loaded via `internal/env/env.go`.

```go
// Adding a new config field:
type Config struct {
	Port int
	DB   DBConfig

	// New:
	JWTSecret string
	SMTPHost  string
}
```

Populate from env:

```go
func Load() Config {
	return Config{
		Port: env.GetInt("PORT", 8080),
		DB: DBConfig{
			DSN: env.GetString("DATABASE_URL", ""),
		},
		JWTSecret: env.GetString("JWT_SECRET", "change-me"),
		SMTPHost:  env.GetString("SMTP_HOST", ""),
	}
}
```

---

## Response conventions

- **Success:** `{"key": value}` with the response data directly as a top-level key.
- **Error:** `{"error": "message"}`
- **Pagination:** include a `pagination` key alongside the data key.
- **Always** use `writeJSON` / `writeErr` from `internal/api/response.go`.

---

## Project conventions

| Convention | Guidance |
|---|---|
| **Go version** | 1.26 (match `go.mod`) |
| **Package name** | Lowercase, short, no underscores — `users`, `auth`, `crm` |
| **File naming** | Singular, descriptive — `service.go`, `repository.go`, `handler.go` |
| **Handler signature** | Always `func (a *App) actionHandler(w http.ResponseWriter, r *http.Request)` |
| **Service signature** | Always accept `context.Context` as the first argument |
| **Tests** | Place next to the file — `service_test.go`, `handler_test.go` |
| **Error handling** | Services return errors; handlers translate them to HTTP responses |
| **No global state** | Everything lives on `App` or is injected via constructor |

---

## Running the project

```sh
# Standard
go run ./cmd/api

# With air (hot-reload, uses .air.toml)
air

# Smoke test
curl http://localhost:8080/health
```

---

## Roadmap patterns (what comes next)

- **cmd/cli/** — standalone CLI commands (user creation, data exports, cron jobs).
- **cmd/migrate/** — database migration runner using `golang-migrate` or similar.
- **internal/mailer/** — transactional email (SMTP or SendGrid/Mailgun).
- **internal/events/** — background job queue (async email, webhooks, reports).
- **internal/test/** — shared test fixtures, factories, and helpers.