CREATE TABLE IF NOT EXISTS purchase_orders (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					po_number VARCHAR(100) NOT NULL,
					supplier_name VARCHAR(255) NOT NULL,
					total_cost DECIMAL(12, 2) NOT NULL,
					ai_supplier_risk_rating VARCHAR(50) DEFAULT 'Low Risk',
					status VARCHAR(50) DEFAULT 'draft',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
