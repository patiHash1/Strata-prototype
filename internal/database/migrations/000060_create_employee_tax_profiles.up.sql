CREATE TABLE IF NOT EXISTS employee_tax_profiles (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					employee_id UUID UNIQUE NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
					tax_country VARCHAR(2) NOT NULL DEFAULT 'US',
					tax_identification_number VARCHAR(50),
					filing_status VARCHAR(50) DEFAULT 'single',
					withholding_allowances INT DEFAULT 0,
					additional_withholding DECIMAL(10, 2) DEFAULT 0.00,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
