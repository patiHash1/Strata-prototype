CREATE TABLE IF NOT EXISTS chart_of_accounts (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					account_code VARCHAR(20) NOT NULL,
					account_name VARCHAR(100) NOT NULL,
					account_type VARCHAR(50) NOT NULL,
					UNIQUE(org_id, account_code)
				)
