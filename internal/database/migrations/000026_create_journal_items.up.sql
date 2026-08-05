CREATE TABLE IF NOT EXISTS journal_items (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					journal_entry_id UUID NOT NULL REFERENCES journal_entries(id) ON DELETE CASCADE,
					account_id UUID NOT NULL REFERENCES chart_of_accounts(id),
					debit DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					credit DECIMAL(12, 2) NOT NULL DEFAULT 0.00
				)
