# Account Management

## Overview

The Account API allows authenticated users to manage their own profile and view their organization memberships. These endpoints are **self-referential** — users can only act on their own account, not on other users. This is distinct from the admin-level user management endpoints (which require `users.manage` permission).

All Account endpoints require a valid JWT (Bearer token) obtained via login or registration.

---

## Endpoints

### Get own profile

Returns the authenticated user's profile information.

```http
GET /api/v1/account
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Response** `200 OK`:
```json
{
    "id": "550e8400-e29b-41d4-a716-446655440001",
    "email": "alice@example.com",
    "full_name": "Alice Johnson",
    "phone_number": "+1-555-0100",
    "mfa_enabled": false,
    "created_at": "2025-01-15T10:30:00Z"
}
```

| Field | Type | Description |
|---|---|---|
| `id` | string | UUID of the user |
| `email` | string | User's email address |
| `full_name` | string | User's full name |
| `phone_number` | string \| null | User's phone number (optional) |
| `mfa_enabled` | boolean | Whether MFA is enabled |
| `created_at` | string | ISO 8601 timestamp of account creation |

**Errors:**
| Status | Condition |
|---|---|
| `401` | Missing or invalid JWT |
| `404` | User not found |
| `500` | Internal server error |

---

### Update own profile

Partially updates the authenticated user's profile fields. Only the fields provided in the request body are updated — omitted fields remain unchanged.

```http
PATCH /api/v1/account
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
Content-Type: application/json

{
    "full_name": "Alice Smith",
    "phone_number": "+1-555-0200"
}
```

**Request body:**

| Field | Type | Required | Description |
|---|---|---|---|
| `full_name` | string | No | New full name (must not be blank if provided) |
| `email` | string | No | New email address (must be valid format if provided) |
| `phone_number` | string | No | New phone number |

At least one field must be provided.

**Response** `200 OK` — Returns the updated profile (same shape as GET).

**Errors:**
| Status | Condition |
|---|---|
| `400` | No fields provided, or invalid field values |
| `401` | Missing or invalid JWT |
| `409` | Email already taken by another user |
| `500` | Internal server error |

---

### Delete own account

Permanently deletes the authenticated user's account. This is irreversible.

```http
DELETE /api/v1/account
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Response** `204 No Content` — Account successfully deleted.

**Errors:**
| Status | Condition |
|---|---|
| `401` | Missing or invalid JWT |
| `500` | Internal server error |

> ⚠️ **Cascading deletes:** The `users` table has foreign key relationships with `organization_members` (via `ON DELETE CASCADE`). Deleting a user will also remove all their organization memberships. Other related data (e.g., CRM leads, journal entries) should be handled by the application layer before deletion in a production environment.

---

### List my organizations

Returns all organization memberships for the authenticated user.

```http
GET /api/v1/account/organizations
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**Response** `200 OK`:
```json
{
    "organizations": [
        {
            "org_id": "550e8400-e29b-41d4-a716-446655440000",
            "role_id": "660e8400-e29b-41d4-a716-446655440001",
            "is_active": true,
            "joined_at": "2025-01-15T10:30:00Z"
        }
    ]
}
```

| Field | Type | Description |
|---|---|---|
| `organizations` | array | List of organization memberships |
| `organizations[].org_id` | string | UUID of the organization |
| `organizations[].role_id` | string | UUID of the user's role in this org |
| `organizations[].is_active` | boolean | Whether the membership is active |
| `organizations[].joined_at` | string | ISO 8601 timestamp of when the user joined |

**Errors:**
| Status | Condition |
|---|---|
| `401` | Missing or invalid JWT |
| `500` | Internal server error |

---

## Authentication

All Account endpoints require a valid JWT in the `Authorization` header:

```http
Authorization: Bearer <token>
```

The JWT's `user_id` claim determines which account is being accessed. Users cannot access or modify other users' accounts through these endpoints.

## Implementation notes

- **Service layer:** Account operations reuse the existing `UserService` in `internal/services/services_users.go`. New methods added: `UpdateProfile()` and `DeleteAccount()`.
- **Repository layer:** Dynamic partial `UPDATE` query with parameterized column building. Email uniqueness is validated at the service layer on change.
- **No new permissions:** These endpoints are self-referential, so only JWT authentication is required — no `RequirePermission` gate.
- **No new database tables:** Uses the existing `users` and `organization_members` tables.
- **Handler file:** `internal/handlers/handlers_account.go`