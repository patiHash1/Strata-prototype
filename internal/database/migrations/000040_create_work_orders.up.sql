CREATE TABLE IF NOT EXISTS work_orders (
						id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
						org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
						bom_id UUID NOT NULL REFERENCES bill_of_materials(id),
						quantity INT NOT NULL DEFAULT 1,
						status VARCHAR(50) DEFAULT 'planned',
						scheduled_start DATE,
						scheduled_end DATE,
						ai_bottleneck_risk VARCHAR(50) DEFAULT 'Low',
						created_at TIMESTAMPTZ DEFAULT NOW()
					)
