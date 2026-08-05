CREATE TABLE IF NOT EXISTS shipments (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					tracking_number VARCHAR(100) UNIQUE NOT NULL,
					origin_address TEXT NOT NULL,
					destination_address TEXT NOT NULL,
					status shipment_status DEFAULT 'pending',
					assigned_vehicle_id UUID REFERENCES fleet_vehicles(id),
					assigned_driver_id UUID REFERENCES fleet_drivers(id),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
