CREATE TABLE IF NOT EXISTS knowledge_base_documents (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					title VARCHAR(255) NOT NULL,
					content TEXT NOT NULL,
					vector_embedding_id VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
