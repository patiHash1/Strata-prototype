# Getting Started

## What is Strata?

Strata is a multi-tenant ERP-CRM platform designed as a modern, API-first alternative to Odoo. It provides:

- **Organization management** with multi-tenant isolation
- **Role-based access control (RBAC)** with dynamic roles and permissions
- **User management** including invitation flows and membership lifecycle
- **CRM & Revenue Operations** with AI-powered lead scoring, contract risk analysis, and pipeline management
- **Finance & Enterprise Accounting** with double-entry ledger, bank reconciliation, multi-currency exchange rates, invoice OCR, and AI fraud-audited expenses
- **Supply Chain & Inventory** with multi-warehouse stock tracking, receive/issue/transfer/snapshot, BOM, work orders, fleet telematics, and route optimization
- **HR & Workforce** with time & attendance, shift management & AI prediction, payroll with per-employee tax withholding, ATS candidate matching, and knowledge base RAG
- **Platform & AI** with text-to-SQL copilot, BI dashboards, low-code workflows, IoT gateway with batch ingestion, and security audit
- **Super Admin** with system observability, real-time SOC monitoring (SSE), partitioned maintenance control panel (UI), per-module health endpoints, SOC event persistence with 25-day retention, CI health ingestion, and platform-wide user/org CRUD
- **Billing/subscription management** with Stripe integration
- **API key authentication** for machine-to-machine integrations

## Technology stack

| Layer | Technology |
|---|---|
| Language | **Go 1.26** |
| HTTP Router | **net/http** (Go 1.22+ enhanced ServeMux with path parameters) |
| Database | **PostgreSQL 16** via `pgx/v5` |
| Cache & Pub/Sub | **Redis 7** via `go-redis/v9` (optional — multi-node sync) |
| Authentication | **JWT** (HS256) with `golang-jwt/jwt/v5` |
| Password hashing | **bcrypt** via `golang.org/x/crypto` |
| Server-side UI | **templ** (`github.com/a-h/templ`) for the super-admin dashboard |
| API documentation | **Swagger/OpenAPI 2.0** via `swaggo/swag` |
| Containerization | **Docker Compose** (PostgreSQL + Redis) |
| Hot reload | **Air** (`.air.toml`) |

## Project structure

```
cmd/
├── api/main.go              # Entry point — wires everything
└── cli/
    └── publish-soc-events/main.go   # Publish mock SOC events to Redis for testing

internal/
├── config/config.go         # Configuration loaded from env vars
├── env/env.go               # Safe environment variable helpers
├── logger/logger.go         # JSON structured logging wrapper (log/slog)
├── database/
│   ├── database.go          # pgx connection pool + versioned migration runner
│   ├── migrations.go        # Embedded migration loader (embed.FS)
│   └── migrations/          # 74 numbered .up.sql + .down.sql files
├── services/                # Business logic + SQL, one file per domain
│   ├── services_auth.go       # JWT, bcrypt, refresh tokens
│   ├── services_users.go      # Users, organization memberships
│   ├── services_orgs.go       # Organizations, invitations, API keys
│   ├── services_rbac.go       # Roles, permissions
│   ├── services_billing.go    # Subscriptions
│   ├── services_crm.go        # CRM: leads, deals, quotes, AI analysis
│   ├── services_accounting.go # Accounting: journal entries, bank rec, multi-currency, invoices, expenses
│   ├── services_supplychain.go# Supply chain: fleet, telematics, inventory, routes
│   ├── services_hr.go         # HR: attendance, shifts, payroll/tax, resume parsing, knowledge search
│   ├── services_platform.go   # Platform: AI copilot, workflows, IoT batch, security anomalies
│   ├── services_super_admin.go# Super Admin: observability, SOC, maintenance, CI health
│   ├── services_registration.go # Atomic org+owner+role+membership registration
│   ├── services_seed.go       # Super-admin seeding
│   └── services_mailer.go     # Transactional email (stub)
├── handlers/                # HTTP handlers + routes + App struct
│   ├── handlers_server.go             # App struct, DI wiring, Serve()
│   ├── handlers_routes.go             # Route registration + middleware
│   ├── handlers_health.go             # GET /health
│   ├── handlers_auth.go               # POST auth/register, auth/login
│   ├── handlers_account.go            # GET/PATCH/DELETE /api/v1/account
│   ├── handlers_org.go                # Org endpoints
│   ├── handlers_billing.go            # Billing endpoints
│   ├── handlers_crm.go                # CRM endpoints
│   ├── handlers_accounting.go         # Accounting endpoints
│   ├── handlers_accounting_extra.go   # Bank reconciliation, exchange rates, currency conversion
│   ├── handlers_supplychain.go        # Fleet & inventory endpoints
│   ├── handlers_supplychain_extra.go  # Inventory receive/issue/transfer/snapshot
│   ├── handlers_hr.go                 # HR endpoints
│   ├── handlers_hr_extra.go           # Clock-out, shift management, tax profiles, payroll detail
│   ├── handlers_platform.go           # AI & Platform endpoints
│   ├── handlers_platform_extra.go     # Batch IoT reading ingestion
│   └── handlers_super_admin.go        # Super-admin: metrics, health, maintenance, SOC SSE, user/org CRUD
├── templates/               # templ components for the super-admin dashboard
├── static/                  # Embedded static assets (CSS)
└── utils/
    ├── response.go          # WriteJSON, WriteErr, Envelope
    ├── middleware.go        # RequireAuth, RequireAuthCookie, RequirePermission, etc.
    ├── ratelimit.go         # RateLimitMiddleware
    ├── csrf.go              # CSRF validation (double-submit cookie)
    ├── totp.go              # TOTP/MFA code validation
    └── validator.go         # Email, slug, string validators

docs/                        # Auto-generated Swagger spec (swag init)
documentation/               # Human-readable documentation (this)
scripts/seed.go              # Standalone dummy-data seed/wipe tool
tests/                       # Integration tests (httptest)
```

## Package dependency flow

```
cmd/api/main.go
    │
    ├── internal/config        (env → struct)
    ├── internal/database      (pgx pool + embedded versioned migrations)
    ├── internal/services      (business logic + SQL)
    │   └── depends on: database
    ├── internal/handlers      (HTTP handlers + App)
    │   └── depends on: services, utils, config, database, templates, static
    └── internal/utils         (pure helpers — no project deps)
```

## API Base URL

All API endpoints are served under `/api/v1/` unless otherwise noted.

| Environment | Base URL |
|---|---|
| Local development | `http://localhost:8080/api/v1` |
| Swagger UI | `http://localhost:8080/swagger/` |

## Response format

All responses use the `utils.Envelope` type (`map[string]any`):

**Success:**
```json
{
    "key": "value"
}
```

**Error:**
```json
{
    "error": "description of the problem"
}
```

## Configuration

Configuration is loaded from environment variables (optionally via a `.env` file in the project root). See [Environment Variables](environment-variables.md) for the full reference. The two **required** variables are:

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string (server panics if missing) |
| `JWT_SECRET` | HMAC signing key for JWTs (server panics if missing) |

## Running the project

```bash
# Start PostgreSQL + Redis
docker compose up -d

# Run the API server (migrations run automatically)
go run ./cmd/api

# With hot-reload (installs air + templ, checks Redis, then runs air)
make dev

# Smoke test
curl http://localhost:8080/health
```

See [Development Setup](development/setup.md) for the full quickstart and [Deployment & Operations](deployment-and-operations.md) for production guidance.
