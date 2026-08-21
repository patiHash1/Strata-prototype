# Deployment & Operations

This guide covers building, containerizing, and operating the Strata API in production-like environments.

## Prerequisites

- **Go 1.26+** (see `go.mod`)
- **Docker** + **Docker Compose** (for local PostgreSQL/Redis and containerized deployments)
- **Make** (optional — the `Makefile` wraps common tasks)

## Building the binary

```bash
# Build the production binary (stripped, static)
go build -ldflags="-s -w" -o ./tmp/main ./cmd/api

# Or via make
make build
```

The binary is self-contained (static, `CGO_ENABLED=0`) and embeds migrations, templates, and static assets, so it has no runtime file dependencies beyond the environment variables.

## Running locally

```bash
# 1. Start PostgreSQL and Redis
docker compose up -d

# 2. Configure environment (see environment-variables.md)
cp .env.example .env   # if you have one; otherwise export vars manually

# 3. Run the server (migrations run automatically on startup)
go run ./cmd/api
```

Migrations are **applied automatically at startup** — there is no separate migration command. The server tracks applied migrations in the `schema_migrations` table and only applies new ones.

## Docker deployment

The included `Dockerfile` is a two-stage build:

1. **Builder** (`golang:1.26.5-alpine`) — downloads modules, compiles `/app/server`.
2. **Runtime** (`alpine:latest`) — runs as a non-root `appuser`, exposes port `8080`.

```bash
# Build the image
docker build -t strata-api .

# Run it (supply required env vars)
docker run --rm -p 8080:8080 \
  -e DATABASE_URL="postgres://strata-user:strata-pass@host.docker.internal:5432/strata-db-beta?sslmode=disable" \
  -e JWT_SECRET="a-long-random-secret" \
  -e ALLOWED_ORIGINS="https://app.example.com" \
  strata-api
```

> `compose.yaml` is intended for **local development** (PostgreSQL + Redis only). It does not run the API itself.

## Production requirements

| Requirement | Notes |
|---|---|
| PostgreSQL | Version 16 recommended. The pool is configured with `MaxConns=25`, `MinConns=5`. |
| Redis (recommended) | Required for multi-node maintenance-cache sync and SSE fan-out. Optional for single-node. |
| `JWT_SECRET` | **Required.** Use a long random value; rotate carefully (rotation invalidates existing tokens). |
| `DATABASE_URL` | **Required.** |
| `ENABLE_SWAGGER` | Set to `false` in production to disable the Swagger UI. |
| `ALLOWED_ORIGINS` | Restrict to your real frontend origins. Empty blocks cross-origin browser access. |
| `SUPERADMIN_UNAME` / `SUPERADMIN_PWORD` | Set to provision the platform super-admin on first boot. |

## Health checks & observability

- **Liveness:** `GET /health` returns `200` with `{"status":"ok","database":"connected"}` (plus `redis` when configured). Returns `503` if the database ping fails.
- **Per-module health:** `GET /api/v1/{module}/health` — public, returns module operational state and maintenance status.
- **Prometheus metrics:** `GET /api/v1/super-admin/metrics/prometheus` — Prometheus exposition format (requires `super_admin.access`).
- **Structured logs:** The app logs JSON to stderr via `log/slog` (see `internal/logger`).

## Data seeding / wiping

A standalone seed script inserts 20 dummy records per table for development:

```bash
make data-seed   # go run ./scripts/seed.go seed
make data-wipe   # go run ./scripts/seed.go wipe
```

> The seed script is separate from the startup migration seeding (super-admin, default currencies, plans, and permissions are seeded by migrations).

## Testing SOC events

To publish mock security events to Redis for testing the super-admin live security feed:

```bash
make testsoc    # publishes 5 mock events
# or
go run ./cmd/cli/publish-soc-events -count 10 -redis localhost:6379 -redis-pass strata-redis-pass
```

## SOC event retention

Security events are persisted to `super_admin_soc_events` and pruned by a background goroutine every 6 hours, deleting events older than **25 days** (constant `socEventRetentionDays` in `services_super_admin.go`).

## Graceful shutdown

The server handles `SIGINT`/`SIGTERM`, stops accepting new connections, and drains in-flight requests with a 10-second timeout before exiting (see `App.Serve` in `handlers_server.go`).
