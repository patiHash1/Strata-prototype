CREATE TABLE IF NOT EXISTS ai_usage_logs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					feature_used VARCHAR(100) NOT NULL,
					credits_consumed INT NOT NULL DEFAULT 1,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
