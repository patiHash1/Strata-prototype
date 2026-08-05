CREATE TABLE IF NOT EXISTS stock_movements (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					movement_type VARCHAR(20) NOT NULL,
					from_warehouse_id UUID REFERENCES warehouses(id),
					to_warehouse_id UUID REFERENCES warehouses(id),
					product_id UUID NOT NULL REFERENCES products(id),
					quantity DECIMAL(12, 4) NOT NULL,
					reference VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
