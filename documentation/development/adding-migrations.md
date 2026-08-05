# Adding Migrations

## Overview

Database migrations are **embedded SQL files** loaded via Go's `embed.FS` package. Each migration is a numbered `.up.sql` file in `internal/database/migrations/`, executed in lexicographic order at server startup.

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

The `Migrate()` method in `internal/database/database.go` calls `loadMigrations()` and executes each SQL file:

```go
func (db *DB) Migrate(ctx context.Context) error {
    migrations, err := loadMigrations()
    if err != nil {
        return fmt.Errorf("load migrations: %w", err)
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

Files follow the pattern `NNNNNN_<descriptive_name>.up.sql`:

| Prefix | Purpose |
|---|---|
| `000001`–`000005` | Custom enum types |
| `000006`–`000067` | Tables, indexes, and seed data |
| `000068+` | Future additions |

The six-digit zero-padded prefix ensures lexicographic sort order matches execution order. The descriptive name after the first underscore is used as the console log label (e.g., `✓ create_leads`).

## Idempotency requirements

All migrations **must be idempotent** because they run on every startup:

| Operation | Safe pattern |
|---|---|
| Create table | `CREATE TABLE IF NOT EXISTS` |
| Create enum type | `DO $$ BEGIN CREATE TYPE ... EXCEPTION WHEN duplicate_object THEN NULL; END $$` |
| Insert seed data | `INSERT ... ON CONFLICT (key) DO NOTHING` |
| Add column | `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` |
| Add index | `CREATE INDEX IF NOT EXISTS` |

## Current migration list (67 files)

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
| 67 | `seed_default_permissions` | Seeds 47 default permissions |

## Limitations & roadmap

The current migration system is simple and runs on every startup. Known limitations:

- **No version tracking** — migrations run unconditionally; they rely on idempotency
- **No rollback** — schema changes cannot be reverted
- **No down migrations** — only `.up.sql` files are supported

For production use, a versioned migration runner (e.g., `golang-migrate`, `atlas`, or `goose`) is recommended. The embedded file approach makes it easy to migrate to such a tool later — the SQL files are already structured and numbered.