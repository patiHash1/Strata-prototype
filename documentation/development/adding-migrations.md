# Adding Migrations

## Overview

Database migrations are **embedded SQL files** loaded via Go's `embed.FS` package. Each migration is a numbered `.up.sql` file in `internal/database/migrations/`, executed in lexicographic order at server startup. Every migration also has a matching `.down.sql` file (for reference/rollback), though only `.up.sql` files are embedded and executed.

The loader lives in `internal/database/migrations.go`:

```go
//go:embed migrations/*.up.sql
var migrationFS embed.FS

func loadMigrations() ([]migration, error) {
    entries, _ := migrationFS.ReadDir("migrations")
    sort.Slice(entries, func(i, j int) bool {
        return entries[i].Name() < entries[j].Name()
    })
    // Read each file, derive name from filename, return sorted slice
}
```

The `Migrate()` method in `internal/database/database.go` calls `loadMigrations()` and executes each SQL file **only if it has not already been recorded** in the `schema_migrations` table:

```go
func (db *DB) Migrate(ctx context.Context) error {
    // Ensure the tracking table exists.
    CREATE TABLE IF NOT EXISTS schema_migrations (
        version VARCHAR(255) PRIMARY KEY,
        applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    )

    migrations, err := loadMigrations()
    // ...
    for _, m := range migrations {
        // Skip the schema_migrations table creation itself.
        if m.name == "create_schema_migrations" { continue }

        // Check if already applied.
        SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)
        if alreadyApplied { continue }

        // Apply the migration, then record it.
        db.Pool.Exec(ctx, m.sql)
        INSERT INTO schema_migrations (version, applied_at) VALUES ($1, NOW())
    }
}
```

Migrations are **version-tracked**: each is applied once and recorded, so only new migrations run on subsequent startups. This differs from the older idempotent-only approach.

## Adding a new migration

1. Create a new `.up.sql` file in `internal/database/migrations/` with the next available sequence number
2. Write idempotent SQL (`CREATE TABLE IF NOT EXISTS`, `ON CONFLICT DO NOTHING`, etc.)
3. Rebuild — the file is embedded at compile time via `embed.FS`

**Example:** Adding a `leads` table:

```sql
-- internal/database/migrations/000068_create_leads.up.sql
CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(150) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'new',
    created_at TIMESTAMPTZ DEFAULT NOW()
);
```

## File naming convention

Files follow the pattern `NNNNNN_<descriptive_name>.up.sql` (with a matching `NNNNNN_<descriptive_name>.down.sql`):

| Prefix | Purpose |
|---|---|
| `000001`–`000005` | Custom enum types |
| `000006`–`000067` | Core tables, indexes, and seed data |
| `000069`–`000075` | Super-admin subsystem, API key prefix, schema tracking, SOC events, last-login tracking |

The six-digit zero-padded prefix ensures lexicographic sort order matches execution order. The descriptive name after the first underscore is used as the console log label (e.g., `✓ create_leads`).

## Idempotency requirements

Although migrations are version-tracked (each runs once), the SQL is still written idempotently as a safety net for partially-applied or manually-managed databases:

| Operation | Safe pattern |
|---|---|
| Create table | `CREATE TABLE IF NOT EXISTS` |
| Create enum type | `DO $$ BEGIN CREATE TYPE ... EXCEPTION WHEN duplicate_object THEN NULL; END $$` |
| Insert seed data | `INSERT ... ON CONFLICT (key) DO NOTHING` |
| Add column | `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` |
| Add index | `CREATE INDEX IF NOT EXISTS` |

## Current migration list (74 `.up.sql` files)

