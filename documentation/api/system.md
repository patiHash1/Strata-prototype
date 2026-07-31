# System

## Overview

The System module provides infrastructure-level endpoints for health checks and monitoring.

---

## Endpoints

### Health check

Returns the current health status of the API and its dependencies.

```http
GET /health
```

**Response** `200 OK`:
```json
{
    "status": "ok",
    "database": "connected"
}
```

**Response** `503 Service Unavailable` (when database is down):
```json
{
    "status": "ok",
    "database": "unavailable"
}
```

**Behavior:**
- Always returns HTTP 200 unless the database check fails
- If a database connection is configured, it performs a `Ping` to verify connectivity
- If the database ping fails, HTTP 503 is returned with `"database": "unavailable"`
- The `"status"` field always reads `"ok"` — this is the service's own status, not the DB's

### Swagger UI

When `ENABLE_SWAGGER=true` (default), the Swagger UI is served at:

```
GET /swagger/
```

This provides an interactive API documentation explorer. The OpenAPI spec is auto-generated from Go annotations using `swaggo/swag`.

**Regenerating the spec:**
```bash
swag init --dir ./cmd/api,./internal/handlers --output ./docs --parseDependency --parseInternal
```