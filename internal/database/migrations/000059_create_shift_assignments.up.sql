CREATE TABLE IF NOT EXISTS shift_assignments (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					shift_template_id UUID NOT NULL REFERENCES shift_templates(id),
					employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
					shift_date DATE NOT NULL,
					actual_start TIMESTAMPTZ,
					actual_end TIMESTAMPTZ,
					status VARCHAR(50) DEFAULT 'scheduled',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
