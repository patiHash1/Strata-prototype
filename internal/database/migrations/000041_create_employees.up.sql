CREATE TABLE IF NOT EXISTS employees (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID UNIQUE REFERENCES users(id),
					employee_code VARCHAR(50) NOT NULL,
					department VARCHAR(100),
					job_title VARCHAR(100),
					salary DECIMAL(12, 2),
					hired_at DATE NOT NULL,
					UNIQUE(org_id, employee_code)
				)
