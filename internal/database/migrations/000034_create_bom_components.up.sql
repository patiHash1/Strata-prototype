CREATE TABLE IF NOT EXISTS bom_components (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					bom_id UUID NOT NULL REFERENCES bill_of_materials(id) ON DELETE CASCADE,
					component_product_id UUID NOT NULL REFERENCES products(id),
					quantity_required DECIMAL(10, 4) NOT NULL
				)
