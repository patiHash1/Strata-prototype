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
PermUsersInvite          = "users.invite"
PermUsersManage          = "users.manage"
PermRBACManage           = "rbac.manage"
PermAPIKeysManage        = "apikeys.manage"
PermBillingManage        = "billing.manage"
PermCRMLeadsWrite        = "crm.leads.write"
PermCRMQuotesWrite       = "crm.quotes.write"
PermCRMTicketsWrite      = "crm.tickets.write"
PermAccountingLedger     = "accounting.ledger.write"
PermAccountingInvoices   = "accounting.invoices.write"
PermExpensesSubmit       = "expenses.submit"
PermFleetTelematicsIngest = "fleet.telematics.ingest"
	PermFleetRoutesManage    = "fleet.routes.manage"
	PermInventoryRead        = "inventory.read"
	PermHRAttendanceWrite    = "hr.attendance.write"
	PermHRRecruitmentWrite   = "hr.recruitment.write"
	PermKnowledgeRead        = "knowledge.read"
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

## CRMService

**File:** `services_crm.go`

Manages CRM contacts (leads), deals, quotes, and AI-powered scoring/risk analysis.

| Method | Description |
|---|---|
| `CreateLead(ctx, orgID, firstName, lastName, email, companyName, estimatedDealSize)` | Creates a contact, triggers AI win probability scoring, and creates a linked deal |
| `AnalyzeContractRisk(ctx, orgID, quoteID, contractText)` | Runs AI risk analysis on a quote's contract text, returns risk score + flagged clauses |
| `CreateTicket(ctx, orgID, contactID, subject, description)` | Creates a helpdesk ticket with AI sentiment analysis, auto-priority, and suggested response |

**Domain errors:**
- `ErrQuoteNotFound` — quote not found
- `ErrQuoteNotInOrg` — quote does not belong to the requesting organization
- `ErrContactNotFound` — contact not found
- `ErrContactNotInOrg` — contact does not belong to the requesting organization

**Types:**

```go
type CRMContact struct {
    ID          uuid.UUID
    OrgID       uuid.UUID
    FirstName   string
    LastName    *string
    Email       *string
    Phone       *string
    CompanyName *string
    AssignedTo  *uuid.UUID
    CreatedAt   time.Time
}

type CRMDeal struct {
    ID               uuid.UUID
    OrgID            uuid.UUID
    ContactID        *uuid.UUID
    Title            string
    Amount           float64
    Stage            string
    AIWinProbability *int
    AssignedTo       *uuid.UUID
    CreatedAt        time.Time
}

type CRMQuote struct {
    ID          uuid.UUID
    OrgID       uuid.UUID
    DealID      *uuid.UUID
    QuoteNumber string
    TotalAmount float64
    AIRiskScore *float64
    CreatedAt   time.Time
}

type CRMHelpdeskTicket struct {
    ID                  uuid.UUID
    OrgID               uuid.UUID
    ContactID           *uuid.UUID
    Subject             string
    Description         string
    Priority            string
    Status              string
    AISentimentScore    *float64
    AISuggestedResponse *string
    AssignedTo          *uuid.UUID
    CreatedAt           time.Time
}

type FlaggedClause struct {
    Clause       string
    RiskLevel    string // low, medium, high, critical
    SuggestedFix string
}
```

**AI simulation:** The current implementation uses keyword-based heuristic analysis for both win probability scoring and contract risk analysis. Win probabilities fall in the 40–80 range. Contract analysis scans for high-risk keywords (indemnification, penalty, termination, confidential) and assigns weighted risk scores. In production, these would be replaced with external AI/ML service calls.

## AccountingService

**File:** `services_accounting.go`

Manages accounting operations: journal entries, invoice OCR processing, and expense submissions.

| Method | Description |
|---|---|
| `PostJournalEntry(ctx, orgID, entryDate, memo, items)` | Creates a journal entry with balanced debit/credit items |
| `ProcessInvoiceOCR(ctx, orgID, fileName, fileSize)` | Simulates AI-powered OCR extraction from an uploaded invoice |
| `SubmitExpense(ctx, orgID, userID, amount, category, receiptFileName)` | Creates an expense with AI fraud audit |

**Domain errors:**
- `ErrNoJournalItems` — journal entry must have at least one item
- `ErrUnbalancedEntry` — total debits must equal total credits
- `ErrAccountNotFound` — account not found
- `ErrAccountNotInOrg` — account does not belong to this organization

## SupplyChainService

**File:** `services_supplychain.go`

Manages supply chain, manufacturing, fleet telematics, and inventory operations. Also handles API key validation for machine-to-machine endpoints.

**Constructor:** `NewSupplyChainService(pool *pgxpool.Pool, authSvc *AuthService)`

Accepts an `*AuthService` for bcrypt verification of API keys during authentication.

