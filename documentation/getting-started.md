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
- **Billing/subscription management** with Stripe integration
- **API key authentication** for machine-to-machine integrations

## Technology stack

| Layer | Technology |
|---|---|
| Language | **Go 1.26** |
| HTTP Router | **net/http** (Go 1.22+ enhanced ServeMux with path parameters) |
| Database | **PostgreSQL 16** via `pgx/v5` |
| Authentication | **JWT** (HS256) with `golang-jwt/jwt/v5` |
| Password hashing | **bcrypt** via `golang.org/x/crypto` |
| API documentation | **Swagger/OpenAPI 2.0** via `swaggo/swag` |
| Containerization | **Docker Compose** (PostgreSQL) |
| Hot reload | **Air** (`.air.toml`) |

## Project structure

```
cmd/
├── api/main.go              # Entry point — wires everything
└── cli/                     # (future) CLI commands
└── migrate/                 # (future) Migration runner

internal/
├── config/config.go         # Configuration loaded from env vars
├── env/env.go               # Safe environment variable helpers
├── database/database.go     # pgx connection pool + migrations
	├── services/
	│   ├── services_auth.go       # JWT, bcrypt, refresh tokens
	│   ├── services_users.go      # Users, organization memberships
	│   ├── services_orgs.go       # Organizations, invitations, API keys
	│   ├── services_rbac.go       # Roles, permissions
	│   ├── services_billing.go    # Subscriptions
	│   ├── services_crm.go        # CRM: leads, deals, quotes, AI analysis
	│   ├── services_accounting.go # Accounting: journal entries, bank reconciliation, multi-currency, invoices, expenses
	│   ├── services_supplychain.go # Supply chain: fleet, telematics, inventory, routes, warehouse stock
	│   ├── services_hr.go         # HR: attendance, shifts, payroll/tax, resume parsing, knowledge search
	│   ├── services_platform.go   # Platform: AI copilot, workflows, IoT batch, security anomalies
	│   └── services_mailer.go     # Transactional email (stub)
	├── handlers/
	│   ├── handlers_server.go             # App struct, DI wiring, Serve()
	│   ├── handlers_routes.go             # Route registration + middleware
	│   ├── handlers_health.go             # GET /health
	│   ├── handlers_auth.go               # POST auth/register, auth/login
	│   ├── handlers_org.go                # Org endpoints
	│   ├── handlers_billing.go            # Billing endpoints
	│   ├── handlers_crm.go                # CRM endpoints
	│   ├── handlers_accounting.go         # Accounting endpoints + bank statements, exchange rates
	│   ├── handlers_accounting_extra.go   # Bank reconciliation, exchange rates, currency conversion
	│   ├── handlers_supplychain.go        # Fleet & inventory endpoints + receive/issue/transfer/snapshot
	│   ├── handlers_supplychain_extra.go  # Inventory receive/issue/transfer/snapshot
	│   ├── handlers_hr.go                 # HR endpoints + clock-out, shifts, payroll detail/tax
	│   ├── handlers_hr_extra.go           # Clock-out, shift management, tax profiles, payroll detail
	│   ├── handlers_platform.go           # AI & Platform endpoints + batch IoT readings
	│   └── handlers_platform_extra.go     # Batch IoT reading ingestion
└── utils/
    ├── response.go          # WriteJSON, WriteErr, Envelope
    ├── middleware.go         # RequireAuth, RequirePermission, etc.
    └── validator.go          # Email, slug, string validators

docs/                        # Auto-generated Swagger (gitignored)
documentation/               # Human-readable documentation (this)
```

## Package dependency flow

```
cmd/api/main.go
    │
    ├── internal/config        (env → struct)
    ├── internal/database      (pgx pool + migrations)
    ├── internal/services      (business logic + SQL)
    │   └── depends on: database
    ├── internal/handlers      (HTTP handlers + App)
    │   └── depends on: services, utils, config, database
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

Configuration is loaded from environment variables (optionally via a `.env` file in the project root).

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP server port |
| `DATABASE_URL` | *(required)* | PostgreSQL connection string |
| `JWT_SECRET` | `dev-secret-change-in-production` | HMAC signing key for JWTs |
| `JWT_ISSUER` | `strata` | JWT issuer claim |
| `ENABLE_SWAGGER` | `true` | Enable Swagger UI at `/swagger/` |

## Running the project

```bash
# Start PostgreSQL
docker compose up -d

# Run the API server
go run ./cmd/api

# With hot-reload
air

# Smoke test
curl http://localhost:8080/health
```