# Super Admin API

The Super Admin subsystem provides centralized platform observability, real-time security monitoring (SOC), partitioned maintenance control, CI health ingestion, and platform-wide user/organization CRUD operations. All API endpoints require a valid JWT with the `super_admin.access` permission.

## Authentication

The super-admin UI uses a cookie-based session flow:

1. **Login** (`POST /api/v1/super-admin/login`) authenticates with email/password, sets an `strata_token` HttpOnly cookie with a signed JWT, and redirects to the dashboard.
2. **Dashboard** (`GET /api/v1/super-admin/dashboard`) validates the JWT from either the `Authorization` header or the `strata_token` cookie via `RequireAuthCookie` middleware.
3. **Logout** (`POST /api/v1/super-admin/logout`) clears the `strata_token` cookie (sets `MaxAge: -1`) and redirects to the login page.

API endpoints (metrics, health, users, organizations, etc.) use standard Bearer token authentication:

```http
Authorization: Bearer <super-admin-jwt>
```

The default super-admin user is seeded on first run from `SUPERADMIN_UNAME` / `SUPERADMIN_PWORD` environment variables (default: `admin@strata.local` / `SuperAdmin123!`).

## Dashboard (HTML)

### GET `/api/v1/super-admin/login`

Renders the super-admin login page as an HTML form. Accepts `GET` to display the form.

### POST `/api/v1/super-admin/login`

Authenticates with `email` and `password` form fields, checks for `super_admin.access` permission, and sets a `strata_token` HttpOnly cookie. On success redirects to `/api/v1/super-admin/dashboard`. On failure re-renders the login page with an error message.

### POST `/api/v1/super-admin/logout`

Clears the `strata_token` session cookie and redirects to the login page. This is a public endpoint (no authentication required — logout must work even if the token is expired).

### GET `/api/v1/super-admin/dashboard`

Renders the super-admin dashboard as an HTML page. Protected by `RequireAuthCookie` + `RequirePermission(services.PermSuperAdmin)`.

**Features:**
- Dark mode toggle (persisted in `localStorage`, applied via `html.dark` CSS class)
- Collapsible sidebar with hamburger button (works on all screen sizes)
- User menu pop-out card with sign-out option (Alpine.js)

## Metrics

### GET `/api/v1/super-admin/metrics`

Returns aggregated system telemetry in JSON format including runtime stats, database pool metrics, HTTP latency percentiles, and recent panics.

**Response (200):**
```json
{
  "metrics": {
    "timestamp": "2026-08-07T12:00:00Z",
    "runtime": {
      "allocated_mb": 32.5,
      "gc_runs": 142,
      "goroutines": 48,
      "heap_objects": 125000
    },
    "db": {
      "acquired_conns": 3,
      "idle_conns": 2,
      "total_conns": 5,
      "max_conns": 25
    },
    "http": {
      "total_requests": 1024,
      "status_2xx": 980,
      "status_4xx": 32,
      "status_5xx": 12,
      "latency_p50_ms": 2.3,
      "latency_p95_ms": 15.7,
      "latency_p99_ms": 45.2,
      "per_module": {}
    },
    "recent_panics": []
  }
}
```

### GET `/api/v1/super-admin/metrics/fragment`

Returns the MetricsGrid as an HTML fragment (no layout shell) for HTMX polling. The dashboard's metrics grid polls this endpoint every 5 seconds.

**Response (200):** `text/html`
```html
<div id="metrics-grid" class="metrics-grid">
  <div class="card">...</div>  <!-- Runtime Memory -->
  <div class="card">...</div>  <!-- DB Connections -->
  <div class="card">...</div>  <!-- Goroutines -->
  <div class="card">...</div>  <!-- System Status -->
</div>
```

**Client usage (HTMX):**
```html
<div hx-get="/api/v1/super-admin/metrics/fragment" hx-trigger="load, every 5s" hx-swap="outerHTML">
  <!-- MetricsGrid renders here -->
</div>
```

### GET `/api/v1/super-admin/metrics/prometheus`

