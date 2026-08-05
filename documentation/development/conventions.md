# Code Conventions

## Go version

The project uses **Go 1.26** (match `go.mod`). All Go code must be compatible with this version.

## Package structure

```
internal/
├── services/     # Business logic + SQL — one file per domain
├── handlers/     # HTTP handlers + routes + App struct
├── utils/        # Pure helpers with no project dependencies
├── config/       # Configuration struct loaded from env vars
├── database/     # pgx connection pool + embedded migrations
│   └── migrations/  # Numbered .up.sql files
└── env/          # Safe environment variable getters
```

## File naming

| Package | Format | Example |
|---|---|---|
| services | `services_<domain>.go` | `services_users.go`, `services_orgs.go`, `services_supplychain.go` |
| handlers | `handlers_<category>.go` | `handlers_auth.go`, `handlers_org.go`, `handlers_supplychain.go` |
| utils | `<function>.go` | `response.go`, `middleware.go`, `validator.go` |

## Naming conventions

### Services

- **Repository:** `type <domain>Repository struct` (unexported)
- **Repository constructor:** `func new<Domain>Repository(pool *pgxpool.Pool) *<domain>Repository`
- **Service:** `type <Domain>Service struct` (exported)
- **Service constructor:** `func New<Domain>Service(pool *pgxpool.Pool) *<Domain>Service`
- **Service methods:** Accept `context.Context` as the first argument

### Handlers

- **Handler function:** `func (a *App) actionHandler(w http.ResponseWriter, r *http.Request)`
- **Request struct:** `type actionRequest struct` (unexported, defined above the handler)
- **Response struct:** `type ActionResponse struct` (exported, used in Swagger annotations)

### Errors

- **Domain errors:** Package-level `var` with `errors.New()`:
  ```go
  var ErrMemberNotFound = errors.New("organization member not found")
  ```
- **Error naming:** `Err<Description>` for exported, `err<Description>` for unexported

## Code style

### Imports

Group imports in three sections separated by blank lines:

```go
import (
    "encoding/json"
    "net/http"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"

    "github.com/patiHash1/Strata-prototype/internal/utils"
)
```

### Handler pattern

Every handler should follow this pattern:

```go
func (a *App) actionHandler(w http.ResponseWriter, r *http.Request) {
    // 1. Parse request body
    var req actionRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
        return
    }

    // 2. Validate input
    if !utils.NotBlank(req.Name) {
        utils.WriteErr(w, http.StatusBadRequest, "name is required")
        return
    }

    // 3. Extract claims (JWT or API key, depending on endpoint)
    claims := utils.GetClaims(r)
    if claims == nil {
        // For API key endpoints, use utils.GetAPIKeyClaims(r) instead
        utils.WriteErr(w, http.StatusUnauthorized, "authentication required")
        return
    }

    // 4. Call service
    result, err := a.Service.Method(r.Context(), args)
    if err != nil {
        utils.WriteErr(w, http.StatusInternalServerError, "could not do thing")
        return
    }

    // 5. Write response
    utils.WriteJSON(w, http.StatusOK, utils.Envelope{
        "key": result,
    })
}
```

### Service method pattern

```go
func (s *Service) Method(ctx context.Context, arg1 Type) (*ResultType, error) {
    // Business logic
    // Return domain errors for expected failures
    // Return plain errors for unexpected failures
}
```

## Response conventions

- **Success:** `{"key": value}` — the response data is a top-level key in the envelope
- **Error:** `{"error": "message"}`
- **Pagination:** Include a `pagination` key alongside the data key
- **Always** use `utils.WriteJSON` / `utils.WriteErr` with the `utils.Envelope` type

## Error handling

- **Services** return errors; handlers translate them to HTTP responses
- **Domain errors** are checked with `errors.Is()` in handlers
- **Handlers** never expose internal error details to the client
- **Handlers** use `http.StatusInternalServerError` with a generic message for unexpected errors

## Comments

- **Go doc comments** on all exported types, functions, and methods
- **Swagger annotations** on handler functions with `@Summary`, `@Description`, `@Param`, `@Success`, `@Failure`, `@Router`
- **Inline comments** only for non-obvious intent, constraints, or tradeoffs

## Imports in handlers

Use the `utils` package for all shared helpers — never import `services` directly from a handler except for `services.Perm*` constants.

## No global state

Everything lives on the `App` struct or is injected via constructor. No global variables, no `init()` functions, no package-level state.

## SQL style

- Use raw SQL with parameterized queries (`$1`, `$2`, ...)
- Use `QueryRow` for single-row results
- Use `Query` + `rows.Next()` for multi-row results
- Handle `pgx.ErrNoRows` by returning `(nil, nil)` — the caller decides what "not found" means
- Use `Exec` for insert/update/delete operations

## Git conventions

- Feature branches: `feature/<description>`
- Bug fixes: `fix/<description>`
- Commit messages: imperative mood, capitalized, 50-char subject line

## Testing conventions

- Test files alongside the code they test: `services_<domain>_test.go`
- Test functions: `Test<FunctionName>`
- Use table-driven tests where appropriate
- Use `t.Run()` for subtests
