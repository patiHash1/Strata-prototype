CREATE TABLE IF NOT EXISTS iot_devices (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					device_name VARCHAR(100) NOT NULL,
					device_type VARCHAR(50) NOT NULL,
					mac_address VARCHAR(100) UNIQUE,
					status VARCHAR(50) DEFAULT 'online',
					last_ping TIMESTAMPTZ DEFAULT NOW()
				)
