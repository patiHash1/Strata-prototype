# Strata-prototype

Building Strata — the frontier ERP-CRM Hybrid, direct competitor to Odoo.

---

## Project structure

```
cmd/
├── api/main.go          # Entry point. Wires config, database, services, and starts the server.
├── cli/                 # CLI commands (future — admin tasks, data imports, etc.)
└── migrate/             # Database migrations (future)

internal/
├── services/            # BUSINESS LAYER — service + repository per domain
│   ├── services_auth.go        # AuthService: JWT, bcrypt
│   ├── services_users.go       # UserService: users, organization memberships
│   ├── services_orgs.go        # OrgService: organizations, invitations, API keys
│   ├── services_rbac.go        # RBACService: roles, permissions
│   ├── services_billing.go     # BillingService: subscriptions
│   ├── services_crm.go         # CRMService: leads, deals, quotes, AI analysis
│   ├── services_accounting.go  # AccountingService: journal entries, OCR, expenses
│   ├── services_supplychain.go # SupplyChainService: fleet, telematics, inventory, routes
│   ├── services_hr.go          # HRService: attendance, resume parsing, knowledge search
│   ├── services_platform.go    # PlatformService: AI copilot, workflows, security anomalies
│   └── services_mailer.go      # Mailer: transactional email (stub)
│
├── handlers/            # HTTP LAYER — handlers, routes, App wiring
│   ├── handlers_server.go      # App struct, New(), Serve() — all dependency injection
│   ├── handlers_routes.go      # Route registration + middleware stack + Swagger UI
│   ├── handlers_health.go      # GET /health
│   ├── handlers_auth.go        # POST auth/register, POST auth/login
│   ├── handlers_org.go         # POST org/invitations, org/roles, org/api-keys
│   ├── handlers_billing.go     # POST billing/subscriptions
│   ├── handlers_crm.go         # POST crm/leads, quotes/risk-analysis, crm/tickets
│   ├── handlers_accounting.go  # POST accounting/journal-entries, invoices/ocr, expenses
│   ├── handlers_supplychain.go # POST fleet/telematics, fleet/routes, GET inventory/reorder-predictions
│   ├── handlers_hr.go          # POST hr/attendance/clock-in, hr/ats/parse-resume, hr/knowledge/search
│   └── handlers_platform.go    # POST ai/copilot/query, workflows/trigger, GET security/audit-anomalies
│
├── utils/               # SHARED HELPERS — no business logic
│   ├── response.go      # WriteJSON, WriteErr, Envelope type
│   ├── middleware.go     # RequireAuth, RequirePermission, RequireAPIKey, Logging, CORS, Recovery
│   └── validator.go     # IsEmail, IsDomainSlug, NotBlank, MinLen
│
├── config/config.go     # App configuration loaded from env vars
├── database/database.go # pgx connection pool (real, not a placeholder)
└── env/env.go           # Safe environment variable helpers (GetString, GetInt, GetBool)

docs/                    # Auto-generated Swagger/OpenAPI spec (do not edit)
```

### Package dependency flow

```
cmd/api/main.go
    │
    ├── internal/config       (env → struct)
    ├── internal/database     (pgx pool)
    ├── internal/services     (business logic + DB queries)
    │   └── depends on: database
    ├── internal/handlers     (HTTP handlers + App struct)
    │   └── depends on: services, utils, config, database
    └── internal/utils        (pure helpers — no dependencies)
```

---

## API documentation (Swagger)

