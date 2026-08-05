CREATE TABLE IF NOT EXISTS iot_device_readings (
					id BIGSERIAL PRIMARY KEY,
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					device_id UUID NOT NULL REFERENCES iot_devices(id) ON DELETE CASCADE,
					metric_name VARCHAR(100) NOT NULL,
					metric_value DOUBLE PRECISION NOT NULL,
					unit VARCHAR(50),
					recorded_at TIMESTAMPTZ DEFAULT NOW()
				)
