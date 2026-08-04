# Database Schema

## Overview

The database is PostgreSQL 16. Tables are created via idempotent migrations in `internal/database/database.go`. The schema uses UUID primary keys and `TIMESTAMPTZ` for timestamps.

## Entity-Relationship Diagram

```mermaid
erDiagram
    organizations ||--o{ organization_members : "has"
    organizations ||--o{ organization_invitations : "sends"
    organizations ||--o{ api_keys : "owns"
    organizations ||--o{ subscriptions : "has"
    organizations ||--o{ roles : "defines"
    users ||--o{ organization_members : "belongs to"
    roles ||--o{ role_permissions : "grants"
    permissions ||--o{ role_permissions : "included in"
    organizations ||--o{ crm_contacts : "owns"
    organizations ||--o{ crm_deals : "owns"
    organizations ||--o{ crm_quotes : "owns"
    organizations ||--o{ crm_helpdesk_tickets : "owns"
    organizations ||--o{ crm_campaigns : "owns"
    crm_contacts ||--o{ crm_deals : "linked to"
    crm_deals ||--o{ crm_quotes : "has"
    crm_contacts ||--o{ crm_helpdesk_tickets : "raises"
    users ||--o{ crm_contacts : "assigned"
    users ||--o{ crm_deals : "assigned"
    users ||--o{ crm_helpdesk_tickets : "assigned"
    organizations ||--o{ warehouses : "owns"
    organizations ||--o{ products : "owns"
    organizations ||--o{ fleet_vehicles : "owns"
    organizations ||--o{ fleet_drivers : "owns"
    organizations ||--o{ shipments : "owns"
    organizations ||--o{ fleet_telematics_logs : "owns"
    organizations ||--o{ purchase_orders : "owns"
    products ||--o{ bill_of_materials : "has"
    bill_of_materials ||--o{ bom_components : "contains"
    products ||--o{ bom_components : "used in"
    fleet_vehicles ||--o{ fleet_telematics_logs : "generates"
    fleet_vehicles ||--o{ shipments : "assigned to"
    fleet_drivers ||--o{ shipments : "assigned to"
    users ||--o{ fleet_drivers : "linked to"
    organizations ||--o{ employees : "employs"
    users ||--o{ employees : "linked to"
    employees ||--o{ attendance_logs : "clocks in"
    organizations ||--o{ attendance_logs : "owns"
    organizations ||--o{ payroll_runs : "runs"
    organizations ||--o{ job_applications : "receives"
    organizations ||--o{ knowledge_base_documents : "maintains"
    organizations ||--o{ ai_copilot_conversations : "uses"
    organizations ||--o{ lowcode_workflows : "defines"
    organizations ||--o{ audit_logs : "generates"
    organizations ||--o{ ai_usage_logs : "tracks"
    organizations ||--o{ iot_devices : "owns"
    users ||--o{ ai_copilot_conversations : "prompts"
    users ||--o{ audit_logs : "performs"
    users ||--o{ ai_usage_logs : "consumes"
```

## Table definitions

### Core tables (organizations, users, roles, permissions, members, invitations, API keys, subscriptions)

See [Architecture overview](overview.md) for core table details.

## Supply Chain & Fleet tables

### `warehouses`

