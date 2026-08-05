CREATE TABLE IF NOT EXISTS attendance_logs (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					employee_id UUID NOT NULL REFERENCES employees(id) ON DELETE CASCADE,
					clock_in TIMESTAMPTZ NOT NULL,
					clock_out TIMESTAMPTZ,
					location_lat DECIMAL(10, 8),
					location_long DECIMAL(11, 8)
				)
