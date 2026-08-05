CREATE TABLE IF NOT EXISTS job_applications (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					candidate_name VARCHAR(150) NOT NULL,
					email VARCHAR(255) NOT NULL,
					resume_url TEXT NOT NULL,
					ai_match_score INT,
					status VARCHAR(50) DEFAULT 'applied',
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
