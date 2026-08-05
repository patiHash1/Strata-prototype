CREATE TABLE IF NOT EXISTS exchange_rates (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					from_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
					to_currency VARCHAR(3) NOT NULL REFERENCES currencies(code),
					rate DECIMAL(14, 8) NOT NULL,
					effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
					source VARCHAR(100) DEFAULT 'manual',
					created_at TIMESTAMPTZ DEFAULT NOW(),
					UNIQUE(org_id, from_currency, to_currency, effective_date)
				)
