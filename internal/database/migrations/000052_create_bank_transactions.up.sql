CREATE TABLE IF NOT EXISTS bank_transactions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					statement_id UUID NOT NULL REFERENCES bank_statements(id) ON DELETE CASCADE,
					transaction_date DATE NOT NULL,
					description VARCHAR(500) NOT NULL,
					reference VARCHAR(100),
					debit DECIMAL(14, 2) DEFAULT 0.00,
					credit DECIMAL(14, 2) DEFAULT 0.00,
					amount DECIMAL(14, 2) NOT NULL,
					is_matched BOOLEAN DEFAULT FALSE,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