The project uses [swaggo/swag](https://github.com/swaggo/swag) to generate an OpenAPI 2.0 spec from Go annotations, served via a Swagger UI.

### Viewing the docs

```sh
# Start the server (Swagger UI is enabled by default)
ENABLE_SWAGGER=true go run ./cmd/api

# Open in your browser:
#   http://localhost:8080/swagger/
```

### Annotating a new endpoint

Add Go comments above your handler function in `internal/handlers/`:

```go
// ListUsersResponse represents the response body for the list users endpoint.
type ListUsersResponse struct {
	Users []services.User `json:"users"`
	Total int             `json:"total" example:"42"`
}

// listUsersHandler returns a paginated list of users.
//
//	@Summary		List users
//	@Description	Returns all users with pagination.
//	@Tags			Users
//	@Produce		json
//	@Param			page	query	int	false	"Page number"	default(1)
//	@Param			limit	query	int	false	"Items per page"	default(20)
//	@Success		200	{object}	ListUsersResponse
//	@Failure		500	{object}	utils.Envelope
//	@Router			/users [get]
func (a *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	// ...
}
```

### Regenerating the spec

```sh
swag init --dir ./cmd/api,./internal/handlers --output ./docs --parseDependency --parseInternal
```

> The `docs/` directory is gitignored and regenerated on demand. CI should run `swag init` before building to ensure the spec is always fresh.

### Config

| Env var | Default | Description |
|---|---|---|
| `ENABLE_SWAGGER` | `true` | Set to `false` in production to disable the Swagger UI route. |
| `JWT_SECRET` | `dev-secret-change-in-production` | HMAC signing key for JWTs |
| `DATABASE_URL` | `""` | Postgres connection string (if empty, runs DB-less) |

---

## Adding a new endpoint

### 1. Create a handler file in `internal/handlers/`

```go
// internal/handlers/handlers_users.go
package handlers

import (
	"net/http"

	"github.com/patiHash1/Strata-prototype/internal/utils"
)

func (a *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
	// Business logic goes here.
	data := utils.Envelope{
		"users": []string{},
	}
	utils.WriteJSON(w, http.StatusOK, data)
}
```

> **File naming:** `handlers_<category>.go` — one file per endpoint category (auth, org, billing, etc.)

### 2. Register the route in `internal/handlers/handlers_routes.go`

```go
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.healthHandler)
	mux.HandleFunc("GET /users", a.listUsersHandler)   // <-- add here

	// ... protected routes with middleware ...
}
```

### 3. Add permission-gated routes

Protected routes use the middleware stack directly in the route registration:

```go
// Bearer token (JWT) auth:
mux.Handle("POST /api/v1/users",
    utils.RequireAuth(a.Auth)(
        utils.RequirePermission(services.PermUsersInvite)(
            http.HandlerFunc(a.inviteHandler),
        ),
    ),
)

// API key auth:
mux.Handle("POST /api/v1/fleet/telematics/ingest",
    utils.RequireAPIKey(a.SupplyChain, services.PermFleetTelematicsIngest)(
        http.HandlerFunc(a.ingestTelemetryHandler),
    ),
)
```

---

## Adding a new feature (service layer)

For non-trivial business logic with database access, add a service file under `internal/services/`:

```
internal/services/
├── services_auth.go
├── services_users.go
├── services_orgs.go
├── services_rbac.go
├── services_billing.go
├── services_mailer.go
├── services_crm.go        # CRM (leads, deals, quotes, AI)
├── services_accounting.go  # Accounting (journal entries, OCR, expenses)
├── services_supplychain.go # Supply chain (fleet, telematics, inventory, routes)
├── services_hr.go          # HR (attendance, resume parsing, knowledge search)
├── services_platform.go    # Platform (AI copilot, workflows, security anomalies)
└── services_mailer.go      # Mailer (transactional email stub)
```

A service file contains its own types, repository (SQL queries), and exported service struct:

```go
// internal/services/services_crm.go
package services

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type Lead struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Email       string `json:"email"`
}

// ---- Repository ----

type crmRepository struct {
	pool *pgxpool.Pool
}

func newCRMRepository(pool *pgxpool.Pool) *crmRepository {
	return &crmRepository{pool: pool}
}

func (r *crmRepository) Create(ctx context.Context, l *Lead) error {
	// ... SQL ...
}

// ---- Service ----

type CRMService struct {
	repo *crmRepository
}

func NewCRMService(pool *pgxpool.Pool) *CRMService {
	return &CRMService{repo: newCRMRepository(pool)}
}

func (s *CRMService) Create(ctx context.Context, name, email string) (*Lead, error) {
	// ... business logic ...
}
```

### Wire the new service

**Step 1 — Add field to `App` in `internal/handlers/handlers_server.go`:**

```go
type App struct {
	Config      config.Config
	DB          *database.DB
	Auth        *services.AuthService
	Users       *services.UserService
	Orgs        *services.OrgService
	RBAC        *services.RBACService
	Billing     *services.BillingService
	Mailer      *services.Mailer
	CRM         *services.CRMService       // <-- new field
	HR          *services.HRService
	Platform    *services.PlatformService   // AI copilot, workflows, security
	SupplyChain *services.SupplyChainService
	server      *http.Server
}
```

**Step 2 — Add constructor parameter in `New()`:**

```go
func New(
	cfg config.Config,
	db *database.DB,
	authSvc *services.AuthService,
	userSvc *services.UserService,
	orgSvc *services.OrgService,
	rbacSvc *services.RBACService,
	billingSvc *services.BillingService,
	mailerSvc *services.Mailer,
	crmSvc *services.CRMService,   // <-- new parameter
	accountingSvc *services.AccountingService,
	supplyChainSvc *services.SupplyChainService,
) *App {
	return &App{
		// ...
		CRM: crmSvc,
	}
}
```

**Step 3 — Create service in `cmd/api/main.go`:**

```go
crmSvc := services.NewCRMService(db.Pool)

app := handlers.New(cfg, db, authSvc, userSvc, orgSvc, rbacSvc, billingSvc, mailerSvc, crmSvc, accountingSvc, supplyChainSvc)
```

> **Note:** Some services may require additional dependencies. For example, `SupplyChainService` also accepts `*AuthService` for API key bcrypt verification.

### Use the service from a handler

```go
// internal/handlers/handlers_crm.go
func (a *App) createLeadHandler(w http.ResponseWriter, r *http.Request) {
	var req createLeadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
		return
	}

	lead, err := a.CRM.Create(r.Context(), req.Name, req.Email)
	if err != nil {
		utils.WriteErr(w, http.StatusInternalServerError, "could not create lead")
		return
	}

	utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"lead": lead})
}
```

This keeps handlers thin — they parse input, call a service, write output. Business logic stays testable without HTTP.

---

## Adding a utility or helper

Shared utilities live in `internal/utils/` and have no dependencies on other project packages:

```
internal/utils/
├── response.go      # WriteJSON, WriteErr, Envelope
├── middleware.go     # RequireAuth, RequirePermission, RequireAPIKey, Logging, CORS, Recovery, GetClaims
└── validator.go     # IsEmail, IsDomainSlug, NotBlank, MinLen
```

Handlers call utils directly:

```go
utils.WriteJSON(w, http.StatusOK, utils.Envelope{
    "users": users,
})
```

```go
if !utils.NotBlank(req.Name) {
    utils.WriteErr(w, http.StatusBadRequest, "name is required")
    return
}
```

---

## Adding middleware

### 1. Write the middleware function in `internal/utils/middleware.go`

```go
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rate-limit logic…
		next.ServeHTTP(w, r)
	})
}
```

### 2. Wire it in `internal/handlers/handlers_routes.go`

```go
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	// ... routes ...

	// Stack middleware (outermost first):
	var handler http.Handler = mux
	handler = utils.CORSMiddleware(handler)
	handler = utils.LoggingMiddleware(handler)
	handler = utils.RecoveryMiddleware(handler)
	return handler
}
```

**Middleware runs outside-in.** The first wrapped middleware runs outermost (last to run).

### Per-route middleware

Auth and permission checks are applied per-route. Strata supports two auth modes:

**Bearer token (JWT) auth:**

```go
mux.Handle("POST /api/v1/org/invitations",
    utils.RequireAuth(a.Auth)(
        utils.RequirePermission(services.PermUsersInvite)(
            http.HandlerFunc(a.inviteHandler),
        ),
    ),
)
```

**API key auth:**

```go
mux.Handle("POST /api/v1/fleet/telematics/ingest",
    utils.RequireAPIKey(a.SupplyChain, services.PermFleetTelematicsIngest)(
        http.HandlerFunc(a.ingestTelemetryHandler),
    ),
)
```

---

## Adding a new database table

### 1. Add service + repository in `internal/services/services_<domain>.go`

The service file contains everything: types, repository struct with SQL, and exported service.

```go
// internal/services/services_crm.go
package services

type Lead struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type crmRepository struct {
	pool *pgxpool.Pool
}

func (r *crmRepository) Create(ctx context.Context, l *Lead) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO leads ...`, ...)
	return err
}

