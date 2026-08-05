CREATE TABLE IF NOT EXISTS products (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					sku VARCHAR(100) NOT NULL,
					name VARCHAR(255) NOT NULL,
					unit_price DECIMAL(10, 2) NOT NULL,
					cost_price DECIMAL(10, 2) NOT NULL,
					ai_reorder_point INT DEFAULT 15,
					UNIQUE(org_id, sku)
				)
