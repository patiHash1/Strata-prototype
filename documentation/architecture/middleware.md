# Middleware

## Overview

Middleware in Strata is implemented as higher-order functions that wrap `http.Handler`. There are two tiers:

1. **Global middleware** — wraps the entire mux (logging, CORS, panic recovery)
2. **Route-level middleware** — wraps individual handlers (auth, permission checks)

## Global middleware

Global middleware is applied in `routes()` in `handlers_routes.go`. The order determines the execution flow (outermost first):

```go
var handler http.Handler = mux
handler = utils.CORSMiddleware(handler)     // 1. CORS headers
handler = utils.LoggingMiddleware(handler)  // 2. Request logging
handler = utils.RecoveryMiddleware(handler) // 3. Panic recovery
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
func LoggingMiddleware(next http.Handler) http.Handler
```

Logs every request with method, path, status code, and duration:

```
2026/08/01 12:00:00 POST /api/v1/auth/login 200 12.345µs
```

Uses a custom `loggingResponseWriter` to capture the status code.

### RecoveryMiddleware

```go
func RecoveryMiddleware(next http.Handler) http.Handler
```

Catches panics in handler code and returns `500 Internal Server Error` with a JSON error body, preventing the server from crashing.

## Route-level middleware

Route-level middleware is composed at registration time:

```go
mux.Handle("POST /api/v1/org/invitations",
    utils.RequireAuth(a.Auth)(
        utils.RequirePermission(services.PermUsersInvite)(
            http.HandlerFunc(a.inviteHandler),
        ),
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

## Context key

Claims are stored in the request context using a typed context key:

```go
type contextKey string
const claimsKey contextKey = "auth.claims"
```

This prevents key collisions with other context values.

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

For route-level middleware, `RequireAuth` must always come before `RequirePermission` because the permission check depends on claims being present in the context.