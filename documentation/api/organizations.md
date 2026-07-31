# Organizations

## Overview

The Organizations module manages multi-tenant organization records, invitations, roles, and API keys. All data is scoped to an organization identified by the `org_id` claim in the JWT.

### Organization model

```go
type Organization struct {
    ID              uuid.UUID  // Primary key
    DomainSlug      string     // Unique slug (e.g., "acme-corp")
    CustomDomain    *string    // Optional custom domain
    CompanyName     string     // Display name
    DefaultCurrency string     // Default: "USD"
    Timezone        string     // Default: "UTC"
    Status          OrgStatus  // "active", "suspended", "pending_verification"
    CreatedAt       time.Time
    UpdatedAt       time.Time
}
```

### Organization statuses

| Status | Description |
|---|---|
| `active` | Organization is fully operational |
| `suspended` | Organization access is suspended |
| `pending_verification` | Awaiting email or domain verification |

---

## Endpoints

### Invite a team member

Sends an invitation to join the organization. Requires `users.invite` permission.

```http
POST /api/v1/org/invitations
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "email": "newuser@example.com",
    "role_id": "550e8400-e29b-41d4-a716-446655440002"
}
```

**Response** `201 Created`:
```json
{
    "invitation_id": "550e8400-e29b-41d4-a716-446655440010",
    "token": "a1b2c3d4e5f6...",
    "expires_at": "2026-08-08T12:00:00Z"
}
```

**Validation:**
| Field | Rule |
|---|---|
| `email` | Required, valid email format |
| `role_id` | Required, valid UUID |

**Behavior:**
- An invitation token is generated (two concatenated UUIDs)
- The invitation expires after 7 days
- The `created_by` field is populated from the authenticated user's ID
- An email is sent via the `Mailer` service (currently a stub that logs to stdout)

**Errors:**
| Status | Condition |
|---|---|
| `400` | Invalid email or role_id |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `500` | Could not create invitation |

---

### Create a dynamic role

Creates a new role with assigned permissions. Requires `rbac.manage` permission.

```http
POST /api/v1/org/roles
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "name": "Sales Manager",
    "description": "Manages the sales pipeline",
    "permission_ids": [
        "550e8400-e29b-41d4-a716-446655440020",
        "550e8400-e29b-41d4-a716-446655440021"
    ]
}
```

**Response** `201 Created`:
```json
{
    "role_id": "550e8400-e29b-41d4-a716-446655440030",
    "name": "Sales Manager",
    "assigned_permissions_count": 2
}
```

**Errors:**
| Status | Condition |
|---|---|
| `400` | Missing name, no permission_ids, or invalid permission_id |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `500` | Could not create role |

---

### Create an API key

Generates a new API key for machine-to-machine integrations. Requires `apikeys.manage` permission.

```http
POST /api/v1/org/api-keys
Content-Type: application/json
Authorization: Bearer <jwt>

{
    "name": "CI Pipeline",
    "scopes": ["users:read", "billing:read"],
    "expires_in_days": 90
}
```

**Response** `201 Created`:
```json
{
    "api_key_id": "550e8400-e29b-41d4-a716-446655440040",
    "plain_text_secret": "a1b2c3d4e5f6..."
}
```

> **Security note:** The `plain_text_secret` is shown only once in this response. It cannot be retrieved later. Store it securely.

**Validation:**
| Field | Rule |
|---|---|
| `name` | Required, non-blank |
| `scopes` | Optional, defaults to empty array |
| `expires_in_days` | Optional, no expiration if 0 or omitted |

**Behavior:**
- The secret key is generated as two concatenated UUIDs
- The key is bcrypt-hashed before storage — the raw secret is never stored
- The key hash is stored in the `api_keys` table
- Expiration is optional; if provided, `expires_at` is set to `now + expires_in_days`

**Errors:**
| Status | Condition |
|---|---|
| `400` | Missing name |
| `401` | Missing or invalid JWT |
| `403` | Insufficient permissions |
| `500` | Could not create API key |

---

## Permission system

### Pre-defined permissions

| Key | Module | Description |
|---|---|---|
| `users.invite` | users | Invite team members to the organization |
| `users.manage` | users | Manage organization members (update role, deactivate, remove) |
| `rbac.manage` | rbac | Create and manage roles and permissions |
| `apikeys.manage` | apikeys | Generate and revoke API keys |
| `billing.manage` | billing | Manage subscriptions and billing |

### Permission checks

Permissions are checked at the route level using the `RequirePermission` middleware, which reads the `permissions` claim from the JWT. The JWT is populated with the user's permissions at login time.

```go
mux.Handle("POST /api/v1/org/invitations",
    utils.RequireAuth(a.Auth)(
        utils.RequirePermission(services.PermUsersInvite)(
            http.HandlerFunc(a.inviteHandler),
        ),
    ),
)
```

The middleware grants access if the user has **any** of the required permissions (OR logic). For AND logic, chain multiple `RequirePermission` calls.