CREATE TABLE IF NOT EXISTS crm_quotes (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					deal_id UUID REFERENCES crm_deals(id) ON DELETE CASCADE,
					quote_number VARCHAR(100) NOT NULL,
					total_amount DECIMAL(12, 2) NOT NULL,
					ai_risk_score DECIMAL(5,2),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
