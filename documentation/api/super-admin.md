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

### GET `/api/v1/{module}/health`

Returns the current operational state of a specific module. This is a **public endpoint** (no authentication required) designed for load balancers, uptime monitors, and client-side health checks.

The response includes:
- **Maintenance status** — whether the module or any of its features are under maintenance
- **CI health** — test coverage, linter issues, and vulnerability counts from the latest CI report (if available)
- **HTTP metrics** — total requests, 5xx error count, and error rate for the module

**Status values:**

| Status | HTTP Code | Meaning |
|:---|:---|:---|
| `operational` | `200` | Module is healthy and serving requests normally |
| `degraded` | `200` | Module is operational but one or more features are under maintenance |
| `maintenance` | `503` | The entire module is under maintenance |

**Response — operational (200):**
```json
{
  "module": "crm",
  "status": "operational",
  "healthy": true
}
```

**Response — degraded (200):**
```json
{
  "module": "crm",
  "status": "degraded",
  "healthy": true,
  "features_under_maintenance": [
    {
      "feature": "ai-copilot",
      "target": "crm:ai-copilot",
      "reason": "Upgrading AI model",
      "since": "2026-08-12T09:00:00Z"
    }
  ]
}
```

**Response — maintenance (503):**
```json
{
  "module": "crm",
  "status": "maintenance",
  "healthy": false,
  "maintenance": {
    "scope": "module",
    "target": "crm",
    "reason": "Database migration in progress",
    "since": "2026-08-12T09:20:00Z"
  }
}
```

**Available modules:** `crm`, `accounting`, `hr`, `billing`, `account`, `org`, `ai`, `bi`, `workflows`, `fleet`, `inventory`, `manufacturing`, `procurement`, `iot`, `security`

**Example:**
```sh
curl http://localhost:8080/api/v1/crm/health
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

### Maintenance Control Panel (UI)

The maintenance management page is accessible at `/api/v1/super-admin/maintenance/rules` from the dashboard sidebar. It provides:

- **Rule list table** — shows all maintenance rules (active and inactive) with scope, target, reason, status, and creation date
- **Add Rule modal** — Alpine.js modal with a form to create new rules (scope selector, target ID, reason)
- **Revoke button** — per-row button that soft-deactivates the rule with a 1-second fade-out CSS transition
- **HTMX OOB swaps** — new rules are appended to the table via out-of-band swaps without a full page reload

### GET `/api/v1/super-admin/maintenance/rules`

Renders the full maintenance rules management page inside the dashboard layout (HTML). Protected by `RequireAuthCookie` + `RequirePermission`.

### GET `/api/v1/super-admin/maintenance/fragment`

Returns the `<tbody>` of the rules table as an HTML fragment for HTMX partial swaps.

### POST `/api/v1/super-admin/maintenance`

Creates a new maintenance rule. The form submits as `application/x-www-form-urlencoded` with fields: `scope`, `target_id`, `reason`.

**On success (200):** Returns an HTML table row with `hx-swap-oob="beforeend:#rules-table"` so HTMX appends it to the table.

**On validation error (400):** Returns an HTML error banner with `hx-swap-oob="innerHTML:#modal-validation"`.

### DELETE `/api/v1/super-admin/maintenance/{id}`

Soft-deactivates a maintenance rule (sets `is_active = FALSE`). Returns an empty `<tr>` to replace the row, allowing the CSS fade-out transition (`hx-swap="outerHTML swap:1s"`) to play before removal.

Publishes a cache-invalidation event to all connected nodes via Redis Pub/Sub channel `strata:events:maintenance-sync`.

### Legacy JSON API

The `POST /api/v1/super-admin/maintenance/toggle` endpoint remains available for programmatic access (JSON request/response).

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

On connect, the handler replays the most recent events from a 50-slot in-memory ring buffer so the client sees events that occurred between page load and SSE connection open (e.g., login events).

### SOC Event Types

The following event types are published to the live security feed:

| Event Type | Severity | Trigger |
|:---|:---|:---|
| `super_admin.login` | info | Super admin successfully authenticates |
| `super_admin.logout` | info | Super admin clears session |
| `maintenance.created` | warning | New maintenance rule created via the control panel |
| `maintenance.revoked` | warning | Maintenance rule revoked/deactivated |
| `user.banned` | high | User banned platform-wide |
| `user.unbanned` | info | User unbanned |
| `org.suspended` | high | Organization suspended |
| `org.activated` | info | Organization reactivated |

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

The super-admin subsystem persists data in four tables:

| Table | Purpose |
|:---|:---|
| `super_admin_maintenance_rules` | Persisted maintenance lock rules |
| `super_admin_system_errors` | Captured panic traces and system errors |
| `super_admin_ci_health_reports` | CI test coverage, linter, and vulnerability reports |
| `super_admin_soc_events` | Persisted security events (login, logout, maintenance, ban/suspend) with JSONB metadata |

Additionally, two columns were added to the `users` table:
- `is_banned` (BOOLEAN, default FALSE)
- `ban_reason` (TEXT, default '')

### SOC Event Retention

Security events are persisted to `super_admin_soc_events` asynchronously (non-blocking). A background goroutine runs a sliding-window delete every 6 hours, pruning events older than 25 days. The retention window is configured via the `socEventRetentionDays` constant in `services_super_admin.go`.

The `super_admin_soc_events` table stores:
- `id` — UUID v4
- `event_type` — e.g., `super_admin.login`, `maintenance.created`
- `severity` — `info`, `warning`, or `high`
- `message` — human-readable description
- `ip_address`, `user_id`, `org_id` — context fields
- `metadata` — JSONB blob for arbitrary event data (e.g., scope, target_id, reason)
- `created_at` — event timestamp

## Architecture Notes

- **Maintenance cache:** Active rules are held in a `sync.RWMutex`-protected map for sub-microsecond read overhead on every HTTP request
- **SOC event buffer:** A 50-slot in-memory ring buffer holds recent SOC events. SSE clients replay the buffer on connect so events published between page load and SSE open are not lost
- **SOC event persistence:** Events are async-persisted to `super_admin_soc_events` (non-blocking) and pruned after 25 days by a background goroutine
- **Multi-node sync:** Cache invalidation and SSE fan-out use Redis Pub/Sub (`strata:events:maintenance-sync` and `strata:events:security-soc`)
- **Bounded memory:** Telemetry, panic traces, HTTP latencies, and SOC events use fixed-size ring buffers (100, 100, 100, and 50 items respectively)
- **Redis optional:** The subsystem gracefully degrades if Redis is unavailable — cache sync and SSE fan-out become single-node only
