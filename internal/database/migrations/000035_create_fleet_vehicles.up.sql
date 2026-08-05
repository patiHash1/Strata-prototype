CREATE TABLE IF NOT EXISTS fleet_vehicles (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					vin VARCHAR(17) UNIQUE NOT NULL,
					license_plate VARCHAR(20) NOT NULL,
					make VARCHAR(50) NOT NULL,
					model VARCHAR(50) NOT NULL,
					status vehicle_status DEFAULT 'active',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
