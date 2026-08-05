CREATE TABLE IF NOT EXISTS tax_rates (
						id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
						org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
						country_code VARCHAR(2) NOT NULL,
						tax_name VARCHAR(50) NOT NULL,
						tax_rate DECIMAL(5,4) NOT NULL,
						is_active BOOLEAN DEFAULT TRUE,
						created_at TIMESTAMPTZ DEFAULT NOW(),
						UNIQUE(org_id, country_code, tax_name)
					)
