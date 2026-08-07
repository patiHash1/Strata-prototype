<div align="center">

<img src="https://avatars.githubusercontent.com/u/233108473?s=200&v=4" alt="Strata Logo" width="120" onerror="this.style.display='none'">

# ⛰️ Strata

### The Open-Source ERP‑CRM Hybrid — Built for the Frontier

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge)](LICENSE)
[![Modules](https://img.shields.io/badge/Modules-25%2F25-brightgreen?style=for-the-badge)]()

<br>

> **Strata** is a modular, AI‑native ERP‑CRM platform — a direct open‑source competitor to Odoo.  
> Built in **Go** with a clean layered architecture, it ships with **25 fully‑implemented business modules**  
> spanning CRM, Finance, Supply Chain, HR, and Platform AI.

</div>

---

## 📦 Project Structure

<details open>
<summary><b>Click to expand/collapse</b></summary>

```
cmd/
├── api/main.go          # Entry point. Wires config, database, services, and starts the server.
└── cli/                 # CLI commands (future — admin tasks, data imports, etc.)

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
│   ├── services_super_admin.go # SuperAdminService: observability, SOC, maintenance, CI health
│   └── services_mailer.go      # Mailer: transactional email (stub)
│
├── handlers/            # HTTP LAYER — handlers, routes, App wiring
│   ├── handlers_server.go             # App struct, New(), Serve() — all dependency injection
│   ├── handlers_routes.go             # Route registration + middleware stack + Swagger UI
│   ├── handlers_health.go             # GET /health
│   ├── handlers_auth.go               # POST auth/register, POST auth/login
│   ├── handlers_account.go            # GET/PATCH/DELETE /api/v1/account, GET /api/v1/account/organizations
│   ├── handlers_org.go                # POST org/invitations, org/roles, org/api-keys
│   ├── handlers_billing.go            # POST billing/subscriptions
│   ├── handlers_crm.go                # POST crm/leads, quotes/risk-analysis, crm/tickets, crm/field-visits, crm/campaigns, crm/campaigns/{id}/launch
│   ├── handlers_accounting.go         # POST accounting/journal-entries, invoices/ocr, expenses, assets, tax-rates, tax/calculate; GET assets/{id}/depreciation; bank statements, exchange rates, currency convert
│   ├── handlers_accounting_extra.go   # Bank reconciliation, exchange rates, currency conversion
│   ├── handlers_supplychain.go        # POST fleet/telematics, fleet/routes, manufacturing/boms, manufacturing/work-orders, procurement/purchase-orders; GET inventory/reorder-predictions, procurement/supplier-risk; inventory receive/issue/transfer/snapshot
│   ├── handlers_supplychain_extra.go  # Inventory receive/issue/transfer/snapshot
│   ├── handlers_hr.go                 # POST hr/attendance/clock-in, hr/ats/parse-resume, hr/knowledge/search, hr/employees, hr/payroll/runs; GET hr/employees, hr/employees/{id}, hr/payroll/runs, hr/payroll/runs/{id}; PATCH hr/employees/{id}; clock-out, shifts/templates, shifts/assignments, shifts/predictions, shifts/schedule, payroll/detail, payroll/tax-profiles
│   ├── handlers_hr_extra.go           # Clock-out, shift management, tax profiles, payroll detail
│   ├── handlers_platform.go           # POST ai/copilot/query, workflows/trigger, bi/dashboards, iot/devices, iot/readings; GET security/audit-anomalies, bi/dashboards, bi/dashboards/{id}/data, iot/devices; iot/readings/batch
│   ├── handlers_platform_extra.go     # Batch IoT reading ingestion
│   └── handlers_super_admin.go        # Super-admin: metrics, health, maintenance, SOC SSE stream, user/org CRUD
│
├── utils/               # SHARED HELPERS — no business logic
│   ├── response.go      # WriteJSON, WriteErr, Envelope type
│   ├── middleware.go     # RequireAuth, RequirePermission, RequireAPIKey, Logging, CORS, Recovery, PartitionedMaintenance
│   └── validator.go     # IsEmail, IsDomainSlug, NotBlank, MinLen
│
├── config/config.go     # App configuration loaded from env vars
├── database/
│   ├── database.go      # pgx connection pool with retry logic
│   ├── migrations.go    # Embedded migration loader (embed.FS)
│   └── migrations/      # 71 numbered .up.sql migration files
└── env/env.go           # Safe environment variable helpers (GetString, GetInt, GetBool)

docs/                    # Auto-generated Swagger/OpenAPI spec (do not edit)
```

</details>

### 🔗 Package Dependency Flow

```mermaid
graph TD
    A[cmd/api/main.go] --> B[internal/config]
    A --> C[internal/database]
    A --> D[internal/services]
    A --> E[internal/handlers]
    A --> F[internal/utils]
    D --> C
    E --> D
    E --> F
    E --> B
    E --> C
    style A fill:#00ADD8,stroke:#00ADD8,color:#fff
    style D fill:#7B42BC,stroke:#7B42BC,color:#fff
    style E fill:#7B42BC,stroke:#7B42BC,color:#fff
    style F fill:#2EA043,stroke:#2EA043,color:#fff
```

---

## 📘 API Documentation (Swagger)

The project uses [swaggo/swag](https://github.com/swaggo/swag) to generate an OpenAPI 2.0 spec from Go annotations, served via a Swagger UI.

### 🔍 Viewing the Docs

```sh
# Start the server (Swagger UI is enabled by default)
ENABLE_SWAGGER=true go run ./cmd/api

# Open in your browser:
#   http://localhost:8080/swagger/
```

### ✍️ Annotating a New Endpoint

Add Go comments above your handler function in `internal/handlers/`:

<details>
<summary><b>📄 Example annotation</b></summary>

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

</details>

### 🔄 Regenerating the Spec

```sh
swag init --dir ./cmd/api,./internal/handlers --output ./docs --parseDependency --parseInternal
```

> ℹ️ The `docs/` directory is gitignored and regenerated on demand. CI should run `swag init` before building to ensure the spec is always fresh.

### ⚙️ Config

| Env Variable | Default | Description |
|:---|:---|:---|
| `PORT` | `8080` | HTTP server port |
| `ENABLE_SWAGGER` | `true` | Set to `false` in production to disable the Swagger UI route |
| `JWT_SECRET` | `dev-secret-change-in-production` | HMAC signing key for JWTs |
| `JWT_ISSUER` | `strata` | JWT issuer claim |
| `DATABASE_URL` | `""` | Postgres connection string |
| `REDIS_ADDR` | `""` | Redis address (optional — enables multi-node cache sync & SSE fan-out) |
| `REDIS_PASSWORD` | `""` | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `SUPERADMIN_UNAME` | `admin@strata.local` | Default super-admin email (seeded on first run) |
| `SUPERADMIN_PWORD` | `SuperAdmin123!` | Default super-admin password (seeded on first run) |

---

## ➕ Adding a New Endpoint

### ① Create a Handler File in `internal/handlers/`

<details>
<summary><b>📄 Example: <code>handlers_users.go</code></b></summary>

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

</details>

> 📁 **File naming:** `handlers_<category>.go` — one file per endpoint category (auth, org, billing, etc.)

### ② Register the Route in `internal/handlers/handlers_routes.go`

```go
func (a *App) routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.healthHandler)
	mux.HandleFunc("GET /users", a.listUsersHandler)   // <-- add here

	// ... protected routes with middleware ...
}
```

### ③ Add Permission‑Gated Routes

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

## 🧩 Adding a New Feature (Service Layer)

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

<details>
<summary><b>📄 Example: <code>services_crm.go</code></b></summary>

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

</details>

### 🔌 Wire the New Service

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

> ℹ️ Some services may require additional dependencies. For example, `SupplyChainService` also accepts `*AuthService` for API key bcrypt verification.

### 🎯 Use the Service from a Handler

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

> 💡 This keeps handlers **thin** — they parse input, call a service, write output. Business logic stays testable without HTTP.

---

## 🛠️ Adding a Utility or Helper

Shared utilities live in `internal/utils/` and have **no dependencies** on other project packages:

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

## 🧱 Adding Middleware

### ① Write the Middleware Function in `internal/utils/middleware.go`

```go
func rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Rate-limit logic…
		next.ServeHTTP(w, r)
	})
}
```

### ② Wire It in `internal/handlers/handlers_routes.go`

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

> 🔄 **Middleware runs outside‑in.** The first wrapped middleware runs outermost (last to run).

### 🔐 Per‑Route Middleware

Auth and permission checks are applied per‑route. Strata supports two auth modes:

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

## 🗄️ Adding a New Database Table

### ① Create a Migration File in `internal/database/migrations/`

Migrations are numbered `.up.sql` files loaded via Go's `embed.FS`. Create a new file with the next available sequence number:

```sql
-- internal/database/migrations/000068_create_leads.up.sql
CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'new',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

> 📋 Files are sorted lexicographically by filename, so the `000068_` prefix guarantees execution order.

### ② Add Service + Repository in `internal/services/services_<domain>.go`

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

### ③ Wire in `handlers_server.go` and `cmd/api/main.go`

Add the field to `App`, add the constructor parameter, create the service in main.

---

## ⚙️ Configuration Conventions

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

## 📬 Response Conventions

| Type | Format | Example |
|:---|:---|:---|
| ✅ **Success** | `{"key": value}` | `{"user": {...}}` |
| ❌ **Error** | `{"error": "message"}` | `{"error": "not found"}` |
| 📄 **Pagination** | `{"key": [...], "pagination": {...}}` | `{"users": [...], "pagination": {...}}` |

> 💡 **Always** use `utils.WriteJSON` / `utils.WriteErr` with the `utils.Envelope` type.

---

## 📐 Project Conventions

| Convention | Guidance |
|:---|:---|
| 🐹 **Go version** | 1.26 (match `go.mod`) |
| 📦 **Services package** | `internal/services/` — types, repos, business logic |
| 🌐 **Handlers package** | `internal/handlers/` — HTTP handlers, routes, App struct |
| 🔧 **Utils package** | `internal/utils/` — response helpers, middleware, validators |
| 📁 **File naming** | `services_<domain>.go`, `handlers_<category>.go` |
| ✍️ **Handler signature** | `func (a *App) actionHandler(w http.ResponseWriter, r *http.Request)` |
| 🏗️ **Service constructor** | Accepts `*pgxpool.Pool` directly — `NewXxxService(pool)` |
| 🧵 **Service signature** | Always accept `context.Context` as the first argument |
| ⚠️ **Error handling** | Services return errors; handlers translate them to HTTP responses |
| 🚫 **No global state** | Everything lives on `App` or is injected via constructor |

---

## 📊 Module Coverage

<div align="center">

### 🟢 **27 / 27 Modules Implemented**

</div>

| # | Category | Module | Status |
|:---:|:---|:---|:---:|
| 0.1 | ⚪ Account | User Profile & Account Management | ✅ |
| 1.1 | 🟣 CRM & RevOps | Sales / Lead Score | ✅ |
| 1.2 | 🟣 CRM & RevOps | Quotes / Contract Risk | ✅ |
| 1.3 | 🟣 CRM & RevOps | Helpdesk / Ticket Router | ✅ |
| 1.4 | 🟣 CRM & RevOps | Field Sales Dispatch | ✅ |
| 1.5 | 🟣 CRM & RevOps | Campaign Engines | ✅ |
| 2.1 | 🟢 Finance & AP | Double‑Entry Ledger & Bank Rec | ✅ |
| 2.2 | 🟢 Finance & AP | Invoice OCR | ✅ |
| 2.3 | 🟢 Finance & AP | Expense / Fraud Detection | ✅ |
| 2.4 | 🟢 Finance & AP | Fixed Assets | ✅ |
| 2.5 | 🟢 Finance & AP | Multi‑Currency Exchange & Tax | ✅ |
| 3.1 | 🔵 Supply & Fleet | Multi‑Warehouse Stock & AI Reorder | ✅ |
| 3.2 | 🔵 Supply & Fleet | BOM & Work Orders | ✅ |
| 3.3 | 🔵 Supply & Fleet | Fleet Telematics | ✅ |
| 3.4 | 🔵 Supply & Fleet | Route Optimizer | ✅ |
| 3.5 | 🔵 Supply & Fleet | Vendor / Supplier Risk | ✅ |
| 4.1 | 🟠 HR & Talent | Core HR / Employee Portal | ✅ |
| 4.2 | 🟠 HR & Talent | Time, Attendance & Shift Predict | ✅ |
| 4.3 | 🟠 HR & Talent | Payroll, Tax Withholding & Disbursements | ✅ |
| 4.4 | 🟠 HR & Talent | ATS / Candidate Matcher | ✅ |
| 4.5 | 🟠 HR & Talent | Knowledge Base / RAG | ✅ |
| 5.1 | 🔴 Platform & AI | Text‑to‑SQL Copilot | ✅ |
| 5.2 | 🔴 Platform & AI | BI & Dashboards | ✅ |
| 5.3 | 🔴 Platform & AI | Low‑Code Workflows | ✅ |
| 5.4 | 🔴 Platform & AI | IoT Gateway & Batch Ingestion | ✅ |
| 5.5 | 🔴 Platform & AI | Audit / Security / RBAC | ✅ |
| 6.1 | ⚪ Super Admin | System Observability & SOC | ✅ |
| 6.2 | ⚪ Super Admin | Partitioned Maintenance | ✅ |
| 6.3 | ⚪ Super Admin | CI Health & Module Scores | ✅ |
| 6.4 | ⚪ Super Admin | User & Org CRUD (Ban/Suspend) | ✅ |

---

## 🚀 Running the Project

```sh
# Start PostgreSQL
docker compose up -d

# Standard (reads PORT from env, defaults to 8080)
go run ./cmd/api

# With air (hot-reload, uses .air.toml)
air

# Custom port (for cloud deployments like Railway)
PORT=3000 go run ./cmd/api

# Smoke test
curl http://localhost:8080/health

# Swagger UI
open http://localhost:8080/swagger/
```

> ☁️ **Cloud deployment:** The server binds to `$PORT` (default `8080`). Railway and similar platforms set `PORT` automatically — no hardcoded port strings.

---

## 🗺️ Roadmap

<div align="center">

| Status | What's Next |
|:---:|:---|
| 🚧 | **`cmd/cli/`** — standalone CLI commands (user creation, data exports, cron jobs) |
| 🚧 | **`internal/services/services_events.go`** — background job queue (async email, webhooks, reports) |
| 🚧 | **`internal/test/`** — shared test fixtures, factories, and helpers |
| 🚧 | **Real AI/ML integration** — replace simulated AI with real ML service calls |
| 🚧 | **Real‑time BI dashboards** — replace simulated dashboard data with real‑time analytics |
| 🚧 | **Vector RAG** — replace ILIKE search with pgvector embedding‑based semantic search |
| 🚧 | **Stripe integration** — replace simulated billing with real Stripe API calls |
| 🚧 | **Webhook support** — add webhook delivery for workflow actions |

</div>

<details>
<summary><b>✅ Completed milestones</b></summary>

- ~~CRM pipeline~~ — full deal pipeline with field sales visits, campaigns, tickets
- ~~Telematics pipeline~~ — real‑time stream processing for vehicle telemetry data
- ~~Workflow engine~~ — low‑code automation engine with event‑driven triggers
- ~~IoT/device management~~ — IoT device registry and telemetry ingestion
- ~~Bank reconciliation~~ — automated bank statement matching and reconciliation
- ~~Multi‑currency exchange rates~~ — real‑time currency conversion and exchange rate management
- ~~Inventory levels per warehouse~~ — multi‑warehouse stock tracking with receive/issue/transfer/snapshot
- ~~Shift management & AI prediction~~ — shift templates, assignments, scheduling, and AI‑driven predictions
- ~~Payroll tax withholding per employee~~ — per‑employee tax profiles, withholding calculations, and payroll detail
- ~~Super‑admin subsystem~~ — system observability, SOC security monitoring, partitioned maintenance, CI health ingestion, user/org CRUD

</details>

---

<div align="center">

### 🏔️ Built with Go · PostgreSQL · Docker

<sub>Strata — The Open‑Source ERP‑CRM for the Frontier</sub>

</div>
