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
| Create enum type | `DO $$ BEGIN CREATE TYPE ... EXCEPTION WHEN duplicate_object THEN NULL; END $$` |
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
| 1 | `create_vehicle_status_enum` | Creates `vehicle_status` enum type |
| 2 | `create_shipment_status_enum` | Creates `shipment_status` enum type |
| 3 | `create_organizations` | Organizations table |
| 4 | `create_users` | Users table |
| 5 | `create_roles` | Roles table |
| 6 | `create_permissions` | Permissions table |
| 7 | `create_role_permissions` | Role-permission join table |
| 8 | `create_organization_members` | Organization membership table |
| 9 | `create_organization_invitations` | Invitation table |
| 10 | `create_api_keys` | API keys table |
| 11 | `create_subscriptions` | Subscriptions table |
| 12 | `create_crm_contacts` | CRM contacts table |
| 13 | `create_crm_deals` | CRM deals table |
| 14 | `create_crm_quotes` | CRM quotes table |
| 15 | `create_crm_helpdesk_tickets` | CRM helpdesk tickets table |
| 16 | `create_crm_campaigns` | CRM campaigns table |
| 17 | `create_subscription_plans` | Subscription plans table |
| 18 | `create_plan_features` | Plan features table |
| 19 | `create_chart_of_accounts` | Chart of accounts table |
| 20 | `create_journal_entries` | Journal entries table |
| 21 | `create_journal_items` | Journal items table |
| 22 | `create_invoices` | Invoices table |
| 23 | `create_expenses` | Expenses table |
| 24 | `create_fixed_assets` | Fixed assets table |
| 25 | `create_warehouses` | Warehouses table |
| 26 | `create_products` | Products table |
| 27 | `create_bill_of_materials` | Bill of materials table |
| 28 | `create_bom_components` | BOM components table |
| 29 | `create_fleet_vehicles` | Fleet vehicles table |
| 30 | `create_fleet_drivers` | Fleet drivers table |
| 31 | `create_shipments` | Shipments table |
| 32 | `create_fleet_telematics_logs` | Telematics logs table |
| 33 | `create_purchase_orders` | Purchase orders table |
| 34 | `seed_default_permissions` | Inserts default permissions |

## Limitations & roadmap

The current migration system is simple and runs on every startup. Known limitations:

- **No version tracking** — migrations run unconditionally; they rely on idempotency
- **No rollback** — schema changes cannot be reverted
- **No ordering guarantees** beyond slice order
- **No external migration tool** — a dedicated tool like `golang-migrate` is planned (`cmd/migrate/`)

For production use, a versioned migration runner (e.g., `golang-migrate`, `atlas`, or `goose`) is recommended.
