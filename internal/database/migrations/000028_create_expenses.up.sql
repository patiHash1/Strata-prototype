CREATE TABLE IF NOT EXISTS expenses (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					amount DECIMAL(10, 2) NOT NULL,
					category VARCHAR(100) NOT NULL,
					receipt_url TEXT,
					ai_fraud_flag BOOLEAN DEFAULT FALSE,
					ai_audit_notes TEXT,
					status VARCHAR(50) DEFAULT 'pending',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
