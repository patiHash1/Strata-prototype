# Services Layer

## Overview

The services layer (`internal/services/`) contains all business logic and database access. Each domain has its own file with three main components: types, repository, and service.

## Architecture pattern

Each service file follows a consistent pattern:

```go
package services

// 1. Domain types
type MyEntity struct { ... }

// 2. Repository (unexported)
type myRepository struct {
    pool *pgxpool.Pool
}

func newMyRepository(pool *pgxpool.Pool) *myRepository { ... }

func (r *myRepository) Create(ctx context.Context, e *MyEntity) error { ... }
func (r *myRepository) GetByID(ctx context.Context, id uuid.UUID) (*MyEntity, error) { ... }

// 3. Service (exported)
type MyService struct {
    repo *myRepository
}

func NewMyService(pool *pgxpool.Pool) *MyService { ... }

func (s *MyService) Create(ctx context.Context, ...) (*MyEntity, error) { ... }

// 4. Domain errors
var (
    ErrMyEntityNotFound = errors.New("entity not found")
)
```

## AuthService

**File:** `services_auth.go`

Handles authentication, JWT, and password operations.

| Method | Description |
|---|---|
| `NewAuthService(secret, issuer)` | Creates the auth service |
| `CreateToken(userID, orgID, roleID, permissions)` | Generates a signed JWT (24h expiry) |
| `ValidateToken(tokenStr)` | Parses and validates a JWT |
| `HashPassword(password)` | Bcrypt hash |
| `VerifyPassword(hash, password)` | Bcrypt compare |
| `GenerateRefreshToken()` | Opaque refresh token (UUID-based) |

**JWT Claims structure:**
```go
type Claims struct {
    UserID      string   `json:"user_id"`
    OrgID       string   `json:"org_id"`
    RoleID      string   `json:"role_id"`
    Permissions []string `json:"permissions"`
    jwt.RegisteredClaims
}
```

## UserService

**File:** `services_users.go`

Manages user accounts and organization memberships.

| Method | Description |
|---|---|
| `Create(ctx, email, passwordHash, fullName)` | Create a new user |
| `GetByEmail(ctx, email)` | Look up user by email |
| `GetByID(ctx, id)` | Look up user by ID |
| `AddMember(ctx, member)` | Add a member to an organization |
| `GetMember(ctx, orgID, userID)` | Get a member by org + user |
| `GetMemberByID(ctx, memberID)` | Get a member by membership ID |
| `UpdateMemberRole(ctx, memberID, roleID)` | Change a member's role |
| `DeactivateMember(ctx, memberID)` | Soft-delete (set `is_active = false`) |
| `RemoveMember(ctx, memberID)` | Hard-delete the membership row |
| `ListMembersByUser(ctx, userID)` | List all org memberships for a user |

**Domain errors:**
- `ErrEmailAlreadyExists` — duplicate email during registration
- `ErrMemberNotFound` — member record not found
- `ErrMemberNotInOrg` — member does not belong to the expected org

## OrgService

**File:** `services_orgs.go`

Manages organizations, invitations, and API keys.

| Method | Description |
|---|---|
| `Create(ctx, domainSlug, companyName)` | Create a new organization |
| `GetByDomainSlug(ctx, slug)` | Look up org by slug |
| `GetByID(ctx, id)` | Look up org by ID |
| `CreateInvitation(ctx, inv)` | Create an invitation record |
| `CreateAPIKey(ctx, key)` | Create an API key record |

**Domain errors:**
- `ErrOrgAlreadyExists` — duplicate domain slug during registration

**Organization model:**
```go
type Organization struct {
    ID              uuid.UUID
    DomainSlug      string
    CustomDomain    *string
    CompanyName     string
    DefaultCurrency string
    Timezone        string
    Status          OrgStatus
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

**OrgStatus values:** `active`, `suspended`, `pending_verification`

## RBACService

**File:** `services_rbac.go`

Manages roles, permissions, and role-permission assignments.

| Method | Description |
|---|---|
| `CreateRole(ctx, orgID, name, description, permissionIDs)` | Create a role with permissions |
| `GetRoleByID(ctx, id)` | Look up a role by ID |
| `ListRolesByOrg(ctx, orgID)` | List all roles in an org |
| `GetPermissionsByRole(ctx, roleID)` | Get permissions for a role |
| `GetPermissionKeysByRole(ctx, roleID)` | Get permission keys for a role (used when building JWT) |

**Permission constants:**
```go
PermUsersInvite   = "users.invite"
PermUsersManage   = "users.manage"
PermRBACManage    = "rbac.manage"
PermAPIKeysManage = "apikeys.manage"
PermBillingManage = "billing.manage"
```

## BillingService

**File:** `services_billing.go`

Manages subscriptions.

| Method | Description |
|---|---|
| `CreateOrUpgrade(ctx, orgID, planCode)` | Create or upgrade a subscription |
| `GetByOrgID(ctx, orgID)` | Get current subscription for an org |

**Domain errors:**
- `ErrSubNotFound` — subscription not found

**Subscription statuses:** `active`, `past_due`, `canceled`, `trialing`

## Mailer

**File:** `services_mailer.go`

Stub for transactional email sending. Currently logs to stdout:

```go
type Mailer struct{}

func (m *Mailer) SendInvitation(email, token string) error {
    log.Printf("[MAILER] invitation to %s with token %s", email, token)
    return nil
}
```

To integrate with a real email provider, replace the `Mailer` implementation with SendGrid, Mailgun, SMTP, or any other provider.

## Error handling conventions

- **Domain errors** are defined as package-level `var` using `errors.New()`
- **Services** return domain errors for business rule violations (e.g., duplicate email)
- **Services** return plain errors for unexpected failures (e.g., database connection issues)
- **Handlers** use `errors.Is()` to check for domain errors and map them to appropriate HTTP status codes
- **Handlers** never expose internal error details to the client

## Adding a new service

1. Create `internal/services/services_<domain>.go`
2. Define types, repository, and service struct
3. Add domain errors
4. Wire the service in `internal/handlers/handlers_server.go` (add field to `App` and constructor parameter)
5. Wire in `cmd/api/main.go` (create the service and pass to `handlers.New()`)
6. Use from handlers via `a.YourService.Method(...)`