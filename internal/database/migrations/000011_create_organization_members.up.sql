CREATE TABLE IF NOT EXISTS organization_members (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					role_id UUID NOT NULL REFERENCES roles(id),
					is_active BOOLEAN DEFAULT TRUE,
					joined_at TIMESTAMPTZ DEFAULT NOW(),
					UNIQUE(org_id, user_id)
				)
