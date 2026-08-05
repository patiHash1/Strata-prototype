CREATE TABLE IF NOT EXISTS reconciliation_matches (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					bank_transaction_id UUID NOT NULL REFERENCES bank_transactions(id) ON DELETE CASCADE,
					journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
					match_type VARCHAR(50) DEFAULT 'auto',
					match_date TIMESTAMPTZ DEFAULT NOW(),
					UNIQUE(bank_transaction_id, journal_entry_id)
				)
