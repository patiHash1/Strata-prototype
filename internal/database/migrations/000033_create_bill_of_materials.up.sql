CREATE TABLE IF NOT EXISTS bill_of_materials (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					parent_product_id UUID NOT NULL REFERENCES products(id),
					bom_code VARCHAR(50) NOT NULL
				)
