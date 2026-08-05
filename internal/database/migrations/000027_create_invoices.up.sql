CREATE TABLE IF NOT EXISTS invoices (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					invoice_number VARCHAR(100) NOT NULL,
					contact_id UUID REFERENCES crm_contacts(id),
					total_amount DECIMAL(12, 2) NOT NULL,
					status VARCHAR(50) DEFAULT 'draft',
					ai_ocr_processed BOOLEAN DEFAULT FALSE,
					due_date DATE NOT NULL,
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
