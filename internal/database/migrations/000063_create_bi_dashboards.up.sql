CREATE TABLE IF NOT EXISTS bi_dashboards (
						id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
						org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
						name VARCHAR(255) NOT NULL,
						config JSONB NOT NULL DEFAULT '{}',
						is_active BOOLEAN DEFAULT TRUE,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						updated_at TIMESTAMPTZ DEFAULT NOW()
					)
