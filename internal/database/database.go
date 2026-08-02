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
			name: "create_enums_if_missing",
			sql:  `SELECT 1`,
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
					status VARCHAR(50) DEFAULT 'active',
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
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					plan_code VARCHAR(50) NOT NULL,
					status VARCHAR(50) DEFAULT 'trialing',
					stripe_customer_id VARCHAR(255),
					stripe_sub_id VARCHAR(255),
					current_period_end TIMESTAMPTZ,
					created_at TIMESTAMPTZ DEFAULT NOW(),
					updated_at TIMESTAMPTZ DEFAULT NOW()
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
					priority VARCHAR(50) DEFAULT 'medium',
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
					('expenses.submit',           'accounting', 'Submit expense reports')
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
