CREATE TABLE IF NOT EXISTS crm_helpdesk_tickets (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					contact_id UUID REFERENCES crm_contacts(id),
					subject VARCHAR(255) NOT NULL,
					description TEXT NOT NULL,
					priority ticket_priority DEFAULT 'medium',
					status VARCHAR(50) DEFAULT 'open',
					ai_sentiment_score DECIMAL(3, 2),
					ai_suggested_response TEXT,
					assigned_to UUID REFERENCES users(id),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
