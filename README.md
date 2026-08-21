<div align="center">

<img src="https://avatars.githubusercontent.com/u/233108473?s=200&v=4" alt="Strata Logo" width="120" onerror="this.style.display='none'">

# ⛰️ Strata

### The Open-Source ERP‑CRM Hybrid — Built for the Frontier

[![Go Version](https://img.shields.io/badge/Go-1.26-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7-DC382D?style=for-the-badge&logo=redis&logoColor=white)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)
[![License](https://img.shields.io/badge/License-MIT-yellow?style=for-the-badge)](LICENSE)

<br>

> **Strata** is a modular, AI‑native ERP‑CRM platform — a direct open‑source competitor to Odoo.
> Built in **Go** with a clean three‑layer architecture, it ships with **30+ fully‑implemented business modules**
> spanning CRM, Finance, Supply Chain, HR, and Platform AI, plus a browser‑based Super Admin dashboard.

</div>

---

## ✨ Highlights

- **API‑first, multi‑tenant** — every resource is scoped to an organization via JWT claims
- **30+ business modules** — CRM, Accounting, Supply Chain & Fleet, HR, and Platform AI
- **AI‑native (simulated)** — lead scoring, contract risk, sentiment routing, reorder predictions, shift forecasting, text‑to‑SQL copilot
- **Super Admin dashboard** — real‑time SOC security feed (SSE), Prometheus metrics, partitioned maintenance control panel, and platform‑wide user/org management
- **Version‑tracked migrations** — embedded `.up.sql` files applied once and recorded in `schema_migrations`
- **Optional Redis** — multi‑node maintenance sync and SSE fan‑out

---

## 🧰 Tech Stack

| Layer | Technology |
|:---|:---|
| **Language** | Go 1.26 |
| **HTTP Router** | `net/http` (Go 1.22+ enhanced ServeMux with path params) |
| **Database** | PostgreSQL 16 via `pgx/v5` |
| **Cache & Pub/Sub** | Redis 7 via `go-redis/v9` (optional) |
| **Auth** | JWT (HS256) via `golang-jwt/jwt/v5` + bcrypt |
| **Server‑side UI** | templ (`github.com/a-h/templ`) |
| **API Docs** | Swagger / OpenAPI 2.0 via `swaggo/swag` |
| **Containerization** | Docker Compose (PostgreSQL + Redis) |
| **Hot Reload** | Air (`.air.toml`) |

---

## 📦 Repository Layout

```
cmd/
├── api/main.go                    # Entry point — wires config, DB, services, starts server
└── cli/
    └── publish-soc-events/main.go # Publish mock SOC events to Redis for testing

internal/
├── config/config.go               # Typed Config loaded from env vars
├── env/env.go                     # Safe env getters (GetString/GetInt/GetBool)
├── logger/logger.go               # JSON structured logging wrapper (log/slog)
├── database/
│   ├── database.go                # pgx pool + version-tracked migration runner
│   ├── migrations.go              # Embedded migration loader (embed.FS)
│   └── migrations/                # 74 numbered .up.sql + .down.sql files
├── services/                      # Business layer — service + repository per domain
│   ├── services_auth.go           # JWT, bcrypt, refresh tokens
│   ├── services_users.go          # Users, organization memberships
│   ├── services_orgs.go           # Organizations, invitations, API keys
│   ├── services_rbac.go           # Roles, permissions
│   ├── services_billing.go        # Subscriptions
│   ├── services_crm.go            # CRM: leads, deals, quotes, AI analysis
│   ├── services_accounting.go     # Accounting: ledger, OCR, expenses, bank rec, multi-currency
│   ├── services_supplychain.go    # Fleet, telematics, inventory, routes
│   ├── services_hr.go             # HR: attendance, shifts, payroll/tax, ATS, knowledge search
│   ├── services_platform.go       # AI copilot, workflows, IoT, security anomalies
│   ├── services_super_admin.go    # Observability, SOC, maintenance, CI health
│   ├── services_registration.go   # Atomic org+owner+role+membership registration
│   ├── services_seed.go           # Idempotent super-admin seeding
│   └── services_mailer.go         # Transactional email (stub)
├── handlers/                      # HTTP layer — handlers, routes, App wiring
│   ├── handlers_server.go         # App struct, DI wiring, Serve()
│   ├── handlers_routes.go         # Route registration + middleware stack
│   ├── handlers_health.go         # GET /health
│   ├── handlers_auth.go           # POST auth/register, auth/login
│   ├── handlers_account.go        # GET/PATCH/DELETE /api/v1/account
│   ├── handlers_org.go            # Org endpoints
│   ├── handlers_billing.go        # Billing endpoints
│   ├── handlers_crm.go            # CRM endpoints
│   ├── handlers_accounting.go     # Accounting endpoints
│   ├── handlers_accounting_extra.go
│   ├── handlers_supplychain.go    # Fleet & inventory endpoints
│   ├── handlers_supplychain_extra.go
│   ├── handlers_hr.go             # HR endpoints
│   ├── handlers_hr_extra.go
│   ├── handlers_platform.go       # AI & Platform endpoints
│   ├── handlers_platform_extra.go
│   └── handlers_super_admin.go    # Super-admin: login, dashboard, metrics, SOC, user/org CRUD
├── templates/                     # templ components for the super-admin dashboard
├── static/                        # Embedded static assets (CSS)
└── utils/                         # Shared helpers (no business logic)
    ├── response.go                # WriteJSON, WriteErr, Envelope
    ├── middleware.go              # RequireAuth, RequireAuthCookie, RequirePermission, etc.
    ├── ratelimit.go               # RateLimitMiddleware
    ├── csrf.go                    # CSRF validation (double-submit cookie)
    ├── totp.go                    # TOTP/MFA code validation
    └── validator.go               # IsEmail, IsDomainSlug, NotBlank, MinLen

docs/                              # Auto-generated Swagger/OpenAPI spec (swag init)
documentation/                     # Human-readable docs (MkDocs / Read the Docs)
scripts/seed.go                    # Standalone dummy-data seed/wipe tool
tests/                             # Integration tests (httptest)
```

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

## 🚀 Quickstart

### Prerequisites

- **Go 1.26+**
- **Docker** + **Docker Compose** (PostgreSQL + Redis)
- **Make** (optional — wraps common tasks)

### 1. Clone & start infrastructure

```bash
git clone https://github.com/patiHash1/Strata-prototype.git
cd Strata-prototype

# Start PostgreSQL + Redis
docker compose up -d
```

### 2. Configure environment

Create a `.env` file in the project root. Two variables are **required** — the server panics at startup if either is missing:

```bash
# .env
PORT=8080
DATABASE_URL=postgres://strata-user:strata-pass@localhost:5432/strata-db-beta?sslmode=disable
JWT_SECRET=change-me-to-a-long-random-string
JWT_ISSUER=strata
ENABLE_SWAGGER=true
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=strata-redis-pass
ALLOWED_ORIGINS=http://localhost:8080
SUPERADMIN_UNAME=admin@strata.local
SUPERADMIN_PWORD=SuperAdmin123!
```

See [Environment Variables](documentation/environment-variables.md) for the full reference.

### 3. Run the server

```bash
# Migrations run automatically on startup
go run ./cmd/api

# Or with hot-reload (installs air + templ, checks Redis, runs air)
make dev
```

### 4. Verify

```bash
curl http://localhost:8080/health
# {"status":"ok","database":"connected","redis":"connected"}

# Swagger UI
open http://localhost:8080/swagger/

# Super Admin dashboard
open http://localhost:8080/api/v1/super-admin/login
```

### Database migrations & seeding

Migrations are **applied automatically at startup** and tracked in the `schema_migrations` table — there is no separate migration command. Default currencies, plans, permissions, and the super-admin are seeded by migrations.

For development dummy data (20 records per table):

```bash
make data-seed   # go run ./scripts/seed.go seed
make data-wipe   # go run ./scripts/seed.go wipe
```

---

## 📘 API Documentation

The project uses [swaggo/swag](https://github.com/swaggo/swag) to generate an OpenAPI 2.0 spec from Go annotations, served via a Swagger UI at `/swagger/` (enabled by default; disable with `ENABLE_SWAGGER=false`).

**Regenerating the spec:**

```bash
swag init -g cmd/api/main.go -o docs
# or via make
make swagger
```

> ℹ️ The `docs/` directory is committed to the repository and regenerated with `make swagger` whenever handlers change.

**Human-readable API reference** lives in [`documentation/`](documentation/index.md) and covers every module:

| Module | Doc |
|:---|:---|
| Authentication | [`api/authentication.md`](documentation/api/authentication.md) |
| Account | [`api/account.md`](documentation/api/account.md) |
| Organizations | [`api/organizations.md`](documentation/api/organizations.md) |
| Members | [`api/members.md`](documentation/api/members.md) |
| Billing | [`api/billing.md`](documentation/api/billing.md) |
| CRM & Revenue | [`api/crm.md`](documentation/api/crm.md) |
| Accounting | [`api/accounting.md`](documentation/api/accounting.md) |
| Fleet & Telematics | [`api/fleet.md`](documentation/api/fleet.md) |
| Inventory | [`api/inventory.md`](documentation/api/inventory.md) |
| HR & Workforce | [`api/hr.md`](documentation/api/hr.md) |
| AI & Platform | [`api/platform-ai.md`](documentation/api/platform-ai.md) |
| Super Admin | [`api/super-admin.md`](documentation/api/super-admin.md) |
| System | [`api/system.md`](documentation/api/system.md) |

---

## 🌙 Super Admin Dashboard

The super-admin dashboard is a browser‑based HTML UI for platform administration with real‑time observability. Login at `/api/v1/super-admin/login` using the `SUPERADMIN_UNAME` / `SUPERADMIN_PWORD` credentials.

### Routes

| Route | Method | Description |
|:---|:---:|:---|
| `/api/v1/super-admin/login` | GET / POST | Login page / authenticate (sets `strata_token` cookie) |
| `/api/v1/super-admin/logout` | POST | Clear session cookie and redirect to login |
| `/api/v1/super-admin/dashboard` | GET | Protected dashboard (HTML) |
| `/api/v1/super-admin/dashboard/kpis` | GET | KPI cards (JSON) |
| `/api/v1/super-admin/dashboard/activity` | GET | Recent activity feed (JSON) |
| `/api/v1/super-admin/dashboard/traffic` | GET | HTTP traffic series (JSON) |
| `/api/v1/super-admin/metrics` | GET | System telemetry page (HTML) |
| `/api/v1/super-admin/metrics/json` | GET | System telemetry (JSON) |
| `/api/v1/super-admin/metrics/fragment` | GET | Metrics grid HTML fragment (HTMX polling) |
| `/api/v1/super-admin/metrics/prometheus` | GET | System telemetry (Prometheus text format) |
| `/api/v1/super-admin/health` | GET | Module health scores (JSON) |
| `/api/v1/super-admin/security` | GET | Security & Audit page (HTML) |
| `/api/v1/super-admin/security/stream` | GET | Real‑time SOC events (SSE stream) |
| `/api/v1/super-admin/settings` | GET | Settings page (HTML) |
| `/api/v1/super-admin/users` | GET | Global user directory page (HTML) |
| `/api/v1/super-admin/users/json` | GET | List all users (JSON) |
| `/api/v1/super-admin/organizations` | GET | Global org directory page (HTML) |
| `/api/v1/super-admin/organizations/json` | GET | List all organizations (JSON) |
| `/api/v1/super-admin/maintenance/rules` | GET | Maintenance rules page (HTML) |
| `/api/v1/super-admin/maintenance/fragment` | GET | Maintenance rules table fragment (HTMX) |
| `/api/v1/super-admin/maintenance` | POST | Create a maintenance rule |
| `/api/v1/super-admin/maintenance/{id}` | DELETE | Revoke/deactivate a maintenance rule |

### Module Health Endpoints

Every module exposes a **public** health endpoint at `GET /api/v1/{module}/health`:

| Status | HTTP Code | Meaning |
|:---|:---:|:---|
| `operational` | `200` | Module is healthy |
| `degraded` | `200` | Module operational but one or more features under maintenance |
| `maintenance` | `503` | Entire module under maintenance |

```sh
curl http://localhost:8080/api/v1/crm/health
# {"module":"crm","status":"operational","healthy":true}
```

**Available modules:** `crm`, `accounting`, `hr`, `billing`, `account`, `org`, `ai`, `bi`, `workflows`, `fleet`, `inventory`, `manufacturing`, `procurement`, `iot`, `security`

### Testing the dashboard

```sh
# 1. Start infrastructure
docker compose up -d

# 2. Start the server (with Redis configured)
go run ./cmd/api

# 3. Open the dashboard
open http://localhost:8080/api/v1/super-admin/login

# 4. Publish mock SOC events to test the live security feed
make testsoc
# or
go run ./cmd/cli/publish-soc-events -count 5 -redis localhost:6379 -redis-pass strata-redis-pass
```

> **Note:** Redis is required for the live security feed and multi‑node maintenance sync. Without `REDIS_ADDR`, the SSE connection works but no events are delivered.

---

## 🧑‍💻 Development

### Adding a new endpoint

1. Create a handler in `internal/handlers/handlers_<category>.go`
2. Register the route in `internal/handlers/handlers_routes.go` with the appropriate middleware
3. Add a permission constant in `internal/services/services_rbac.go` and seed it in `internal/database/migrations/000067_seed_default_permissions.up.sql` (if needed)
4. Regenerate Swagger: `make swagger`

See [Adding Endpoints](documentation/development/adding-endpoints.md) for the full walkthrough.

### Adding a new service

1. Create `internal/services/services_<domain>.go` (types + repository + service)
2. Add the field to `App` and the constructor parameter in `internal/handlers/handlers_server.go`
3. Wire it in `cmd/api/main.go`
4. Add a migration in `internal/database/migrations/` if a new table is needed

See [Adding Services](documentation/development/adding-services.md) and [Adding Migrations](documentation/development/adding-migrations.md).

### Adding middleware

Global middleware is stacked in `handlers_routes.go` (outermost first):

```go
var handler http.Handler = mux
handler = utils.MaxBodySizeMiddleware(1 << 20)(handler)                          // 1 MiB body limit
handler = utils.CORSMiddleware(a.Config.AllowedOrigins)(handler)                 // CORS
handler = utils.LoggingMiddleware(a.SuperAdmin)(handler)                         // logging + latency metrics
handler = utils.RecoveryMiddleware(a.SuperAdmin)(handler)                        // panic recovery
handler = utils.PartitionedMaintenanceMiddleware(a.SuperAdmin)(handler)          // maintenance enforcement
```

Route‑level auth supports three modes:

| Mode | Middleware | Header / Cookie | Claims Helper |
|:---|:---|:---|:---|
| Bearer token (JWT) | `RequireAuth` + `RequirePermission` | `Authorization: Bearer <jwt>` | `utils.GetClaims(r)` |
| Cookie + header | `RequireAuthCookie` + `RequirePermission` | `strata_token` cookie OR Bearer header | `utils.GetClaims(r)` |
| API key | `RequireAPIKey` | `X-API-Key: <key>` | `utils.GetAPIKeyClaims(r)` |

---

## ⚙️ Configuration

Configuration is loaded from environment variables (optionally via a `.env` file). See [Environment Variables](documentation/environment-variables.md) for the full reference.

| Variable | Required | Default | Description |
|:---|:---:|:---|:---|
| `DATABASE_URL` | ✅ | — | PostgreSQL connection string (server panics if missing) |
| `JWT_SECRET` | ✅ | — | HMAC signing key (server panics if missing) |
| `PORT` | — | `8080` | HTTP port |
| `ENABLE_SWAGGER` | — | `true` | Serve Swagger UI at `/swagger/` |
| `JWT_ISSUER` | — | `strata` | JWT `iss` claim |
| `ALLOWED_ORIGINS` | — | *(empty)* | Comma‑separated CORS origins (empty blocks cross‑origin) |
| `SUPERADMIN_UNAME` | — | *(empty)* | Super‑admin email (seeded on startup when paired with `SUPERADMIN_PWORD`) |
| `SUPERADMIN_PWORD` | — | *(empty)* | Super‑admin password |
| `REDIS_ADDR` | — | *(empty)* | Redis address (empty disables Redis) |
| `REDIS_PASSWORD` | — | *(empty)* | Redis password |
| `REDIS_DB` | — | `0` | Redis database number |

---

## 🧪 Testing & Quality Assurance

```bash
# Run all tests
make test          # go test ./... -count=1

# With race detection
go test -race ./...

# Format & vet
go fmt ./... && go vet ./...
```

Integration tests live in `tests/` (e.g. `tests/handlers_super_admin_test.go`) and use `httptest` against the real route handler.

---

## 📊 Module Coverage

<div align="center">

### 🟢 **30+ Modules Implemented**

</div>

| # | Category | Module |
|:---:|:---|:---|
| 0.1 | ⚪ Account | User Profile & Account Management |
| 1.1 | 🟣 CRM & RevOps | Sales / Lead Score |
| 1.2 | 🟣 CRM & RevOps | Quotes / Contract Risk |
| 1.3 | 🟣 CRM & RevOps | Helpdesk / Ticket Router |
| 1.4 | 🟣 CRM & RevOps | Field Sales Dispatch |
| 1.5 | 🟣 CRM & RevOps | Campaign Engines |
| 2.1 | 🟢 Finance & AP | Double‑Entry Ledger & Bank Rec |
| 2.2 | 🟢 Finance & AP | Invoice OCR |
| 2.3 | 🟢 Finance & AP | Expense / Fraud Detection |
| 2.4 | 🟢 Finance & AP | Fixed Assets |
| 2.5 | 🟢 Finance & AP | Multi‑Currency Exchange & Tax |
| 3.1 | 🔵 Supply & Fleet | Multi‑Warehouse Stock & AI Reorder |
| 3.2 | 🔵 Supply & Fleet | BOM & Work Orders |
| 3.3 | 🔵 Supply & Fleet | Fleet Telematics |
| 3.4 | 🔵 Supply & Fleet | Route Optimizer |
| 3.5 | 🔵 Supply & Fleet | Vendor / Supplier Risk |
| 4.1 | 🟠 HR & Talent | Core HR / Employee Portal |
| 4.2 | 🟠 HR & Talent | Time, Attendance & Shift Predict |
| 4.3 | 🟠 HR & Talent | Payroll, Tax Withholding & Disbursements |
| 4.4 | 🟠 HR & Talent | ATS / Candidate Matcher |
| 4.5 | 🟠 HR & Talent | Knowledge Base / RAG |
| 5.1 | 🔴 Platform & AI | Text‑to‑SQL Copilot |
| 5.2 | 🔴 Platform & AI | BI & Dashboards |
| 5.3 | 🔴 Platform & AI | Low‑Code Workflows |
| 5.4 | 🔴 Platform & AI | IoT Gateway & Batch Ingestion |
| 5.5 | 🔴 Platform & AI | Audit / Security / RBAC |
| 6.1 | ⚪ Super Admin | System Observability & SOC |
| 6.2 | ⚪ Super Admin | Partitioned Maintenance |
| 6.3 | ⚪ Super Admin | CI Health & Module Scores |
| 6.4 | ⚪ Super Admin | User & Org CRUD (Ban/Suspend) |
| 6.5 | ⚪ Super Admin | Module Health Endpoints |
| 6.6 | ⚪ Super Admin | Maintenance Control Panel (UI) |
| 6.7 | ⚪ Super Admin | SOC Event Persistence & Retention |

---

## 🛠️ CLI & Make Targets

### CLI Commands

| Command | Description |
|:---|:---|
| `go run ./cmd/cli/publish-soc-events` | Publish mock SOC security events to Redis for testing the live feed |

```sh
go run ./cmd/cli/publish-soc-events -count 5 -redis localhost:6379 -redis-pass strata-redis-pass
```

### Make Targets

| Target | Description |
|:---|:---|
| `make dev` | Start dev server with hot‑reload (installs tools, generates templ, checks Redis, runs air) |
| `make build` | Build the Go binary to `./tmp/main` |
| `make test` | Run all tests |
| `make swagger` | Regenerate Swagger spec |
| `make templ-generate` | Regenerate templ Go files |
| `make data-seed` | Seed dummy data (20 records/table) |
| `make data-wipe` | Wipe dummy data |
| `make testsoc` | Publish mock SOC events to Redis |
| `make check-redis` | Verify Redis connectivity |
| `make install-tools` | Install air + templ if missing |
| `make clean` | Remove build artifacts |

---

## 📚 Documentation

Full human‑readable documentation is maintained in [`documentation/`](documentation/index.md) and published via MkDocs / Read the Docs:

- **Getting Started** — [`getting-started.md`](documentation/getting-started.md)
- **Environment Variables** — [`environment-variables.md`](documentation/environment-variables.md)
- **Deployment & Operations** — [`deployment-and-operations.md`](documentation/deployment-and-operations.md)
- **Architecture** — [`architecture/overview.md`](documentation/architecture/overview.md)
- **Development Guides** — [`development/setup.md`](documentation/development/setup.md)

---

## 🤝 Contributing

1. **Fork** the repository and create a feature branch:
   - Features: `feature/<description>`
   - Bug fixes: `fix/<description>`
2. **Make your changes** — keep them focused and consistent with the [Code Conventions](documentation/development/conventions.md).
3. **Run the checks:**
   ```bash
   go fmt ./... && go vet ./... && go test ./...
   ```
4. **Open a pull request** with a clear description of the change and any relevant issue references.

Commit messages should use the imperative mood with a capitalized, ≤50‑character subject line.

---

## 🗺️ Roadmap

| Status | What's Next |
|:---:|:---|
| 🚧 | **Background job queue** — async email, webhooks, reports |
| 🚧 | **Shared test fixtures** — factories and helpers |
| 🚧 | **Real AI/ML integration** — replace simulated AI with real ML service calls |
| 🚧 | **Real‑time BI dashboards** — replace simulated data with live analytics |
| 🚧 | **Vector RAG** — replace ILIKE search with pgvector semantic search |
| 🚧 | **Stripe integration** — replace simulated billing with real Stripe API calls |
| 🚧 | **Webhook support** — webhook delivery for workflow actions |

---

<div align="center">

### 🏔️ Built with Go · PostgreSQL · Redis · Docker

<sub>Strata — The Open‑Source ERP‑CRM for the Frontier</sub>

</div>
