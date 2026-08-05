CREATE TABLE IF NOT EXISTS journal_entries (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					entry_number VARCHAR(100) NOT NULL,
					entry_date DATE NOT NULL DEFAULT CURRENT_DATE,
					memo VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
