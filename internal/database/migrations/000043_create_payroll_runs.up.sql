CREATE TABLE IF NOT EXISTS payroll_runs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					pay_period_start DATE NOT NULL,
					pay_period_end DATE NOT NULL,
					total_disbursed DECIMAL(14, 2) NOT NULL,
					status VARCHAR(50) DEFAULT 'draft',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