Returns the same telemetry in [Prometheus exposition format](https://prometheus.io/docs/instrumenting/exposition_formats/) suitable for scraping by Prometheus or Grafana.

**Response (200):** `text/plain`
```
# HELP strata_runtime_allocated_mb Heap memory allocated in MB
# TYPE strata_runtime_allocated_mb gauge
strata_runtime_allocated_mb 32.50
# HELP strata_http_requests_total Total HTTP requests
# TYPE strata_http_requests_total counter
strata_http_requests_total 1024
...
```

## Health

### GET `/api/v1/super-admin/health`

Returns composite health scores (0–100%) for all modules, factoring in CI test coverage, linter issues, vulnerability counts, and 5xx error rates.

**Scoring formula:** 40% coverage score + 30% linter score + 20% vulnerability score + 10% error rate score.

**Response (200):**
```json
{
  "modules": [
    {
      "module": "accounting",
      "health_score": 87.5,
      "coverage_percent": 92.0,
      "linter_issues": 3,
      "vulnerabilities": 0,
      "error_rate_5xx_percent": 1.2
    }
  ]
}
```

## CI Health Ingestion

### POST `/api/v1/super-admin/telemetry/ci-health`

Stores a CI health report for a given module. Data is used to compute composite health scores.

**Request:**
```json
{
  "module": "accounting",
  "coverage_percent": 92.0,
  "linter_issues": 3,
  "vulnerabilities_count": 0,
  "commit_sha": "abc123def456"
}
```

**Response (201):**
```json
{
  "report": {
    "module": "accounting",
    "coverage_percent": 92.0,
    "linter_issues": 3,
    "vulnerabilities_count": 0,
    "commit_sha": "abc123def456",
    "created_at": "2026-08-07T12:00:00Z"
  }
}
```

## Partitioned Maintenance

Strata supports granular maintenance locks by `module` (e.g., `crm`, `accounting`), `tenant_id` (organization UUID), or `feature` (e.g., `ai_copilot`). When active, the `PartitionedMaintenanceMiddleware` returns HTTP 503 for all non-admin requests targeting the locked scope. Rules are cached in-memory per node and synchronized across instances via Redis Pub/Sub.

### GET `/api/v1/super-admin/maintenance`

Lists all currently active maintenance rules.

**Response (200):**
```json
{
  "rules": [
    {
      "id": 1,
      "scope": "module",
      "target_id": "crm",
      "is_active": true,
      "reason": "Database migration in progress",
      "allowed_roles": [],
      "created_at": "2026-08-07T12:00:00Z",
      "updated_at": "2026-08-07T12:00:00Z"
    }
  ]
}
```

### POST `/api/v1/super-admin/maintenance/toggle`

Activates or deactivates a partitioned maintenance lock. Publishes a cache-invalidation event to all connected nodes via Redis Pub/Sub channel `strata:events:maintenance-sync`.

**Request:**
```json
{
  "scope": "module",
  "target_id": "crm",
  "is_active": true,
  "reason": "Database migration in progress"
}
```

**Response (200):**
```json
{
  "rule": {
    "scope": "module",
    "target_id": "crm",
    "is_active": true,
    "reason": "Database migration in progress"
  }
}
```

| Parameter | Type | Required | Description |
|:---|:---|:---|:---|
| `scope` | string | yes | Lock scope: `module`, `tenant_id`, or `feature` |
| `target_id` | string | yes | Target identifier (module name, org UUID, feature name) |
| `is_active` | bool | yes | `true` to activate, `false` to deactivate |
| `reason` | string | yes | Human-readable reason for the maintenance lock |
| `allowed_roles` | []string | no | Role IDs allowed through the lock (future use) |

## Security Event Stream (SSE)

### GET `/api/v1/super-admin/security/stream`

Opens a Server-Sent Events (SSE) connection that streams real-time SOC security events as HTML fragments. Events are fanned out across all server nodes via Redis Pub/Sub channel `strata:events:security-soc`.

**Authentication:** Requires `strata_token` cookie (browser) or `Authorization: Bearer <token>` header. Uses `RequireAuthCookie` middleware so HTMX SSE extension connections work with session cookies.

**Event types:**
- `user.banned` — User banned platform-wide
- `user.unbanned` — User unbanned
- `org.suspended` — Organization suspended
- `org.activated` — Organization reactivated

**Example SSE stream:**
```
event: connected
data: {"status":"connected"}

event: security-event
data: <div class="sse-entry sse-severity-high"><div class="sse-entry-header"><span class="sse-entry-type">user.banned</span><span class="badge badge-high">high</span><span class="sse-entry-ip">192.168.1.100</span><span class="sse-entry-time">12:34:56</span></div><p class="sse-entry-message">User user@example.com banned: Policy violation</p></div>
```

**Client usage (HTMX SSE extension):**
```html
<div hx-ext="sse" sse-connect="/api/v1/super-admin/security/stream" sse-swap="security-event" hx-swap="afterbegin">
  <!-- New log entries appear at the top -->
</div>
```

**Client usage (curl):**
```bash
curl -N -H "Authorization: Bearer <token>" http://localhost:8080/api/v1/super-admin/security/stream
```

**Testing with mock events:**
```sh
# Publish 5 mock SOC events to Redis
go run ./cmd/cli/publish-soc-events -count 5 -redis localhost:6379

# Or using make:
make testsoc
```

The connection is automatically cleaned up when the HTTP request context is cancelled (client disconnects).

## User & Organization Management

Super admins can perform platform-wide CRUD operations on users and organizations, including ban/suspend actions that generate SOC events.

### List All Users

```http
GET /api/v1/super-admin/users?offset=0&limit=50
```

**Query params:** `offset` (default 0), `limit` (default 50, max 100)

**Response (200):**
```json
{
  "users": [
    {
      "id": "uuid",
      "email": "user@example.com",
      "full_name": "John Doe",
      "is_banned": false,
      "ban_reason": "",
      "created_at": "2026-08-07T12:00:00Z"
    }
  ],
  "total": 42
}
```

### Get User Details

```http
GET /api/v1/super-admin/users/{user_id}
```

Returns full user profile including organization memberships.

### Ban a User

```http
POST /api/v1/super-admin/users/{user_id}/ban
```

**Request:**
```json
{
  "reason": "Violation of terms of service"
}
```

Generates a `user.banned` SOC event.

### Unban a User

```http
POST /api/v1/super-admin/users/{user_id}/unban
```

No request body required. Generates a `user.unbanned` SOC event.

### List All Organizations

```http
GET /api/v1/super-admin/organizations?offset=0&limit=50
```

**Query params:** `offset` (default 0), `limit` (default 50, max 100)

**Response (200):**
```json
{
  "organizations": [
    {
      "id": "uuid",
      "domain_slug": "acme-corp",
      "company_name": "Acme Corporation",
      "status": "active",
      "created_at": "2026-08-07T12:00:00Z"
    }
  ],
  "total": 15
}
```

### Get Organization Details

```http
GET /api/v1/super-admin/organizations/{org_id}
```

### Suspend an Organization

```http
POST /api/v1/super-admin/organizations/{org_id}/suspend
```

Sets organization status to `suspended`. Generates an `org.suspended` SOC event.

### Activate an Organization

```http
POST /api/v1/super-admin/organizations/{org_id}/activate
```

Sets organization status back to `active`. Generates an `org.activated` SOC event.

## Database Tables

The super-admin subsystem persists data in three tables:

| Table | Purpose |
|:---|:---|
| `super_admin_maintenance_rules` | Persisted maintenance lock rules |
| `super_admin_system_errors` | Captured panic traces and system errors |
| `super_admin_ci_health_reports` | CI test coverage, linter, and vulnerability reports |

Additionally, two columns were added to the `users` table:
- `is_banned` (BOOLEAN, default FALSE)
- `ban_reason` (TEXT, default '')

## Architecture Notes

- **Maintenance cache:** Active rules are held in a `sync.RWMutex`-protected map for sub-microsecond read overhead on every HTTP request
- **Multi-node sync:** Cache invalidation and SSE fan-out use Redis Pub/Sub (`strata:events:maintenance-sync` and `strata:events:security-soc`)
- **Bounded memory:** Telemetry, panic traces, and HTTP latencies use fixed-size ring buffers (max 100 items)
- **Redis optional:** The subsystem gracefully degrades if Redis is unavailable — cache sync and SSE fan-out become single-node only
