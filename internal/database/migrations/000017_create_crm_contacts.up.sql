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
				)
