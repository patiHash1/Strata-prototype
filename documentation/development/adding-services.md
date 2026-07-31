# Adding Services

This guide walks through adding a new service (business logic + database access) to Strata.

## Step 1: Create the service file

Create `internal/services/services_<domain>.go`. Each service file contains four parts:

### 1. Domain types

```go
// internal/services/services_crm.go
package services

import (
    "context"
    "errors"
    "time"

    "github.com/google/uuid"
    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"
)

// ---- Types ----

type Lead struct {
    ID        uuid.UUID `json:"id"`
    OrgID     uuid.UUID `json:"org_id"`
    Name      string    `json:"name"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}
```

### 2. Repository (SQL layer)

```go
// ---- Repository ----

type crmRepository struct {
    pool *pgxpool.Pool
}

func newCRMRepository(pool *pgxpool.Pool) *crmRepository {
    return &crmRepository{pool: pool}
}

func (r *crmRepository) Create(ctx context.Context, l *Lead) error {
    l.ID = uuid.New()
    l.CreatedAt = time.Now()
    _, err := r.pool.Exec(ctx, `
        INSERT INTO leads (id, org_id, name, email, created_at)
        VALUES ($1, $2, $3, $4, $5)
    `, l.ID, l.OrgID, l.Name, l.Email, l.CreatedAt)
    return err
}

func (r *crmRepository) GetByID(ctx context.Context, id uuid.UUID) (*Lead, error) {
    l := &Lead{}
    err := r.pool.QueryRow(ctx, `
        SELECT id, org_id, name, email, created_at FROM leads WHERE id = $1
    `, id).Scan(&l.ID, &l.OrgID, &l.Name, &l.Email, &l.CreatedAt)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, nil
    }
    return l, err
}
```

### 3. Service (business logic)

```go
// ---- Service ----

type CRMService struct {
    repo *crmRepository
}

func NewCRMService(pool *pgxpool.Pool) *CRMService {
    return &CRMService{repo: newCRMRepository(pool)}
}

func (s *CRMService) Create(ctx context.Context, orgID uuid.UUID, name, email string) (*Lead, error) {
    // Business logic here (validation, domain rules)
    lead := &Lead{
        OrgID: orgID,
        Name:  name,
        Email: email,
    }
    if err := s.repo.Create(ctx, lead); err != nil {
        return nil, err
    }
    return lead, nil
}

// Domain errors
var (
    ErrLeadNotFound = errors.New("lead not found")
)
```

## Step 2: Wire the service into the App

### 2a. Add the field to `App` in `internal/handlers/handlers_server.go`

```go
type App struct {
    Config  config.Config
    DB      *database.DB
    Auth    *services.AuthService
    Users   *services.UserService
    Orgs    *services.OrgService
    RBAC    *services.RBACService
    Billing *services.BillingService
    Mailer  *services.Mailer
    CRM     *services.CRMService       // <-- new field
    server  *http.Server
}
```

### 2b. Add the constructor parameter to `New()`

```go
func New(
    cfg config.Config,
    db *database.DB,
    authSvc *services.AuthService,
    userSvc *services.UserService,
    orgSvc *services.OrgService,
    rbacSvc *services.RBACService,
    billingSvc *services.BillingService,
    mailerSvc *services.Mailer,
    crmSvc *services.CRMService,   // <-- new parameter
) *App {
    return &App{
        // ...
        CRM: crmSvc,
    }
}
```

### 2c. Create the service in `cmd/api/main.go`

```go
crmSvc := services.NewCRMService(db.Pool)

app := handlers.New(cfg, db, authSvc, userSvc, orgSvc, rbacSvc, billingSvc, mailerSvc, crmSvc)
```

## Step 3: Add the database table

Add the table migration to `internal/database/database.go`:

```go
{
    name: "create_leads",
    sql: `
        CREATE TABLE IF NOT EXISTS leads (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
            name VARCHAR(150) NOT NULL,
            email VARCHAR(255) NOT NULL,
            created_at TIMESTAMPTZ DEFAULT NOW()
        )`,
},
```

Migrations run on startup and are idempotent (`CREATE TABLE IF NOT EXISTS`).

## Step 4: Use the service from a handler

```go
// internal/handlers/handlers_crm.go
func (a *App) createLeadHandler(w http.ResponseWriter, r *http.Request) {
    var req createLeadRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        utils.WriteErr(w, http.StatusBadRequest, "invalid request body")
        return
    }

    claims := utils.GetClaims(r)
    orgID, _ := uuid.Parse(claims.OrgID)

    lead, err := a.CRM.Create(r.Context(), orgID, req.Name, req.Email)
    if err != nil {
        utils.WriteErr(w, http.StatusInternalServerError, "could not create lead")
        return
    }

    utils.WriteJSON(w, http.StatusCreated, utils.Envelope{"lead": lead})
}
```

## Conventions recap

| Convention | Guidance |
|---|---|
| File naming | `services_<domain>.go` |
| Repository | Unexported struct holding `*pgxpool.Pool` |
| Repository constructor | `new<Domain>Repository(pool)` |
| Service constructor | `New<Domain>Service(pool)` — accepts `*pgxpool.Pool` |
| Service methods | Always accept `context.Context` as the first argument |
| Domain errors | Package-level `var` with `errors.New(...)` |
| SQL style | Raw SQL, parameterized with `$1, $2, ...` placeholders |
| No rows handling | Return `(nil, nil)` when `pgx.ErrNoRows` — let the caller decide |