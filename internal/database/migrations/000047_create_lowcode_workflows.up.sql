CREATE TABLE IF NOT EXISTS lowcode_workflows (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					trigger_event VARCHAR(100) NOT NULL,
					action_steps JSONB NOT NULL,
					is_active BOOLEAN DEFAULT TRUE,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
