# Middleware

## Overview

Middleware in Strata is implemented as higher-order functions that wrap `http.Handler`. There are two tiers:

1. **Global middleware** — wraps the entire mux (logging, CORS, panic recovery)
2. **Route-level middleware** — wraps individual handlers (auth, permission checks)

There are two authentication modes supported at the route level:
- **Bearer token (JWT)** — `RequireAuth` + `RequirePermission`
- **API key** — `RequireAPIKey` (for machine-to-machine endpoints like telemetry ingestion)

## Global middleware

Global middleware is applied in `routes()` in `handlers_routes.go`. The order determines the execution flow (outermost first):

```go
var handler http.Handler = mux
handler = utils.MaxBodySizeMiddleware(1 << 20)(handler)                          // 1. 1 MiB body limit
handler = utils.CORSMiddleware(a.Config.AllowedOrigins)(handler)                 // 2. CORS headers
handler = utils.LoggingMiddleware(a.SuperAdmin)(handler)                         // 3. Request logging + latency tracking
handler = utils.RecoveryMiddleware(a.SuperAdmin)(handler)                        // 4. Panic recovery + stack capture
handler = utils.PartitionedMaintenanceMiddleware(a.SuperAdmin)(handler)          // 5. Maintenance mode enforcement
```

### MaxBodySizeMiddleware

```go
func MaxBodySizeMiddleware(maxBytes int64) func(http.Handler) http.Handler
```

Limits the maximum request body size to prevent memory exhaustion from oversized payloads. Applied globally with a `1 << 20` (1 MiB) limit. If `maxBytes <= 0`, it defaults to 1 MiB.

### CORSMiddleware

```go
func CORSMiddleware(allowedOrigins string) func(http.Handler) http.Handler
```

Sets CORS headers based on a **configurable comma-separated list of allowed origins** (from the `ALLOWED_ORIGINS` env var):

- For a matching `Origin`, sets `Access-Control-Allow-Origin` to that origin plus `Vary: Origin` and `Access-Control-Allow-Credentials: true`
- `Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-Tenant-Domain, X-API-Key`
- If `allowedOrigins` is empty, no `Access-Control-Allow-Origin` header is set, which blocks cross-origin browser requests

Preflight `OPTIONS` requests are handled immediately with `204 No Content`.

### LoggingMiddleware

```go
func LoggingMiddleware(adminSvc *services.SuperAdminService) func(http.Handler) http.Handler
```

Logs every request with method, path, status code, and duration. Also pushes latency records into the `SuperAdminService` for percentile computation and per-module HTTP metrics aggregation:

```
2026/08/01 12:00:00 POST /api/v1/auth/login 200 12.345µs
```

Uses a custom `loggingResponseWriter` to capture the status code.

### RecoveryMiddleware

```go
func RecoveryMiddleware(adminSvc *services.SuperAdminService) func(http.Handler) http.Handler
```

Catches panics in handler code and returns `500 Internal Server Error` with a JSON error body. Captures the full stack trace into the `SuperAdminService` ring buffer and persists it asynchronously to `super_admin_system_errors`.

### PartitionedMaintenanceMiddleware

```go
func PartitionedMaintenanceMiddleware(adminSvc *services.SuperAdminService) func(http.Handler) http.Handler
```

Runs on **every incoming HTTP request** and checks the local in-memory maintenance cache (`sync.RWMutex` map) for active maintenance locks. Returns HTTP 503 with a JSON error body if the request's module or tenant is under maintenance.

**Bypass rules:**
- All routes matching `/api/v1/super-admin/*` are always accessible
- Module health endpoints (`GET /api/v1/{module}/health`) are registered before the maintenance middleware and return the current maintenance state rather than being blocked by it
- Users with the `super_admin.access` permission bypass all maintenance checks

**Performance:** The in-memory cache read is O(1) with sub-microsecond overhead — no database queries or allocations on the hot path.

**Multi-node sync:** When a maintenance rule is toggled via the API, a cache-invalidation message is published to Redis channel `strata:events:maintenance-sync`. All connected nodes reload their cache from PostgreSQL within milliseconds.

**SOC events:** Maintenance create and revoke actions automatically publish `maintenance.created` and `maintenance.revoked` SOC events. These are persisted to `super_admin_soc_events`, buffered in a 50-slot in-memory ring buffer for SSE replay, and fanned out via Redis Pub/Sub.

## Route-level middleware

Route-level middleware is composed at registration time. Strata supports three authentication modes:

**Bearer token (JWT):**

```go
mux.Handle("POST /api/v1/org/invitations",
    utils.RequireAuth(a.Auth)(
        utils.RequirePermission(services.PermUsersInvite)(
            http.HandlerFunc(a.inviteHandler),
        ),
    ),
)
```

**Cookie + header (for HTML dashboards):**

```go
mux.Handle("GET /api/v1/super-admin/dashboard",
    utils.RequireAuthCookie(a.Auth)(
        utils.RequirePermission(services.PermSuperAdmin)(
            http.HandlerFunc(a.superAdminDashboardHandler),
        ),
    ),
)
```

**API key:**

