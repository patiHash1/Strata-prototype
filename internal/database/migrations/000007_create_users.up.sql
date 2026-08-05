CREATE TABLE IF NOT EXISTS users (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					email VARCHAR(255) UNIQUE NOT NULL,
					password_hash VARCHAR(255) NOT NULL,
					full_name VARCHAR(150) NOT NULL,
					phone_number VARCHAR(50),
					mfa_enabled BOOLEAN DEFAULT FALSE,
					mfa_secret VARCHAR(255),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
