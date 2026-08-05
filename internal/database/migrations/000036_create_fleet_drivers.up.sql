CREATE TABLE IF NOT EXISTS fleet_drivers (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
					license_number VARCHAR(100) NOT NULL,
					safety_rating DECIMAL(3, 2) DEFAULT 5.00
				)
