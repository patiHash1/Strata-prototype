CREATE TABLE IF NOT EXISTS field_sales_visits (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
	contact_id UUID REFERENCES crm_contacts(id),
	sales_rep_id UUID REFERENCES users(id),
	scheduled_at TIMESTAMPTZ NOT NULL,
	location_lat DECIMAL(10,8),
	location_long DECIMAL(11,8),
	status VARCHAR(50) DEFAULT 'scheduled',
	notes TEXT,
	created_at TIMESTAMPTZ DEFAULT NOW()
)