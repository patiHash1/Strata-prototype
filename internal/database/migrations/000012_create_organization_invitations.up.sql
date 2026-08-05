CREATE TABLE IF NOT EXISTS organization_invitations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					email VARCHAR(255) NOT NULL,
					role_id UUID NOT NULL REFERENCES roles(id),
					token VARCHAR(255) UNIQUE NOT NULL,
					expires_at TIMESTAMPTZ NOT NULL,
					accepted_at TIMESTAMPTZ,
					created_by UUID REFERENCES users(id)
				)
