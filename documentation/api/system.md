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

### Module Health Endpoints

Each module exposes a public health endpoint at `GET /api/v1/{module}/health`. These endpoints do not require authentication and are designed for load balancers, uptime monitors, and client-side health checks.

The response includes the module's operational status, active maintenance rules, CI health data, and HTTP metrics. See [Super Admin — Module Health](super-admin.md#get-apiv1modulehealth) for full documentation.

**Example:**
```sh
curl http://localhost:8080/api/v1/crm/health
# {"module":"crm","status":"operational","healthy":true}
```