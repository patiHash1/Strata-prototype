CREATE TABLE IF NOT EXISTS roles (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					description VARCHAR(255),
					is_system_default BOOLEAN DEFAULT FALSE,
					UNIQUE(org_id, name)
				)
