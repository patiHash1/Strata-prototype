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
				)