```go
mux.Handle("POST /api/v1/fleet/telematics/ingest",
    utils.RequireAPIKey(a.SupplyChain, services.PermFleetTelematicsIngest)(
        http.HandlerFunc(a.ingestTelemetryHandler),
    ),
)
```

### RequireAuth

```go
func RequireAuth(authSvc *services.AuthService) func(http.Handler) http.Handler
```

Validates the `Authorization: Bearer <token>` header:

1. Extracts the token from the `Authorization` header
2. Validates the JWT using `AuthService.ValidateToken()`
3. Injects the parsed `Claims` into the request context under the `claimsKey` context key

Claims are retrievable by handlers via `utils.GetClaims(r)`:

```go
claims := utils.GetClaims(r)
if claims == nil {
    // authentication required
}
```

### RequireAuthCookie

```go
func RequireAuthCookie(authSvc *services.AuthService) func(http.Handler) http.Handler
```

Validates a JWT from **either** the `Authorization` header **or** the `strata_token` cookie. Token extraction order:

1. Check `Authorization: Bearer <token>` header (same as `RequireAuth`)
2. Fall back to `strata_token` cookie
3. If neither is present or valid:
   - **Browser requests** (Accept header contains `text/html`): redirects to `/api/v1/super-admin/login` (302)
   - **API requests**: returns `401 Unauthorized` (JSON)

Claims are injected into the request context identically to `RequireAuth`, so handlers can use `utils.GetClaims(r)` regardless of which middleware was used.

**Usage:** Use `RequireAuthCookie` for HTML pages served via the super-admin dashboard. Use `RequireAuth` for JSON API endpoints.

### RequirePermission

```go
func RequirePermission(perms ...string) func(http.Handler) http.Handler
```

Checks that the authenticated user has at least one of the specified permissions:

1. Retrieves claims from the request context
2. Builds a set from `claims.Permissions`
3. If any of the required permissions are present, the handler proceeds
4. Otherwise, returns `403 Forbidden`

The permission check logic is **OR** — the user needs only one of the listed permissions. For AND logic, chain multiple `RequirePermission` calls.

### RequireAPIKey

```go
func RequireAPIKey(svc interface{ ValidateAPIKey(ctx context.Context, rawKey string) (uuid.UUID, []string, error) }, requiredScopes ...string) func(http.Handler) http.Handler
```

Validates the `X-API-Key` header against stored API keys:

1. Reads the raw key from the `X-API-Key` header
2. Calls the service's `ValidateAPIKey()` method which bcrypt-verifies the key against all active keys
3. Checks that the key has at least one of the required scopes
4. Injects `APIKeyClaims` (containing `OrgID` and `Scopes`) into the request context

API key claims are retrievable by handlers via `utils.GetAPIKeyClaims(r)`:

```go
claims := utils.GetAPIKeyClaims(r)
if claims == nil {
    // API key authentication required
}
orgID, err := uuid.Parse(claims.OrgID)
```

**Scope checking:** Like `RequirePermission`, the scope check uses **OR** logic — the key needs only one of the required scopes.

## Context keys

Two context keys are used for storing authentication state:

```go
type contextKey string
const claimsKey contextKey = "auth.claims"          // JWT (Bearer) claims
const apiKeyClaimsKey contextKey = "apikey.claims"  // API key claims
```

This prevents key collisions with other context values.

## Helpers

### GetClaims

```go
func GetClaims(r *http.Request) *services.Claims
```

Extracts JWT auth claims from the request context. Returns `nil` if absent.

### GetAPIKeyClaims

```go
func GetAPIKeyClaims(r *http.Request) *APIKeyClaims
```

Extracts API key claims from the request context. Returns `nil` if absent.

```go
type APIKeyClaims struct {
    OrgID  string
    Scopes []string
}
```

## Adding new middleware

### 1. Write the middleware function

```go
// internal/utils/middleware.go
func RateLimitMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Rate-limit logic…
        next.ServeHTTP(w, r)
    })
}
```

### 2. Apply globally or per-route

**Global** (in `handlers_routes.go`):
```go
handler = utils.RateLimitMiddleware(handler)
```

**Per-route** (in `handlers_routes.go`):
```go
mux.Handle("POST /api/v1/org/invitations",
    utils.RequireAuth(a.Auth)(
        utils.RateLimitMiddleware(
            http.HandlerFunc(a.inviteHandler),
        ),
    ),
)
```

## Middleware order

The order of middleware composition matters:

- **Global middleware** is applied outermost-first, meaning the first middleware wrapped runs last (it's the outermost wrapper)
- **Route-level middleware** is applied in the order it's composed: `RequireAuth` runs before `RequirePermission`, which runs before the handler

**Current global stack (execution order):**
1. `MaxBodySizeMiddleware` — enforces the 1 MiB body limit (outermost)
2. `CORSMiddleware` — sets CORS headers for allowed origins
3. `LoggingMiddleware` — logs request + records latency metrics
4. `RecoveryMiddleware` — catches panics from all inner layers
5. `PartitionedMaintenanceMiddleware` — blocks maintenance-locked requests (innermost global)

For route-level middleware, `RequireAuth` (or `RequireAuthCookie`) must always come before `RequirePermission` because the permission check depends on claims being present in the context.