type CRMService struct {
	repo *crmRepository
}

func NewCRMService(pool *pgxpool.Pool) *CRMService {
	return &CRMService{repo: &crmRepository{pool: pool}}
}
```

### 2. Wire in `handlers_server.go` and `cmd/api/main.go`

Add the field to `App`, add the constructor parameter, create the service in main.

---

## Configuration conventions

All configuration lives in `internal/config/config.go` loaded via `internal/env/env.go`.

```go
// Adding a new config field:
type Config struct {
	Port    int
	BaseURL string
	DB      DBConfig
	JWTSecret string

	// New:
	SMTPHost string
}
```

Populate from env:

```go
func Load() Config {
	return Config{
		Port:    env.GetInt("PORT", 8080),
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
- **Always** use `utils.WriteJSON` / `utils.WriteErr` with the `utils.Envelope` type.

---

## Project conventions

| Convention | Guidance |
|---|---|
| **Go version** | 1.26 (match `go.mod`) |
| **Services package** | `internal/services/` — types, repos, business logic |
| **Handlers package** | `internal/handlers/` — HTTP handlers, routes, App struct |
| **Utils package** | `internal/utils/` — response helpers, middleware, validators |
| **File naming** | `services_<domain>.go`, `handlers_<category>.go` |
| **Handler signature** | Always `func (a *App) actionHandler(w http.ResponseWriter, r *http.Request)` |
| **Service constructor** | Accepts `*pgxpool.Pool` directly — `NewXxxService(pool)` |
| **Service signature** | Always accept `context.Context` as the first argument |
| **Error handling** | Services return errors; handlers translate them to HTTP responses |
| **No global state** | Everything lives on `App` or is injected via constructor |

---

## Running the project

```sh
docker compose up -d
# Standard
go run ./cmd/api

# With air (hot-reload, uses .air.toml)
air

# Smoke test
curl http://localhost:8080/health

# Swagger UI
open http://localhost:8080/swagger/
```

---

## Roadmap patterns (what comes next)

- **cmd/cli/** — standalone CLI commands (user creation, data exports, cron jobs).
- **cmd/migrate/** — database migration runner using `golang-migrate` or similar.
- **internal/services/services_events.go** — background job queue (async email, webhooks, reports).
- **internal/test/** — shared test fixtures, factories, and helpers.
- **AI integration** — replace simulated AI scoring/risk analysis/copilot with real ML service calls.
- **CRM pipeline** — full deal pipeline with stages, activity tracking, and reporting.
- **Telematics pipeline** — real-time stream processing for vehicle telemetry data.
- **Workflow engine** — a real low-code automation engine (e.g., Temporal) replacing simulated workflow execution.
- **IoT/device management** — real IoT device registry and telemetry ingestion for the `iot_devices` table.
