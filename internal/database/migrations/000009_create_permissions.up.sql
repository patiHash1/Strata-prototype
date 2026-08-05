CREATE TABLE IF NOT EXISTS permissions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					permission_key VARCHAR(100) UNIQUE NOT NULL,
					module VARCHAR(50) NOT NULL,
					description VARCHAR(255)
				)
