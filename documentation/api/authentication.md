# Authentication

## Overview

Authentication in Strata uses **two modes**:

1. **JWT (JSON Web Tokens)** signed with HMAC-SHA256 — for human-facing endpoints
2. **API keys** validated via bcrypt — for machine-to-machine endpoints (e.g., telemetry ingestion)

The `AuthService` in `internal/services/services_auth.go` handles JWT operations and password management. API key validation is handled by `SupplyChainService.ValidateAPIKey()`.

### JWT claims

```json
{
    "user_id":      "550e8400-e29b-41d4-a716-446655440001",
    "org_id":       "550e8400-e29b-41d4-a716-446655440000",
    "role_id":      "550e8400-e29b-41d4-a716-446655440002",
    "permissions":  ["users.invite", "rbac.manage"],
    "iss":          "strata",
    "iat":          1690000000,
    "exp":          1690086400
}
```

| Claim | Type | Description |
|---|---|---|
| `user_id` | string | UUID of the authenticated user |
| `org_id` | string | UUID of the user's organization |
| `role_id` | string | UUID of the user's assigned role |
| `permissions` | string[] | Permission keys granted by the role |
| `iss` | string | JWT issuer (configurable) |
| `iat` | number | Issued-at timestamp (epoch) |
| `exp` | number | Expiration timestamp (epoch) — tokens are valid for 24 hours |

### API key validation

API keys are stored as bcrypt hashes in the `api_keys` table. When a request arrives with an `X-API-Key` header:

1. All active (non-expired) API keys are fetched from the database
2. The raw key is bcrypt-verified against each stored hash
3. On match, the key's org ID and scopes are injected into the request context

API keys are created via `POST /api/v1/org/api-keys` with specific scope permissions.

---

## Endpoints

### Register a new organization

Creates a new organization with an owner user. Returns a JWT access token.

```http
POST /api/v1/auth/register
Content-Type: application/json

{
    "company_name": "Acme Corp",
    "domain_slug": "acme-corp",
    "owner_email": "owner@acme.com",
    "owner_password": "securepassword123",
    "owner_full_name": "Jane Doe"
}
```

**Response** `201 Created`:
```json
{
    "org_id": "550e8400-e29b-41d4-a716-446655440000",
    "user_id": "550e8400-e29b-41d4-a716-446655440001",
    "access_token": "eyJhbGciOiJIUzI1NiIs..."
}
```

**Validation rules:**
| Field | Rule |
|---|---|
| `company_name` | Required, non-blank |
| `domain_slug` | Required, lowercase alphanumeric + hyphens, max 100 chars |
| `owner_email` | Required, valid email format |
| `owner_password` | Required, minimum 8 characters |
| `owner_full_name` | Required, non-blank |

**Side effects:**
1. Organization is created with status `active`
2. User is created with the provided password (bcrypt-hashed)
3. An "Admin" role is created with no permissions assigned (owner gets full access via code)
4. User is added as an organization member with the Admin role
5. A JWT is generated and returned

**Errors:**
| Status | Condition |
|---|---|
| `400` | Missing or invalid fields |
| `409` | Domain slug or email already exists |
| `500` | Internal server error |

---

### Login

Authenticates with email and password. Returns a JWT access token, refresh token, and user profile.

```http
POST /api/v1/auth/login
Content-Type: application/json

{
    "email": "owner@acme.com",
    "password": "securepassword123"
}
```

**Response** `200 OK`:
```json
{
    "access_token": "eyJhbGciOiJIUzI1NiIs...",
    "refresh_token": "a1b2c3d4e5f6...",
    "user_profile": {
        "id": "550e8400-e29b-41d4-a716-446655440001",
        "email": "owner@acme.com",
        "full_name": "Jane Doe"
    }
}
```

**MFA flow:**
If the user has MFA enabled, the first request without `mfa_code` returns a special response:

```json
{
    "mfa_required": true,
    "message": "MFA code required"
}
```

The client should then prompt for an MFA code and retry:

```http
POST /api/v1/auth/login
Content-Type: application/json

{
    "email": "owner@acme.com",
    "password": "securepassword123",
    "mfa_code": "123456"
}
```

**Token permissions:** On login, the user's role is resolved and all permission keys are embedded in the JWT. This allows the `RequirePermission` middleware to check access without additional database queries on every request.

**Errors:**
| Status | Condition |
|---|---|
| `400` | Missing email or password |
| `401` | Invalid email or password |
| `401` | No organization membership found |
| `500` | Internal server error |

---

## Using the JWT

Include the JWT in the `Authorization` header for all protected requests:

```http
GET /api/v1/org/invitations
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

## Using API keys

Include the API key in the `X-API-Key` header for machine-to-machine endpoints:

```http
POST /api/v1/fleet/telematics/ingest
X-API-Key: a1b2c3d4e5f6...
```

API keys must be created with the appropriate scopes for the endpoint they will access. For example, the telemetry ingestion endpoint requires the `fleet.telematics.ingest` scope.

## Token lifecycle

- **JWT Expiration:** 24 hours from issuance
- **Refresh:** The `refresh_token` returned at login can be used to obtain a new JWT (refresh endpoint is not yet implemented — coming soon)
- **JWT Revocation:** Not yet implemented. In production, a token blacklist or short-lived tokens with refresh rotation should be used.
- **API Key Expiration:** Configurable at creation time via `expires_in_days`. Keys with no expiration are valid indefinitely (until manually revoked).