| Method | Description |
|---|---|
| `IngestTelemetry(ctx, orgID, input)` | Processes vehicle telemetry data, triggers AI predictive alerts for high engine temp (>110°C) or speed (>130 km/h) |
| `OptimizeRoutes(ctx, orgID, shipmentIDs, vehicleIDs)` | Generates AI-optimized delivery routes with GeoJSON waypoints, ETA, and carbon offset |
| `GetReorderPredictions(ctx, orgID, warehouseID)` | Returns AI-driven stockout predictions and reorder recommendations |
| `ValidateAPIKey(ctx, rawKey)` | bcrypt-verifies an API key against stored hashes, returns org ID and scopes |

**Domain errors:**
- `ErrVehicleNotFound` — vehicle not found by VIN
- `ErrNoShipmentsProvided` — empty shipment IDs array
- `ErrNoVehiclesProvided` — empty vehicle IDs array
- `ErrShipmentsNotFound` — no matching shipments
- `ErrVehiclesNotFound` — no matching vehicles
- `ErrAPIKeyInvalid` — invalid or expired API key

**Types:**

```go
type FleetVehicle struct {
    ID           uuid.UUID
    OrgID        uuid.UUID
    VIN          string
    LicensePlate string
    Make         string
    Model        string
    Status       string    // active, maintenance, decommissioned
    CreatedAt    time.Time
}

type Shipment struct {
    ID                 uuid.UUID
    OrgID              uuid.UUID
    TrackingNumber     string
    OriginAddress      string
    DestinationAddress string
    Status             string    // pending, in_transit, delivered, cancelled
    AssignedVehicleID  *uuid.UUID
    AssignedDriverID   *uuid.UUID
    CreatedAt          time.Time
}

type TelemetryIngestResult struct {
    Status                     string
    ProcessedAt                time.Time
    AIPredictiveAlertTriggered bool
}

type RoutePlan struct {
    RoutePlanID        uuid.UUID
    OptimizedWaypoints []Waypoint  // GeoJSON points
    PredictedETA       time.Time
    CarbonOffsetKg     float64
}

type StockoutPrediction struct {
    ProductID             uuid.UUID
    SKU                   string
    CurrentStock          int
    PredictedStockoutDays int
    RecommendedReorderQty int
}
```

**AI simulation:** Telemetry alerts are triggered by rule-based thresholds (engine temp > 110°C, speed > 130 km/h). Route optimization generates simulated GeoJSON paths with randomized ETAs. Reorder predictions use simulated current stock levels with randomized daily consumption rates and 20–50% safety buffers. All AI features are designed to be replaced with real ML/optimization services in production.

## HRService

**File:** `services_hr.go`

Manages HR operations: employee attendance with geofence validation, AI-powered resume parsing and job matching, and RAG-based knowledge base semantic search.

| Method | Description |
|---|---|
| `ClockIn(ctx, orgID, userID, latitude, longitude)` | Records an attendance clock-in for the employee linked to the user, validates geofence, returns log ID + geofence status |
| `ParseResume(ctx, orgID, jobDescriptionID, resumeBytes, fileName)` | Simulates AI extraction of candidate name, email, and skills from a resume file, scores match against a job description, stores the application |
| `SearchKnowledge(ctx, orgID, query)` | Performs semantic search over the org's knowledge base, returns an AI-synthesized answer with source documents and relevance scores |

**Domain errors:**
- `ErrEmployeeNotFound` — no employee record found for the given user in the organization

**Types:**

```go
type Employee struct {
    ID           uuid.UUID
    OrgID        uuid.UUID
    UserID       *uuid.UUID
    EmployeeCode string
    Department   *string
    JobTitle     *string
    Salary       *float64
    HiredAt      time.Time
}

type AttendanceLog struct {
    ID           uuid.UUID
    OrgID        uuid.UUID
    EmployeeID   uuid.UUID
    ClockIn      time.Time
    ClockOut     *time.Time
    LocationLat  *float64
    LocationLong *float64
}

type AttendanceClockInResult struct {
    AttendanceLogID  uuid.UUID
    ClockIn          time.Time
    IsWithinGeofence bool
}

type JobApplication struct {
    ID            uuid.UUID
    OrgID         uuid.UUID
    CandidateName string
    Email         string
    ResumeURL     string
    AIMatchScore  *int
    Status        string
    CreatedAt     time.Time
}

type KnowledgeBaseDocument struct {
    ID                uuid.UUID
    OrgID             uuid.UUID
    Title             string
    Content           string
    VectorEmbeddingID *string
    CreatedAt         time.Time
}

type ResumeParseResult struct {
    CandidateName   string
    Email           string
    ExtractedSkills []string
    AIMatchScore    int
}

type SourceDocument struct {
    Title          string
    RelevanceScore float64
}

type KnowledgeSearchResult struct {
    AIAnswer        string
    SourceDocuments []SourceDocument
}
```

**AI simulation:** Geofence validation uses a distance-from-center calculation against a simulated office location (San Francisco). Resume parsing extracts candidate name from the file name and randomly selects 3–7 skills from a pool of 30 common tech skills. Match scoring uses a base score of 40 + (skills × 3) with ±10 jitter. Knowledge search uses PostgreSQL `ILIKE` for full-text search with term-overlap relevance scoring. All AI features are designed to be replaced with real ML/NLP services in production.

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
