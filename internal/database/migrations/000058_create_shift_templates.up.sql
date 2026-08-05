CREATE TABLE IF NOT EXISTS shift_templates (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(100) NOT NULL,
					start_time TIME NOT NULL,
					end_time TIME NOT NULL,
					day_of_week SMALLINT,
					department VARCHAR(100),
					required_headcount INT DEFAULT 1,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
