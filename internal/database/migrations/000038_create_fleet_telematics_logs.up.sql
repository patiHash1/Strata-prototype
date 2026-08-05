CREATE TABLE IF NOT EXISTS fleet_telematics_logs (
					id BIGSERIAL PRIMARY KEY,
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					vehicle_id UUID NOT NULL REFERENCES fleet_vehicles(id) ON DELETE CASCADE,
					latitude DECIMAL(10, 8) NOT NULL,
					longitude DECIMAL(11, 8) NOT NULL,
					speed_kmh DECIMAL(5, 2) NOT NULL,
					fuel_level_pct DECIMAL(5, 2),
					recorded_at TIMESTAMPTZ DEFAULT NOW()
				)
