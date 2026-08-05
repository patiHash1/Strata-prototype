CREATE TABLE IF NOT EXISTS payroll_disbursements (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					payroll_run_id UUID NOT NULL REFERENCES payroll_runs(id) ON DELETE CASCADE,
					employee_id UUID NOT NULL REFERENCES employees(id),
					gross_pay DECIMAL(12, 2) NOT NULL,
					tax_withheld DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					social_security DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					other_deductions DECIMAL(12, 2) NOT NULL DEFAULT 0.00,
					net_pay DECIMAL(12, 2) NOT NULL,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
