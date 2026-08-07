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
handler = utils.CORSMiddleware(handler)                           // 1. CORS headers
handler = utils.LoggingMiddleware(adminSvc)(handler)              // 2. Request logging + latency tracking
handler = utils.RecoveryMiddleware(adminSvc)(handler)             // 3. Panic recovery + stack capture
handler = utils.PartitionedMaintenanceMiddleware(adminSvc)(handler) // 4. Maintenance mode enforcement
```

### CORSMiddleware

```go
func CORSMiddleware(next http.Handler) http.Handler
```

Sets permissive CORS headers for development:

- `Access-Control-Allow-Origin: *`
- `Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS`
- `Access-Control-Allow-Headers: Content-Type, Authorization, X-Tenant-Domain, X-API-Key`

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
- Users with the `super_admin.access` permission bypass all maintenance checks

**Performance:** The in-memory cache read is O(1) with sub-microsecond overhead — no database queries or allocations on the hot path.

**Multi-node sync:** When a maintenance rule is toggled via the API, a cache-invalidation message is published to Redis channel `strata:events:maintenance-sync`. All connected nodes reload their cache from PostgreSQL within milliseconds.

## Route-level middleware

Route-level middleware is composed at registration time:

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
1. `PartitionedMaintenanceMiddleware` — blocks maintenance-locked requests first (outermost)
2. `RecoveryMiddleware` — catches panics from all inner layers
3. `LoggingMiddleware` — logs request + records latency metrics
4. `CORSMiddleware` — sets CORS headers (innermost global)

For route-level middleware, `RequireAuth` must always come before `RequirePermission` because the permission check depends on claims being present in the context.
