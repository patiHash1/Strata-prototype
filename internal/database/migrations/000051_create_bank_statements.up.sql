CREATE TABLE IF NOT EXISTS bank_statements (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					bank_name VARCHAR(255) NOT NULL,
					account_number VARCHAR(100) NOT NULL,
					statement_date DATE NOT NULL,
					opening_balance DECIMAL(14, 2) NOT NULL,
					closing_balance DECIMAL(14, 2) NOT NULL,
					status VARCHAR(50) DEFAULT 'imported',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
