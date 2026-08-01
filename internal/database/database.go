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
					('crm.quotes.write', 'crm',  'Manage and analyze CRM quotes')
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
