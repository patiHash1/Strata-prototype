# Adding Migrations

## Overview

Database migrations live in `internal/database/database.go` in the `Migrate()` function. They are simple, ordered SQL statements executed at server startup.

```go
func (db *DB) Migrate(ctx context.Context) error {
    migrations := []struct {
        name string
        sql  string
    }{
        {
            name: "create_organizations",
            sql:  `CREATE TABLE IF NOT EXISTS organizations (...)`,
        },
        // ...
    }

    for _, m := range migrations {
        if _, err := db.Pool.Exec(ctx, m.sql); err != nil {
            return fmt.Errorf("migration %q: %w", m.name, err)
        }
        fmt.Printf("  ✓ %s\n", m.name)
    }

    return nil
}
```

## Adding a new migration

1. Add a new entry to the `migrations` slice in `internal/database/database.go`
2. Give it a descriptive, unique `name`
3. Write idempotent SQL (`CREATE TABLE IF NOT EXISTS`, `ON CONFLICT DO NOTHING`, etc.)

```go
{
    name: "create_leads",
    sql: `
        CREATE TABLE IF NOT EXISTS leads (
            id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
            org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
            name VARCHAR(150) NOT NULL,
            email VARCHAR(255) NOT NULL,
            status VARCHAR(50) DEFAULT 'new',
            created_at TIMESTAMPTZ DEFAULT NOW()
        )`,
},
```

## Idempotency requirements

All migrations **must be idempotent** because they run on every startup:

| Operation | Safe pattern |
|---|---|
| Create table | `CREATE TABLE IF NOT EXISTS` |
| Insert seed data | `INSERT ... ON CONFLICT (key) DO NOTHING` |
| Add column | `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` |
| Add index | `CREATE INDEX IF NOT EXISTS` |

## Adding seed data

Seed data (like default permissions) uses `ON CONFLICT DO NOTHING`:

```go
{
    name: "seed_default_permissions",
    sql: `
        INSERT INTO permissions (permission_key, module, description)
        VALUES
            ('users.invite',   'users', 'Invite team members to the organization'),
            ('users.manage',   'users', 'Manage organization members (update role, deactivate, remove)'),
            ('rbac.manage',    'rbac',  'Create and manage roles and permissions'),
            ('apikeys.manage', 'apikeys', 'Generate and revoke API keys'),
            ('billing.manage', 'billing', 'Manage subscriptions and billing')
        ON CONFLICT (permission_key) DO NOTHING`,
},
```

## Current migration list

| Order | Name | Purpose |
|---|---|---|
| 1 | `create_enums_if_missing` | Placeholder (currently no-op) |
| 2 | `create_organizations` | Organizations table |
| 3 | `create_users` | Users table |
| 4 | `create_roles` | Roles table |
| 5 | `create_permissions` | Permissions table |
| 6 | `create_role_permissions` | Role-permission join table |
| 7 | `create_organization_members` | Organization membership table |
| 8 | `create_organization_invitations` | Invitation table |
| 9 | `create_api_keys` | API keys table |
| 10 | `create_subscriptions` | Subscriptions table |
| 11 | `seed_default_permissions` | Inserts default permissions |

## Limitations & roadmap

The current migration system is simple and runs on every startup. Known limitations:

- **No version tracking** — migrations run unconditionally; they rely on idempotency
- **No rollback** — schema changes cannot be reverted
- **No ordering guarantees** beyond slice order
- **No external migration tool** — a dedicated tool like `golang-migrate` is planned (`cmd/migrate/`)

For production use, a versioned migration runner (e.g., `golang-migrate`, `atlas`, or `goose`) is recommended.