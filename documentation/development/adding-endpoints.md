# Adding Endpoints

This guide walks through adding a new HTTP endpoint to the Strata API.

## Step 1: Add the handler

Create a handler function in the appropriate file under `internal/handlers/`. If your endpoint belongs to a new domain, create a new file following the `handlers_<category>.go` naming convention.

**File naming:** `handlers_<category>.go` — one file per endpoint category.

```go
// internal/handlers/handlers_users.go
package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/patiHash1/Strata-prototype/internal/utils"
)

type listUsersRequest struct {
    Page  int `json:"page"`
    Limit int `json:"limit"`
}

type ListUsersResponse struct {
    Users []services.User `json:"users"`
    Total int             `json:"total" example:"42"`
}

// listUsersHandler returns a paginated list of users.
//
//  @Summary      List users
//  @Description  Returns all users with pagination.
//  @Tags         Users
//  @Produce      json
//  @Param        page  query  int  false  "Page number"  default(1)
//  @Param        limit query  int  false  "Items per page"  default(20)
//  @Success      200  {object}  ListUsersResponse
//  @Failure      500  {object}  utils.Envelope
//  @Router       /users [get]
func (a *App) listUsersHandler(w http.ResponseWriter, r *http.Request) {
    var req listUsersRequest
    // Parse request...

    // Call service...
    users, err := a.Users.List(r.Context(), req.Page, req.Limit)
    if err != nil {
        utils.WriteErr(w, http.StatusInternalServerError, "could not list users")
        return
    }

    utils.WriteJSON(w, http.StatusOK, utils.Envelope{
        "users": users,
        "total": len(users),
    })
}
```

### Handler conventions

- **Signature:** `func (a *App) actionHandler(w http.ResponseWriter, r *http.Request)`
- **Parse input:** Use `json.NewDecoder(r.Body).Decode()` for JSON bodies
- **Validate:** Use `utils` validators (`NotBlank`, `IsEmail`, `MinLen`, etc.)
- **Call service:** Delegate to the appropriate service via `a.ServiceName.Method()`
- **Write response:** Use `utils.WriteJSON()` or `utils.WriteErr()`
- **Swagger annotations:** Add Go comments above the handler with `@Summary`, `@Description`, `@Param`, `@Success`, `@Failure`, `@Router`, etc.

## Step 2: Register the route

Add the route in `internal/handlers/handlers_routes.go`:

```go
func (a *App) routes() http.Handler {
    mux := http.NewServeMux()

    // ... existing routes ...

    // Public route (no auth)
    mux.HandleFunc("GET /users", a.listUsersHandler)

    // Protected route (with auth + permission)
    mux.Handle("POST /api/v1/users",
        utils.RequireAuth(a.Auth)(
            utils.RequirePermission(services.PermUsersManage)(
                http.HandlerFunc(a.createUserHandler),
            ),
        ),
    )

    return mux
}
```

### Route patterns

The project uses Go 1.22+ enhanced ServeMux patterns:

| Pattern | Example | Description |
|---|---|---|
| Exact path | `GET /health` | Matches exactly |
| Prefix | `GET /api/v1/` | Matches all under prefix |
| Path parameter | `PATCH /api/v1/org/members/{member_id}` | Extracts `member_id` from path |
| Method+path | `POST /api/v1/auth/login` | Method-specific routing |

Access path parameters with `r.PathValue("name")`.

## Step 3: Add permission (if needed)

If your endpoint needs a new permission:

1. Add it to the seed migration in `internal/database/database.go`:

```go
{
    name: "seed_default_permissions",
    sql: `
        INSERT INTO permissions (permission_key, module, description)
        VALUES
            ...
            ('users.read', 'users', 'Read user information')
        ON CONFLICT (permission_key) DO NOTHING`,
},
```

2. Add a constant in `internal/services/services_rbac.go`:

```go
const (
    // ... existing permissions ...
    PermUsersRead = "users.read"
)
```

3. Use the constant in route registration (see Step 2).

## Step 4: Regenerate Swagger spec

```bash
swag init --dir ./cmd/api,./internal/handlers --output ./docs --parseDependency --parseInternal
```

## Pattern summary

| Step | File | What to do |
|---|---|---|
| 1 | `internal/handlers/handlers_<category>.go` | Write handler function with Swagger annotations |
| 2 | `internal/handlers/handlers_routes.go` | Register route with middleware |
| 3 | `internal/database/database.go` | Add new permission to seed (if needed) |
| 4 | `internal/services/services_rbac.go` | Add permission constant (if needed) |
| 5 | — | Regenerate Swagger spec |