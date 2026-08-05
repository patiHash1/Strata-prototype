CREATE TABLE IF NOT EXISTS audit_logs (
					id BIGSERIAL PRIMARY KEY,
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					action VARCHAR(100) NOT NULL,
					ip_address VARCHAR(45),
					ai_anomaly_flag BOOLEAN DEFAULT FALSE,
					metadata JSONB,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
