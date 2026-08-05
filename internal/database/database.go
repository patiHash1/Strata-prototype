package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DB wraps a pgx connection pool.
type DB struct {
	Pool *pgxpool.Pool
}

// New creates a new connection pool from the provided DSN.
func New(ctx context.Context, dsn string) (*DB, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse pool config: %w", err)
	}

	cfg.MaxConns = 25
	cfg.MinConns = 5
	cfg.MaxConnLifetime = 30 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{Pool: pool}, nil
}

// Ping checks database connectivity.
func (db *DB) Ping(ctx context.Context) error {
	return db.Pool.Ping(ctx)
}

// Close shuts down the connection pool.
func (db *DB) Close() {
	db.Pool.Close()
}

// Migrate runs the schema migration — creates all tables if they don't exist.
func (db *DB) Migrate(ctx context.Context) error {
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "create_org_status_enum",
			sql: `
				DO $$ BEGIN
					CREATE TYPE org_status AS ENUM ('active', 'suspended', 'pending_verification');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$`,
		},
		{
			name: "create_subscription_status_enum",
			sql: `
				DO $$ BEGIN
					CREATE TYPE subscription_status AS ENUM ('active', 'past_due', 'canceled', 'trialing');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$`,
		},
		{
			name: "create_ticket_priority_enum",
			sql: `
				DO $$ BEGIN
					CREATE TYPE ticket_priority AS ENUM ('low', 'medium', 'high', 'urgent');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$`,
		},
		{
			name: "create_vehicle_status_enum",
			sql: `
				DO $$ BEGIN
					CREATE TYPE vehicle_status AS ENUM ('active', 'in_transit', 'maintenance', 'decommissioned');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$`,
		},
		{
			name: "create_shipment_status_enum",
			sql: `
				DO $$ BEGIN
					CREATE TYPE shipment_status AS ENUM ('pending', 'assigned', 'in_transit', 'delivered', 'delayed');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$`,
		},
		{
			name: "create_organizations",
			sql: `
				CREATE TABLE IF NOT EXISTS organizations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					domain_slug VARCHAR(100) UNIQUE NOT NULL,
					custom_domain VARCHAR(255) UNIQUE,
					company_name VARCHAR(255) NOT NULL,
					default_currency VARCHAR(3) DEFAULT 'USD',
					timezone VARCHAR(50) DEFAULT 'UTC',
					status org_status DEFAULT 'active',
					created_at TIMESTAMPTZ DEFAULT NOW(),
					updated_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_users",
			sql: `
				CREATE TABLE IF NOT EXISTS users (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email VARCHAR(255) UNIQUE NOT NULL,
					password_hash VARCHAR(255) NOT NULL,
					full_name VARCHAR(150) NOT NULL,
					phone_number VARCHAR(50),
					mfa_enabled BOOLEAN DEFAULT FALSE,
					mfa_secret VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_roles",
			sql: `
				CREATE TABLE IF NOT EXISTS roles (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					description VARCHAR(255),
					is_system_default BOOLEAN DEFAULT FALSE,
					UNIQUE(org_id, name)
				)`,
		},
		{
			name: "create_permissions",
			sql: `
				CREATE TABLE IF NOT EXISTS permissions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					permission_key VARCHAR(100) UNIQUE NOT NULL,
					module VARCHAR(50) NOT NULL,
					description VARCHAR(255)
				)`,
		},
		{
			name: "create_role_permissions",
			sql: `
				CREATE TABLE IF NOT EXISTS role_permissions (
					role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
					permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
					PRIMARY KEY (role_id, permission_id)
				)`,
		},
		{
			name: "create_organization_members",
			sql: `
				CREATE TABLE IF NOT EXISTS organization_members (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					role_id UUID NOT NULL REFERENCES roles(id),
					is_active BOOLEAN DEFAULT TRUE,
					joined_at TIMESTAMPTZ DEFAULT NOW(),
					UNIQUE(org_id, user_id)
				)`,
		},
		{
			name: "create_organization_invitations",
			sql: `
				CREATE TABLE IF NOT EXISTS organization_invitations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					email VARCHAR(255) NOT NULL,
					role_id UUID NOT NULL REFERENCES roles(id),
					token VARCHAR(255) UNIQUE NOT NULL,
					expires_at TIMESTAMPTZ NOT NULL,
					accepted_at TIMESTAMPTZ,
					created_by UUID REFERENCES users(id)
				)`,
		},
		{
			name: "create_api_keys",
			sql: `
				CREATE TABLE IF NOT EXISTS api_keys (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					key_hash VARCHAR(255) UNIQUE NOT NULL,
					scopes TEXT[],
					last_used_at TIMESTAMPTZ,
					expires_at TIMESTAMPTZ,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_subscriptions",
			sql: `
				CREATE TABLE IF NOT EXISTS subscriptions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID UNIQUE NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					plan_id UUID NOT NULL REFERENCES subscription_plans(id),
					stripe_customer_id VARCHAR(255),
					stripe_subscription_id VARCHAR(255),
					status subscription_status DEFAULT 'trialing',
					current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					current_period_end TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_crm_contacts",
			sql: `
				CREATE TABLE IF NOT EXISTS crm_contacts (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					first_name VARCHAR(100) NOT NULL,
					last_name VARCHAR(100),
					email VARCHAR(255),
					phone VARCHAR(50),
					company_name VARCHAR(255),
					assigned_to UUID REFERENCES users(id),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_crm_deals",
			sql: `
				CREATE TABLE IF NOT EXISTS crm_deals (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					contact_id UUID REFERENCES crm_contacts(id) ON DELETE CASCADE,
					title VARCHAR(255) NOT NULL,
					amount DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					stage VARCHAR(50) NOT NULL,
					ai_win_probability INT,
					assigned_to UUID REFERENCES users(id),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_crm_quotes",
			sql: `
				CREATE TABLE IF NOT EXISTS crm_quotes (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					deal_id UUID REFERENCES crm_deals(id) ON DELETE CASCADE,
					quote_number VARCHAR(100) NOT NULL,
					total_amount DECIMAL(12, 2) NOT NULL,
					ai_risk_score DECIMAL(5,2),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_crm_helpdesk_tickets",
			sql: `
				CREATE TABLE IF NOT EXISTS crm_helpdesk_tickets (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					contact_id UUID REFERENCES crm_contacts(id),
					subject VARCHAR(255) NOT NULL,
					description TEXT NOT NULL,
					priority ticket_priority DEFAULT 'medium',
					status VARCHAR(50) DEFAULT 'open',
					ai_sentiment_score DECIMAL(3, 2),
					ai_suggested_response TEXT,
					assigned_to UUID REFERENCES users(id),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_crm_campaigns",
			sql: `
				CREATE TABLE IF NOT EXISTS crm_campaigns (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(255) NOT NULL,
					channel VARCHAR(50) NOT NULL,
					ai_target_segment_criteria JSONB,
					budget DECIMAL(12, 2),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "add_campaign_status",
			sql:  "ALTER TABLE crm_campaigns ADD COLUMN IF NOT EXISTS status VARCHAR(50) DEFAULT 'draft'",
		},
		{
			name: "create_field_sales_visits",
			sql: `
				CREATE TABLE IF NOT EXISTS field_sales_visits (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					contact_id UUID REFERENCES crm_contacts(id),
					sales_rep_id UUID REFERENCES users(id),
					scheduled_at TIMESTAMPTZ NOT NULL,
					location_lat DECIMAL(10,8),
					location_long DECIMAL(11,8),
					status VARCHAR(50) DEFAULT 'scheduled',
					notes TEXT,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_subscription_plans",
			sql: `
				CREATE TABLE IF NOT EXISTS subscription_plans (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					plan_code VARCHAR(50) UNIQUE NOT NULL,
					name VARCHAR(100) NOT NULL,
					max_tenants_limit INT NOT NULL,
					max_vehicles_limit INT NOT NULL,
					max_ai_credits_per_month INT NOT NULL,
					monthly_price DECIMAL(10, 2) NOT NULL
				)`,
		},
		{
			name: "create_plan_features",
			sql: `
				CREATE TABLE IF NOT EXISTS plan_features (
					plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
					feature_key VARCHAR(100) NOT NULL,
					PRIMARY KEY (plan_id, feature_key)
				)`,
		},
		{
			name: "create_chart_of_accounts",
			sql: `
				CREATE TABLE IF NOT EXISTS chart_of_accounts (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					account_code VARCHAR(20) NOT NULL,
					account_name VARCHAR(100) NOT NULL,
					account_type VARCHAR(50) NOT NULL,
					UNIQUE(org_id, account_code)
				)`,
		},
		{
			name: "create_journal_entries",
			sql: `
				CREATE TABLE IF NOT EXISTS journal_entries (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					entry_number VARCHAR(100) NOT NULL,
					entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
					memo VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_journal_items",
			sql: `
				CREATE TABLE IF NOT EXISTS journal_items (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
					account_id UUID NOT NULL REFERENCES chart_of_accounts(id),
					debit DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					credit DECIMAL(12, 2) NOT NULL DEFAULT 0.00
				)`,
		},
		{
			name: "create_invoices",
			sql: `
				CREATE TABLE IF NOT EXISTS invoices (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					invoice_number VARCHAR(100) NOT NULL,
					contact_id UUID REFERENCES crm_contacts(id),
					total_amount DECIMAL(12, 2) NOT NULL,
					status VARCHAR(50) DEFAULT 'draft',
					ai_ocr_processed BOOLEAN DEFAULT FALSE,
					due_date DATE NOT NULL,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_expenses",
			sql: `
				CREATE TABLE IF NOT EXISTS expenses (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					amount DECIMAL(10, 2) NOT NULL,
					category VARCHAR(100) NOT NULL,
					receipt_url TEXT,
					ai_fraud_flag BOOLEAN DEFAULT FALSE,
					ai_audit_notes TEXT,
					status VARCHAR(50) DEFAULT 'pending',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_fixed_assets",
			sql: `
				CREATE TABLE IF NOT EXISTS fixed_assets (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					asset_name VARCHAR(255) NOT NULL,
					purchase_date DATE NOT NULL,
					purchase_cost DECIMAL(12, 2) NOT NULL,
					salvage_value DECIMAL(12, 2) DEFAULT 0.00,
					useful_life_years INT NOT NULL,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_tax_rates",
			sql: `
					CREATE TABLE IF NOT EXISTS tax_rates (
						id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
						org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
						country_code VARCHAR(2) NOT NULL,
						tax_name VARCHAR(50) NOT NULL,
						tax_rate DECIMAL(5,4) NOT NULL,
						is_active BOOLEAN DEFAULT TRUE,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						UNIQUE(org_id, country_code, tax_name)
					)`,
		},
		{
			name: "create_warehouses",
			sql: `
				CREATE TABLE IF NOT EXISTS warehouses (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					address TEXT
				)`,
		},
		{
			name: "create_products",
			sql: `
				CREATE TABLE IF NOT EXISTS products (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					sku VARCHAR(100) NOT NULL,
					name VARCHAR(255) NOT NULL,
					unit_price DECIMAL(10, 2) NOT NULL,
					cost_price DECIMAL(10, 2) NOT NULL,
					ai_reorder_point INT DEFAULT 15,
					UNIQUE(org_id, sku)
				)`,
		},
		{
			name: "create_bill_of_materials",
			sql: `
				CREATE TABLE IF NOT EXISTS bill_of_materials (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					parent_product_id UUID NOT NULL REFERENCES products(id),
					bom_code VARCHAR(50) NOT NULL
				)`,
		},
		{
			name: "create_bom_components",
			sql: `
				CREATE TABLE IF NOT EXISTS bom_components (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					bom_id UUID NOT NULL REFERENCES bill_of_materials(id) ON DELETE CASCADE,
					component_product_id UUID NOT NULL REFERENCES products(id),
					quantity_required DECIMAL(10, 4) NOT NULL
				)`,
		},
		{
			name: "create_fleet_vehicles",
			sql: `
				CREATE TABLE IF NOT EXISTS fleet_vehicles (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					vin VARCHAR(17) UNIQUE NOT NULL,
					license_plate VARCHAR(20) NOT NULL,
					make VARCHAR(50) NOT NULL,
					model VARCHAR(50) NOT NULL,
					status vehicle_status DEFAULT 'active',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_fleet_drivers",
			sql: `
				CREATE TABLE IF NOT EXISTS fleet_drivers (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
					license_number VARCHAR(100) NOT NULL,
					safety_rating DECIMAL(3, 2) DEFAULT 5.00
				)`,
		},
		{
			name: "create_shipments",
			sql: `
				CREATE TABLE IF NOT EXISTS shipments (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					tracking_number VARCHAR(100) UNIQUE NOT NULL,
					origin_address TEXT NOT NULL,
					destination_address TEXT NOT NULL,
					status shipment_status DEFAULT 'pending',
					assigned_vehicle_id UUID REFERENCES fleet_vehicles(id),
					assigned_driver_id UUID REFERENCES fleet_drivers(id),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_fleet_telematics_logs",
			sql: `
				CREATE TABLE IF NOT EXISTS fleet_telematics_logs (
					id BIGSERIAL PRIMARY KEY,
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					vehicle_id UUID NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
					latitude DECIMAL(10, 8) NOT NULL,
					longitude DECIMAL(11, 8) NOT NULL,
					speed_kmh DECIMAL(5, 2) NOT NULL,
					fuel_level_pct DECIMAL(5, 2),
					recorded_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_purchase_orders",
			sql: `
				CREATE TABLE IF NOT EXISTS purchase_orders (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					po_number VARCHAR(100) NOT NULL,
					supplier_name VARCHAR(255) NOT NULL,
					total_cost DECIMAL(12, 2) NOT NULL,
					ai_supplier_risk_rating VARCHAR(50) DEFAULT 'Low Risk',
					status VARCHAR(50) DEFAULT 'draft',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_work_orders",
			sql: `
					CREATE TABLE IF NOT EXISTS work_orders (
						id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
						org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
						bom_id UUID NOT NULL REFERENCES bill_of_materials(id),
						quantity INT NOT NULL DEFAULT 1,
						status VARCHAR(50) DEFAULT 'planned',
						scheduled_start DATE,
						scheduled_end DATE,
						ai_bottleneck_risk VARCHAR(50) DEFAULT 'Low',
						created_at TIMESTAMPTZ DEFAULT NOW()
					)`,
		},
		{
			name: "create_employees",
			sql: `
				CREATE TABLE IF NOT EXISTS employees (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID UNIQUE REFERENCES users(id),
					employee_code VARCHAR(50) NOT NULL,
					department VARCHAR(100),
					job_title VARCHAR(100),
					salary DECIMAL(12, 2),
					hired_at DATE NOT NULL,
					UNIQUE(org_id, employee_code)
				)`,
		},
		{
			name: "create_attendance_logs",
			sql: `
				CREATE TABLE IF NOT EXISTS attendance_logs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
					clock_in TIMESTAMPTZ NOT NULL,
					clock_out TIMESTAMPTZ,
					location_lat DECIMAL(10, 8),
					location_long DECIMAL(11, 8)
				)`,
		},
		{
			name: "create_payroll_runs",
			sql: `
				CREATE TABLE IF NOT EXISTS payroll_runs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					pay_period_start DATE NOT NULL,
					pay_period_end DATE NOT NULL,
					total_disbursed DECIMAL(14, 2) NOT NULL,
					status VARCHAR(50) DEFAULT 'draft',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_job_applications",
			sql: `
				CREATE TABLE IF NOT EXISTS job_applications (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					candidate_name VARCHAR(150) NOT NULL,
					email VARCHAR(255) NOT NULL,
					resume_url TEXT NOT NULL,
					ai_match_score INT,
					status VARCHAR(50) DEFAULT 'applied',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_knowledge_base_documents",
			sql: `
				CREATE TABLE IF NOT EXISTS knowledge_base_documents (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					title VARCHAR(255) NOT NULL,
					content TEXT NOT NULL,
					vector_embedding_id VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_ai_copilot_conversations",
			sql: `
				CREATE TABLE IF NOT EXISTS ai_copilot_conversations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					prompt_text TEXT NOT NULL,
					generated_sql TEXT,
					response_payload JSONB,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_lowcode_workflows",
			sql: `
				CREATE TABLE IF NOT EXISTS lowcode_workflows (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					trigger_event VARCHAR(100) NOT NULL,
					action_steps JSONB NOT NULL,
					is_active BOOLEAN DEFAULT TRUE,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_iot_devices",
			sql: `
				CREATE TABLE IF NOT EXISTS iot_devices (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					device_name VARCHAR(100) NOT NULL,
					device_type VARCHAR(50) NOT NULL,
					mac_address VARCHAR(100) UNIQUE,
					status VARCHAR(50) DEFAULT 'online',
					last_ping TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_audit_logs",
			sql: `
				CREATE TABLE IF NOT EXISTS audit_logs (
					id BIGSERIAL PRIMARY KEY,
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					action VARCHAR(100) NOT NULL,
					ip_address VARCHAR(45),
					ai_anomaly_flag BOOLEAN DEFAULT FALSE,
					metadata JSONB,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_ai_usage_logs",
			sql: `
				CREATE TABLE IF NOT EXISTS ai_usage_logs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					feature_used VARCHAR(100) NOT NULL,
					credits_consumed INT NOT NULL DEFAULT 1,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		// ── Module 2.1: Bank Reconciliation ──
		{
			name: "create_bank_statements",
			sql: `
				CREATE TABLE IF NOT EXISTS bank_statements (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					bank_name VARCHAR(255) NOT NULL,
					account_number VARCHAR(100) NOT NULL,
					statement_date DATE NOT NULL,
					opening_balance DECIMAL(14, 2) NOT NULL,
					closing_balance DECIMAL(14, 2) NOT NULL,
					status VARCHAR(50) DEFAULT 'imported',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_bank_transactions",
			sql: `
				CREATE TABLE IF NOT EXISTS bank_transactions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					statement_id UUID NOT NULL REFERENCES bank_statements(id) ON DELETE CASCADE,
					transaction_date DATE NOT NULL,
					description VARCHAR(500) NOT NULL,
					reference VARCHAR(100),
					debit DECIMAL(14, 2) DEFAULT 0.00,
					credit DECIMAL(14, 2) DEFAULT 0.00,
					amount DECIMAL(14, 2) NOT NULL,
					is_matched BOOLEAN DEFAULT FALSE,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_reconciliation_matches",
			sql: `
				CREATE TABLE IF NOT EXISTS reconciliation_matches (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					bank_transaction_id UUID NOT NULL REFERENCES bank_transactions(id) ON DELETE CASCADE,
					journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
					match_type VARCHAR(50) DEFAULT 'auto',
					match_date TIMESTAMPTZ DEFAULT NOW(),
					UNIQUE(bank_transaction_id, journal_entry_id)
				)`,
		},
		// ── Module 2.5: Multi-Currency Exchange Rates ──
		{
			name: "create_currencies",
			sql: `
				CREATE TABLE IF NOT EXISTS currencies (
					code VARCHAR(3) PRIMARY KEY,
					name VARCHAR(100) NOT NULL,
					symbol VARCHAR(10) NOT NULL,
					decimal_places INT DEFAULT 2,
					is_active BOOLEAN DEFAULT TRUE
				)`,
		},
		{
			name: "create_exchange_rates",
			sql: `
				CREATE TABLE IF NOT EXISTS exchange_rates (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					from_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
					to_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
					rate DECIMAL(14, 8) NOT NULL,
					effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
					source VARCHAR(100) DEFAULT 'manual',
					created_at TIMESTAMPTZ DEFAULT NOW(),
					UNIQUE(org_id, from_currency, to_currency, effective_date)
				)`,
		},
		// ── Module 3.1: Inventory Levels Per Warehouse ──
		{
			name: "create_inventory_levels",
			sql: `
				CREATE TABLE IF NOT EXISTS inventory_levels (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
					product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
					quantity_on_hand DECIMAL(12, 4) NOT NULL DEFAULT 0,
					quantity_reserved DECIMAL(12, 4) NOT NULL DEFAULT 0,
					quantity_available DECIMAL(12, 4) GENERATED ALWAYS AS (quantity_on_hand - quantity_reserved) STORED,
					last_counted_at TIMESTAMPTZ,
					UNIQUE(warehouse_id, product_id)
				)`,
		},
		{
			name: "create_stock_movements",
			sql: `
				CREATE TABLE IF NOT EXISTS stock_movements (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					movement_type VARCHAR(20) NOT NULL,
					from_warehouse_id UUID REFERENCES warehouses(id),
					to_warehouse_id UUID REFERENCES warehouses(id),
					product_id UUID NOT NULL REFERENCES products(id),
					quantity DECIMAL(12, 4) NOT NULL,
					reference VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		// ── Module 4.2: Shift Management & AI Shift Prediction ──
		{
			name: "create_shift_templates",
			sql: `
				CREATE TABLE IF NOT EXISTS shift_templates (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					start_time TIME NOT NULL,
					end_time TIME NOT NULL,
					day_of_week SMALLINT,
					department VARCHAR(100),
					required_headcount INT DEFAULT 1,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_shift_assignments",
			sql: `
				CREATE TABLE IF NOT EXISTS shift_assignments (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					shift_template_id UUID NOT NULL REFERENCES shift_templates(id),
					employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
					shift_date DATE NOT NULL,
					actual_start TIMESTAMPTZ,
					actual_end TIMESTAMPTZ,
					status VARCHAR(50) DEFAULT 'scheduled',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		// ── Module 4.3: Payroll Tax Withholding Per Employee ──
		{
			name: "create_employee_tax_profiles",
			sql: `
				CREATE TABLE IF NOT EXISTS employee_tax_profiles (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					employee_id UUID UNIQUE NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
					tax_country VARCHAR(2) NOT NULL DEFAULT 'US',
					tax_identification_number VARCHAR(50),
					filing_status VARCHAR(50) DEFAULT 'single',
					withholding_allowances INT DEFAULT 0,
					additional_withholding DECIMAL(10, 2) DEFAULT 0.00,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_payroll_disbursements",
			sql: `
				CREATE TABLE IF NOT EXISTS payroll_disbursements (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					payroll_run_id UUID NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
					employee_id UUID NOT NULL REFERENCES employees(id),
					gross_pay DECIMAL(12, 2) NOT NULL,
					tax_withheld DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					social_security DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					other_deductions DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					net_pay DECIMAL(12, 2) NOT NULL,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		// ── Module 5.4: IoT Device Readings Storage ──
		{
			name: "create_iot_device_readings",
			sql: `
				CREATE TABLE IF NOT EXISTS iot_device_readings (
					id BIGSERIAL PRIMARY KEY,
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					device_id UUID NOT NULL REFERENCES iot_devices(id) ON DELETE CASCADE,
					metric_name VARCHAR(100) NOT NULL,
					metric_value DOUBLE PRECISION NOT NULL,
					unit VARCHAR(50),
					recorded_at TIMESTAMPTZ DEFAULT NOW()
				)`,
		},
		{
			name: "create_bi_dashboards",
			sql: `
					CREATE TABLE IF NOT EXISTS bi_dashboards (
						id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
						org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
						name VARCHAR(255) NOT NULL,
						config JSONB NOT NULL DEFAULT '{}',
						is_active BOOLEAN DEFAULT TRUE,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						updated_at TIMESTAMPTZ DEFAULT NOW()
					)`,
		},
		{
			name: "create_indexes_category5",
			sql: `
				CREATE INDEX IF NOT EXISTS idx_telematics_vehicle_time ON fleet_telematics_logs(vehicle_id, recorded_at DESC);
				CREATE INDEX IF NOT EXISTS idx_org_members_lookup ON organization_members(user_id, org_id);
				CREATE INDEX IF NOT EXISTS idx_orgs_domain_slug ON organizations(domain_slug);
				CREATE INDEX IF NOT EXISTS idx_audit_org_time ON audit_logs(org_id, created_at DESC);
				CREATE INDEX IF NOT EXISTS idx_invoices_org_status ON invoices(org_id, status);
				CREATE INDEX IF NOT EXISTS idx_contacts_org ON crm_contacts(org_id);
				CREATE INDEX IF NOT EXISTS idx_bank_txns_statement ON bank_transactions(statement_id);
				CREATE INDEX IF NOT EXISTS idx_inventory_levels_wh ON inventory_levels(warehouse_id);
				CREATE INDEX IF NOT EXISTS idx_stock_movements_product ON stock_movements(product_id, created_at DESC);
				CREATE INDEX IF NOT EXISTS idx_shift_assignments_emp ON shift_assignments(employee_id, shift_date);
				CREATE INDEX IF NOT EXISTS idx_payroll_disb_run ON payroll_disbursements(payroll_run_id);
				CREATE INDEX IF NOT EXISTS idx_iot_readings_device ON iot_device_readings(device_id, recorded_at DESC);
			`,
		},
		{
			name: "seed_default_currencies",
			sql: `
				INSERT INTO currencies (code, name, symbol, decimal_places) VALUES
					('USD', 'US Dollar', '$', 2),
					('EUR', 'Euro', '€', 2),
					('GBP', 'British Pound', '£', 2),
					('JPY', 'Japanese Yen', '¥', 0),
					('INR', 'Indian Rupee', '₹', 2),
					('CAD', 'Canadian Dollar', 'C$', 2),
					('AUD', 'Australian Dollar', 'A$', 2),
					('CHF', 'Swiss Franc', 'Fr', 2),
					('CNY', 'Chinese Yuan', '¥', 2),
					('BRL', 'Brazilian Real', 'R$', 2)
				ON CONFLICT (code) DO NOTHING`,
		},
		{
			name: "seed_default_plans",
			sql: `
				INSERT INTO subscription_plans (id, plan_code, name, max_tenants_limit, max_vehicles_limit, max_ai_credits_per_month, monthly_price)
				VALUES
					(gen_random_uuid(), 'starter', 'Starter', 1, 5, 1000, 49.00),
					(gen_random_uuid(), 'professional', 'Professional', 3, 25, 5000, 149.00),
					(gen_random_uuid(), 'enterprise', 'Enterprise', 10, 100, 25000, 499.00)
				ON CONFLICT (plan_code) DO NOTHING`,
		},
		{
			name: "seed_default_permissions",
			sql: `
				INSERT INTO permissions (permission_key, module, description)
				VALUES
					('users.invite',   'users', 'Invite team members to the organization'),
					('rbac.manage',    'rbac',  'Create and manage roles and permissions'),
					('apikeys.manage', 'apikeys', 'Generate and revoke API keys'),
					('billing.manage', 'billing', 'Manage subscriptions and billing'),
					('users.manage',  'users',  'Manage organization members (update role, deactivate)'),
					('crm.leads.write',  'crm',  'Create and manage CRM leads'),
					('crm.quotes.write', 'crm',  'Manage and analyze CRM quotes'),
					('crm.tickets.write', 'crm',  'Create and manage CRM support tickets'),
					('accounting.ledger.write',   'accounting', 'Post general ledger journal entries'),
					('accounting.invoices.write', 'accounting', 'Upload and process invoices'),
					('expenses.submit',           'accounting', 'Submit expense reports'),
					('fleet.telematics.ingest',   'fleet',     'Ingest vehicle telemetry data via API key'),
					('fleet.routes.manage',       'fleet',     'Generate and manage optimized fleet routes'),
					('inventory.read',            'inventory', 'Read inventory reorder predictions and stock levels'),
					('hr.attendance.write',       'hr',        'Clock in/out and manage attendance records'),
					('hr.recruitment.write',      'hr',        'Parse resumes and manage job applications'),
					('knowledge.read',            'knowledge', 'Search and read knowledge base documents'),
					('copilot.use',               'platform',  'Use the AI text-to-SQL copilot query feature'),
					('workflows.execute',         'platform',  'Trigger and execute low-code automated workflows'),
					('security.audit.read',       'platform',  'Read security audit anomaly logs'),
					('crm.fieldvisits.write',     'crm',       'Schedule and manage field sales visits'),
					('crm.campaigns.write',       'crm',       'Create and manage marketing campaigns'),
					('accounting.assets.write',    'accounting', 'Register and manage fixed assets'),
					('accounting.assets.read',     'accounting', 'View fixed assets and depreciation'),
					('accounting.tax.manage',      'accounting', 'Create and manage tax rates'),
					('accounting.tax.read',        'accounting', 'Calculate and view tax computations'),
					('accounting.bankrec.write',   'accounting', 'Import bank statements and run reconciliation'),
					('accounting.exchangerates.write', 'accounting', 'Set and manage currency exchange rates'),
					('accounting.currencyconvert.read', 'accounting', 'Convert amounts between currencies'),
					('hr.employees.write',         'hr',        'Create and update employee records'),
					('hr.employees.read',          'hr',        'View employee records'),
					('hr.payroll.write',           'hr',        'Run payroll and manage payroll records'),
					('hr.payroll.read',            'hr',        'View payroll runs and history'),
					('hr.shifts.write',            'hr',        'Create shift templates and assignments'),
					('hr.shifts.read',             'hr',        'View shift schedules and predictions'),
					('hr.attendance.clockout',     'hr',        'Clock out and complete attendance records'),
					('inventory.receive',          'inventory', 'Receive stock into warehouses'),
					('inventory.issue',            'inventory', 'Issue stock from warehouses'),
					('inventory.transfer',         'inventory', 'Transfer stock between warehouses'),
					('inventory.snapshot',         'inventory', 'View current inventory levels'),
					('bi.dashboards.write',        'platform',  'Create and manage BI executive dashboards'),
					('bi.dashboards.read',         'platform',  'View BI executive dashboards'),
					('iot.devices.write',          'platform',  'Register and manage IoT devices'),
					('iot.readings.ingest',        'platform',  'Ingest IoT device readings'),
					('manufacturing.boms.write',   'manufacturing', 'Create and manage bills of materials'),
					('manufacturing.workorders.write', 'manufacturing', 'Create and manage work orders'),
					('procurement.po.write',       'procurement', 'Create and manage purchase orders'),
					('procurement.supplier.read',  'procurement', 'View supplier risk reports')
				ON CONFLICT (permission_key) DO NOTHING`,
		},
	}

	for _, m := range migrations {
		if _, err := db.Pool.Exec(ctx, m.sql); err != nil {
			return fmt.Errorf("migration %q: %w", m.name, err)
		}
		fmt.Printf("  ✓ %s\n", m.name)
	}

	return nil
}
