CREATE TABLE IF NOT EXISTS fixed_assets (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					asset_name VARCHAR(255) NOT NULL,
					purchase_date DATE NOT NULL,
					purchase_cost DECIMAL(12, 2) NOT NULL,
					salvage_value DECIMAL(12, 2) DEFAULT 0.00,
					useful_life_years INT NOT NULL,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
