CREATE TABLE IF NOT EXISTS crm_campaigns (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(255) NOT NULL,
					channel VARCHAR(50) NOT NULL,
					ai_target_segment_criteria JSONB,
					budget DECIMAL(12, 2),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
