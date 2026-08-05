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
				)
