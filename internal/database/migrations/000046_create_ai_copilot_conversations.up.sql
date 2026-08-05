CREATE TABLE IF NOT EXISTS ai_copilot_conversations (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID REFERENCES users(id),
					prompt_text TEXT NOT NULL,
					generated_sql TEXT,
					response_payload JSONB,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