| # | Name | Purpose |
|---|---|---|
| 1 | `create_org_status_enum` | Creates `org_status` enum type |
| 2 | `create_subscription_status_enum` | Creates `subscription_status` enum type |
| 3 | `create_ticket_priority_enum` | Creates `ticket_priority` enum type |
| 4 | `create_vehicle_status_enum` | Creates `vehicle_status` enum type |
| 5 | `create_shipment_status_enum` | Creates `shipment_status` enum type |
| 6 | `create_organizations` | Organizations table |
| 7 | `create_users` | Users table |
| 8 | `create_roles` | Roles table |
| 9 | `create_permissions` | Permissions table |
| 10 | `create_role_permissions` | Role-permission join table |
| 11 | `create_organization_members` | Organization membership table |
| 12 | `create_organization_invitations` | Invitation table |
| 13 | `create_api_keys` | API keys table |
| 14 | `create_subscription_plans` | Subscription plans table |
| 15 | `create_plan_features` | Plan features table |
| 16 | `create_subscriptions` | Subscriptions table |
| 17 | `create_crm_contacts` | CRM contacts table |
| 18 | `create_crm_deals` | CRM deals table |
| 19 | `create_crm_quotes` | CRM quotes table |
| 20 | `create_crm_helpdesk_tickets` | CRM helpdesk tickets table |
| 21 | `create_crm_campaigns` | CRM campaigns table |
| 22 | `create_field_sales_visits` | Field sales visits table |
| 23 | `add_campaign_status` | Adds `status` column to campaigns |
| 24 | `create_chart_of_accounts` | Chart of accounts table |
| 25 | `create_journal_entries` | Journal entries table |
| 26 | `create_journal_items` | Journal items table |
| 27 | `create_invoices` | Invoices table |
| 28 | `create_expenses` | Expenses table |
| 29 | `create_fixed_assets` | Fixed assets table |
| 30 | `create_tax_rates` | Tax rates table |
| 31 | `create_warehouses` | Warehouses table |
| 32 | `create_products` | Products table |
| 33 | `create_bill_of_materials` | Bill of materials table |
| 34 | `create_bom_components` | BOM components table |
| 35 | `create_fleet_vehicles` | Fleet vehicles table |
| 36 | `create_fleet_drivers` | Fleet drivers table |
| 37 | `create_shipments` | Shipments table |
| 38 | `create_fleet_telematics_logs` | Telematics logs table |
| 39 | `create_purchase_orders` | Purchase orders table |
| 40 | `create_work_orders` | Work orders table |
| 41 | `create_employees` | Employees table |
| 42 | `create_attendance_logs` | Attendance logs table |
| 43 | `create_payroll_runs` | Payroll runs table |
| 44 | `create_job_applications` | Job applications table |
| 45 | `create_knowledge_base_documents` | Knowledge base documents table |
| 46 | `create_ai_copilot_conversations` | AI copilot conversations table |
| 47 | `create_lowcode_workflows` | Low-code workflow definitions table |
| 48 | `create_iot_devices` | IoT device registry table |
| 49 | `create_audit_logs` | Security audit logs table |
| 50 | `create_ai_usage_logs` | AI usage tracking table |
| 51 | `create_bank_statements` | Bank statements table |
| 52 | `create_bank_transactions` | Bank transactions table |
| 53 | `create_reconciliation_matches` | Reconciliation matches table |
| 54 | `create_currencies` | Currencies table |
| 55 | `create_exchange_rates` | Exchange rates table |
| 56 | `create_inventory_levels` | Inventory levels per warehouse |
| 57 | `create_stock_movements` | Stock movements table |
| 58 | `create_shift_templates` | Shift templates table |
| 59 | `create_shift_assignments` | Shift assignments table |
| 60 | `create_employee_tax_profiles` | Employee tax profiles table |
| 61 | `create_payroll_disbursements` | Payroll disbursements table |
| 62 | `create_iot_device_readings` | IoT device readings table |
| 63 | `create_bi_dashboards` | BI dashboards table |
| 64 | `create_indexes_category5` | Performance indexes (12 indexes) |
| 65 | `seed_default_currencies` | Seeds 10 default currencies |
| 66 | `seed_default_plans` | Seeds 3 subscription plans |
| 67 | `seed_default_permissions` | Seeds default permissions |
| 69 | `super_admin_system` | Super-admin maintenance rules, system errors, CI health tables |
| 70 | `seed_super_admin_permission` | Seeds the `super_admin.access` permission |
| 71 | `super_admin_user_org_columns` | Adds `is_banned` / `ban_reason` to users, org status columns |
| 72 | `add_api_key_prefix` | Adds `key_prefix` column + partial index to `api_keys` |
| 73 | `create_schema_migrations` | Creates the migration version-tracking table |
| 74 | `super_admin_soc_events` | SOC security events table + indexes |
| 75 | `add_users_last_login_at` | Adds `last_login_at` to users |

> **Note:** There is no migration `000068`; the sequence jumps from `000067` to `000069`. In total there are **74 `.up.sql` files** (and 74 matching `.down.sql` files).
## Limitations & roadmap

The current migration system is simple and runs on startup. Known limitations:

- **Version tracking is present** via the `schema_migrations` table, so migrations run once each
- **No automatic rollback** — `.down.sql` files exist for reference, but the runner does not execute them
- **No CLI migration tool** — migrations run only at server startup

For more advanced workflows (explicit up/down commands, environment-specific migrations), a dedicated runner (e.g., `golang-migrate`, `atlas`, or `goose`) could be adopted. The embedded, numbered file approach makes this migration straightforward.