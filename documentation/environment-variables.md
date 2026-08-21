# Environment Variables

All application configuration is read from environment variables at startup. A `.env` file in the project root is loaded automatically (if present) via `godotenv` — see `internal/env/env.go`. In production, set these directly in your environment or container.

The typed `Config` struct that consumes these is defined in `internal/config/config.go`.

## Application variables

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `PORT` | int | No | `8080` | HTTP port the API server listens on. |
| `ENABLE_SWAGGER` | bool | No | `true` | When `true`, serves the Swagger UI at `/swagger/`. Set to `false` in production to disable it. |
| `DATABASE_URL` | string | **Yes** | — | PostgreSQL connection string (e.g. `postgres://user:pass@localhost:5432/strata-db-beta?sslmode=disable`). The server **panics at startup** if this is empty. |
| `JWT_SECRET` | string | **Yes** | — | HMAC-SHA256 signing key for JWTs. The server **panics at startup** if this is empty. Use a long, random, high-entropy value in production. |
| `JWT_ISSUER` | string | No | `strata` | Value of the `iss` claim embedded in issued JWTs. |
| `ALLOWED_ORIGINS` | string | No | *(empty)* | Comma-separated list of allowed CORS origins (e.g. `https://app.example.com,https://admin.example.com`). When empty, no `Access-Control-Allow-Origin` header is set, which blocks cross-origin browser requests. |
| `SUPERADMIN_UNAME` | string | No | *(empty)* | Email of the platform super-admin. If both this and `SUPERADMIN_PWORD` are set, the super-admin account is seeded (idempotently) on startup. |
| `SUPERADMIN_PWORD` | string | No | *(empty)* | Password for the super-admin account. Seeded on startup only when paired with `SUPERADMIN_UNAME`. |

## Redis variables (optional)

Redis powers multi-node maintenance-cache sync and SSE fan-out for the super-admin SOC feed. If `REDIS_ADDR` is empty, the application runs without Redis and degrades to single-node operation.

| Variable | Type | Required | Default | Description |
|---|---|---|---|---|
| `REDIS_ADDR` | string | No | *(empty)* | Redis address, e.g. `localhost:6379`. Empty disables Redis. |
| `REDIS_PASSWORD` | string | No | *(empty)* | Redis password (required if the server uses `--requirepass`). |
| `REDIS_DB` | int | No | `0` | Redis logical database number. |

## Docker Compose variables

`compose.yaml` (PostgreSQL 16 + Redis 7) reads a few additional variables with defaults:

| Variable | Default | Description |
|---|---|---|
| `POSTGRES_USER` | `strata-user` | PostgreSQL superuser for the container. |
| `POSTGRES_PASSWORD` | *(required by compose)* | PostgreSQL password. |
| `POSTGRES_DB` | `strata-db-beta` | Name of the initial database created by the container. |
| `REDIS_PASSWORD` | *(required by compose)* | Redis password; also passed to the API via `REDIS_PASSWORD`. |

## Minimal `.env` example

```bash
# Server
PORT=8080
ENABLE_SWAGGER=true

# Database (required)
DATABASE_URL=postgres://strata-user:strata-pass@localhost:5432/strata-db-beta?sslmode=disable

# Auth (required)
JWT_SECRET=change-me-to-a-long-random-string
JWT_ISSUER=strata

# Redis (optional)
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=strata-redis-pass
REDIS_DB=0

# CORS (optional)
ALLOWED_ORIGINS=http://localhost:8080

# Super admin seeding (optional)
SUPERADMIN_UNAME=admin@strata.local
SUPERADMIN_PWORD=SuperAdmin123!
```

> **Note:** `JWT_SECRET` and `DATABASE_URL` are the only two variables that cause a hard startup failure when missing. All others fall back to safe defaults.
