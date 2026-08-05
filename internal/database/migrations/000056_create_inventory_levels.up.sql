CREATE TABLE IF NOT EXISTS inventory_levels (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					warehouse_id UUID NOT NULL REFERENCES warehouses(id) ON DELETE CASCADE,
					product_id UUID NOT NULL REFERENCES products(id) ON DELETE CASCADE,
					quantity_on_hand DECIMAL(12, 4) NOT NULL DEFAULT 0,
					quantity_reserved DECIMAL(12, 4) NOT NULL DEFAULT 0,
					quantity_available DECIMAL(12, 4) GENERATED ALWAYS AS (quantity_on_hand - quantity_reserved) STORED,
					last_counted_at TIMESTAMPTZ,
					UNIQUE(warehouse_id, product_id)
				)