```sql
CREATE TABLE warehouses (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id  UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name    VARCHAR(100) NOT NULL,
    address TEXT
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `name` | VARCHAR(100) | Required |
| `address` | TEXT | Nullable |

### `products`

```sql
CREATE TABLE products (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sku              VARCHAR(100) NOT NULL,
    name             VARCHAR(255) NOT NULL,
    unit_price       DECIMAL(10, 2) NOT NULL,
    cost_price       DECIMAL(10, 2) NOT NULL,
    ai_reorder_point INT DEFAULT 15,
    UNIQUE(org_id, sku)
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `sku` | VARCHAR(100) | Unique per organization, stock keeping unit |
| `name` | VARCHAR(255) | Product name |
| `unit_price` | DECIMAL(10,2) | Selling price |
| `cost_price` | DECIMAL(10,2) | Cost price |
| `ai_reorder_point` | INT | AI-generated reorder threshold, default 15 |

### `bill_of_materials`

```sql
CREATE TABLE bill_of_materials (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id            UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    parent_product_id UUID NOT NULL REFERENCES products(id),
    bom_code          VARCHAR(50) NOT NULL
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `parent_product_id` | UUID | FK → products, the assembled product |
| `bom_code` | VARCHAR(50) | BOM identifier code |

### `bom_components`

```sql
CREATE TABLE bom_components (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    bom_id               UUID NOT NULL REFERENCES bill_of_materials(id) ON DELETE CASCADE,
    component_product_id UUID NOT NULL REFERENCES products(id),
    quantity_required    DECIMAL(10, 4) NOT NULL
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `bom_id` | UUID | FK → bill_of_materials |
| `component_product_id` | UUID | FK → products, the component |
| `quantity_required` | DECIMAL(10,4) | Quantity of component needed |

### `fleet_vehicles`

```sql
CREATE TABLE fleet_vehicles (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    vin           VARCHAR(17) UNIQUE NOT NULL,
    license_plate VARCHAR(20) NOT NULL,
    make          VARCHAR(50) NOT NULL,
    model         VARCHAR(50) NOT NULL,
    status        vehicle_status DEFAULT 'active',
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `vin` | VARCHAR(17) | Unique, 17-char vehicle identification number |
| `license_plate` | VARCHAR(20) | License plate number |
| `make` | VARCHAR(50) | Vehicle manufacturer |
| `model` | VARCHAR(50) | Vehicle model |
| `status` | vehicle_status | Enum: `active`, `maintenance`, `decommissioned` |
| `created_at` | TIMESTAMPTZ | Auto-set |

### `fleet_drivers`

```sql
CREATE TABLE fleet_drivers (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id        UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    license_number VARCHAR(100) NOT NULL,
    safety_rating  DECIMAL(3, 2) DEFAULT 5.00
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `user_id` | UUID | FK → users, unique (one driver per user) |
| `license_number` | VARCHAR(100) | Driver's license number |
| `safety_rating` | DECIMAL(3,2) | Safety rating, default 5.00 |

### `shipments`

```sql
CREATE TABLE shipments (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    tracking_number     VARCHAR(100) UNIQUE NOT NULL,
    origin_address      TEXT NOT NULL,
    destination_address TEXT NOT NULL,
    status              shipment_status DEFAULT 'pending',
    assigned_vehicle_id UUID REFERENCES fleet_vehicles(id),
    assigned_driver_id  UUID REFERENCES fleet_drivers(id),
    created_at          TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `tracking_number` | VARCHAR(100) | Unique |
| `origin_address` | TEXT | Shipment origin |
| `destination_address` | TEXT | Shipment destination |
| `status` | shipment_status | Enum: `pending`, `in_transit`, `delivered`, `cancelled` |
| `assigned_vehicle_id` | UUID | FK → fleet_vehicles, nullable |
| `assigned_driver_id` | UUID | FK → fleet_drivers, nullable |
| `created_at` | TIMESTAMPTZ | Auto-set |

### `fleet_telematics_logs`

```sql
CREATE TABLE fleet_telematics_logs (
    id             BIGSERIAL PRIMARY KEY,
    org_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    vehicle_id     UUID NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
    latitude       DECIMAL(10, 8) NOT NULL,
    longitude      DECIMAL(11, 8) NOT NULL,
    speed_kmh      DECIMAL(5, 2) NOT NULL,
    fuel_level_pct DECIMAL(5, 2),
    recorded_at    TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | BIGSERIAL | Auto-incrementing, suitable for high-volume ingestion |
| `org_id` | UUID | FK → organizations |
| `vehicle_id` | UUID | FK → fleet_vehicles, cascade delete |
| `latitude` | DECIMAL(10,8) | GPS latitude |
| `longitude` | DECIMAL(11,8) | GPS longitude |
| `speed_kmh` | DECIMAL(5,2) | Speed in km/h |
| `fuel_level_pct` | DECIMAL(5,2) | Fuel level percentage, nullable |
| `recorded_at` | TIMESTAMPTZ | Time of recording, default NOW() |

### `purchase_orders`

```sql
CREATE TABLE purchase_orders (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                 UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    po_number              VARCHAR(100) NOT NULL,
    supplier_name          VARCHAR(255) NOT NULL,
    total_cost             DECIMAL(12, 2) NOT NULL,
    ai_supplier_risk_rating VARCHAR(50) DEFAULT 'Low Risk',
    status                 VARCHAR(50) DEFAULT 'draft',
    created_at             TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `po_number` | VARCHAR(100) | PO identifier |
| `supplier_name` | VARCHAR(255) | Supplier name |
| `total_cost` | DECIMAL(12,2) | Total order cost |
| `ai_supplier_risk_rating` | VARCHAR(50) | AI risk rating, default 'Low Risk' |
| `status` | VARCHAR(50) | Default 'draft' |
| `created_at` | TIMESTAMPTZ | Auto-set |

## HR & Workforce tables

### `employees`

```sql
CREATE TABLE employees (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id       UUID UNIQUE REFERENCES users(id),
    employee_code VARCHAR(50) NOT NULL,
    department    VARCHAR(100),
    job_title     VARCHAR(100),
    salary        DECIMAL(12, 2),
    hired_at      DATE NOT NULL,
    UNIQUE(org_id, employee_code)
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `user_id` | UUID | FK → users, unique (one employee record per user) |
| `employee_code` | VARCHAR(50) | Unique per organization |
| `department` | VARCHAR(100) | Nullable |
| `job_title` | VARCHAR(100) | Nullable |
| `salary` | DECIMAL(12,2) | Nullable |
| `hired_at` | DATE | Required |

### `attendance_logs`

```sql
CREATE TABLE attendance_logs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    employee_id    UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
    clock_in       TIMESTAMPTZ NOT NULL,
    clock_out      TIMESTAMPTZ,
    location_lat   DECIMAL(10, 8),
    location_long  DECIMAL(11, 8)
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `employee_id` | UUID | FK → employees, cascade delete |
| `clock_in` | TIMESTAMPTZ | Required, time of clock-in |
| `clock_out` | TIMESTAMPTZ | Nullable, time of clock-out |
| `location_lat` | DECIMAL(10,8) | GPS latitude at clock-in |
| `location_long` | DECIMAL(11,8) | GPS longitude at clock-in |

### `payroll_runs`

```sql
CREATE TABLE payroll_runs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    pay_period_start DATE NOT NULL,
    pay_period_end   DATE NOT NULL,
    total_disbursed  DECIMAL(14, 2) NOT NULL,
    status           VARCHAR(50) DEFAULT 'draft',
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `pay_period_start` | DATE | Start of the pay period |
| `pay_period_end` | DATE | End of the pay period |
| `total_disbursed` | DECIMAL(14,2) | Total amount disbursed in this run |
| `status` | VARCHAR(50) | Default 'draft' |
| `created_at` | TIMESTAMPTZ | Auto-set |

### `job_applications`

```sql
CREATE TABLE job_applications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    candidate_name  VARCHAR(150) NOT NULL,
    email           VARCHAR(255) NOT NULL,
    resume_url      TEXT NOT NULL,
    ai_match_score  INT,
    status          VARCHAR(50) DEFAULT 'applied',
    created_at      TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `candidate_name` | VARCHAR(150) | Required |
| `email` | VARCHAR(255) | Required |
| `resume_url` | TEXT | URL/path to the uploaded resume file |
| `ai_match_score` | INT | AI match score (0–100), nullable |
| `status` | VARCHAR(50) | Default 'applied' |
| `created_at` | TIMESTAMPTZ | Auto-set |

### `knowledge_base_documents`

```sql
CREATE TABLE knowledge_base_documents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id              UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    title               VARCHAR(255) NOT NULL,
    content             TEXT NOT NULL,
    vector_embedding_id VARCHAR(255),
    created_at          TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `title` | VARCHAR(255) | Document title |
| `content` | TEXT | Full document content for search |
| `vector_embedding_id` | VARCHAR(255) | Reference to external vector store embedding, nullable |
| `created_at` | TIMESTAMPTZ | Auto-set |

## Custom Enums

### `vehicle_status`

```sql
CREATE TYPE vehicle_status AS ENUM ('active', 'maintenance', 'decommissioned');
```

### `shipment_status`

```sql
CREATE TYPE shipment_status AS ENUM ('pending', 'in_transit', 'delivered', 'cancelled');
```

## Seed data

The following permissions are inserted on every migration (idempotent via `ON CONFLICT DO NOTHING`):

| Permission Key | Module | Description |
|---|---|---|
| `users.invite` | users | Invite team members to the organization |
| `users.manage` | users | Manage organization members (update role, deactivate, remove) |
| `rbac.manage` | rbac | Create and manage roles and permissions |
| `apikeys.manage` | apikeys | Generate and revoke API keys |
| `billing.manage` | billing | Manage subscriptions and billing |
| `crm.leads.write` | crm | Create and manage CRM leads |
| `crm.quotes.write` | crm | Manage and analyze CRM quotes |
| `crm.tickets.write` | crm | Create and manage CRM support tickets |
| `accounting.ledger.write` | accounting | Post general ledger journal entries |
| `accounting.invoices.write` | accounting | Upload and process invoices |
| `expenses.submit` | accounting | Submit expense reports |
| `fleet.telematics.ingest` | fleet | Ingest vehicle telemetry data via API key |
| `fleet.routes.manage` | fleet | Generate and manage optimized fleet routes |
| `inventory.read` | inventory | Read inventory reorder predictions and stock levels |
| `hr.attendance.write` | hr | Clock in/out and manage attendance records |
| `hr.recruitment.write` | hr | Parse resumes and manage job applications |
| `knowledge.read` | knowledge | Search and read knowledge base documents |
| `copilot.use` | platform | Use the AI text-to-SQL copilot query feature |
| `workflows.execute` | platform | Trigger and execute low-code automated workflows |
| `security.audit.read` | platform | Read security audit anomaly logs |

## Platform, AI Core & BI tables

### `ai_copilot_conversations`

Stores natural language prompts and their AI-generated SQL responses.

```sql
CREATE TABLE ai_copilot_conversations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id          UUID REFERENCES users(id),
    prompt_text      TEXT NOT NULL,
    generated_sql    TEXT,
    response_payload JSONB,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `user_id` | UUID | FK → users, nullable |
| `prompt_text` | TEXT | The natural language prompt |
| `generated_sql` | TEXT | The AI-generated SQL, nullable |
| `response_payload` | JSONB | Full response payload for audit, nullable |
| `created_at` | TIMESTAMPTZ | Auto-set |

### `lowcode_workflows`

Defines automated workflows with trigger events and action steps.

```sql
CREATE TABLE lowcode_workflows (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id        UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name          VARCHAR(100) NOT NULL,
    trigger_event VARCHAR(100) NOT NULL,
    action_steps  JSONB NOT NULL,
    is_active     BOOLEAN DEFAULT TRUE,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `name` | VARCHAR(100) | Human-readable workflow name |
| `trigger_event` | VARCHAR(100) | Event type that triggers this workflow (e.g., `invoice.paid`) |
| `action_steps` | JSONB | Array of action step definitions |
| `is_active` | BOOLEAN | Whether the workflow is enabled |
| `created_at` | TIMESTAMPTZ | Auto-set |

### `iot_devices`

Registry for IoT devices connected to the platform.

```sql
CREATE TABLE iot_devices (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    device_name VARCHAR(100) NOT NULL,
    device_type VARCHAR(50) NOT NULL,
    mac_address VARCHAR(100) UNIQUE,
    status      VARCHAR(50) DEFAULT 'online',
    last_ping   TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `device_name` | VARCHAR(100) | Human-readable device name |
| `device_type` | VARCHAR(50) | Device category |
| `mac_address` | VARCHAR(100) | Unique MAC address, nullable |
| `status` | VARCHAR(50) | Device status (`online`, `offline`, etc.) |
| `last_ping` | TIMESTAMPTZ | Last heartbeat timestamp |

### `audit_logs`

Stores auditable actions with AI anomaly detection flags.

```sql
CREATE TABLE audit_logs (
    id               BIGSERIAL PRIMARY KEY,
    org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id          UUID REFERENCES users(id),
    action           VARCHAR(100) NOT NULL,
    ip_address       VARCHAR(45),
    ai_anomaly_flag  BOOLEAN DEFAULT FALSE,
    metadata         JSONB,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | BIGSERIAL | Auto-incrementing primary key |
| `org_id` | UUID | FK → organizations |
| `user_id` | UUID | FK → users, nullable |
| `action` | VARCHAR(100) | Action description (e.g., `user.login`, `permission.change`) |
| `ip_address` | VARCHAR(45) | IPv4 or IPv6 address, nullable |
| `ai_anomaly_flag` | BOOLEAN | Whether AI flagged this as anomalous |
| `metadata` | JSONB | Additional context data, nullable |
| `created_at` | TIMESTAMPTZ | Auto-set |

### `ai_usage_logs`

Tracks AI feature usage and credit consumption.

```sql
CREATE TABLE ai_usage_logs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id           UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id          UUID REFERENCES users(id),
    feature_used     VARCHAR(100) NOT NULL,
    credits_consumed INT NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);
```

| Column | Type | Notes |
|---|---|---|
| `id` | UUID | Primary key, auto-generated |
| `org_id` | UUID | FK → organizations |
| `user_id` | UUID | FK → users, nullable |
| `feature_used` | VARCHAR(100) | Feature identifier (e.g., `copilot.query`, `workflows.execute`) |
| `credits_consumed` | INT | Number of AI credits consumed |
| `created_at` | TIMESTAMPTZ | Auto-set |

## Performance indexes

```sql
CREATE INDEX idx_telematics_vehicle_time ON fleet_telematics_logs(vehicle_id, recorded_at DESC);
CREATE INDEX idx_org_members_lookup ON organization_members(user_id, org_id);
CREATE INDEX idx_orgs_domain_slug ON organizations(domain_slug);
CREATE INDEX idx_audit_org_time ON audit_logs(org_id, created_at DESC);
CREATE INDEX idx_invoices_org_status ON invoices(org_id, status);
CREATE INDEX idx_contacts_org ON crm_contacts(org_id);
```

## Key constraints

- `organizations.domain_slug` is **UNIQUE**
- `organizations.custom_domain` is **UNIQUE** (nullable)
- `users.email` is **UNIQUE**
- `roles` has a **UNIQUE** constraint on `(org_id, name)`
- `permissions.permission_key` is **UNIQUE**
- `organization_members` has a **UNIQUE** constraint on `(org_id, user_id)`
- `api_keys.key_hash` is **UNIQUE**
- `organization_invitations.token` is **UNIQUE**
- `products` has a **UNIQUE** constraint on `(org_id, sku)`
- `fleet_vehicles.vin` is **UNIQUE**
- `fleet_drivers.user_id` is **UNIQUE**
- `shipments.tracking_number` is **UNIQUE**
- `employees` has a **UNIQUE** constraint on `(org_id, employee_code)`
- `employees.user_id` is **UNIQUE** (one employee record per user)
- Foreign keys use `ON DELETE CASCADE` for clean teardown, except `organization_members.role_id` which prevents accidental role deletion while members are assigned
