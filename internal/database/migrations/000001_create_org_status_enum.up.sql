DO $$ BEGIN
					CREATE TYPE org_status AS ENUM ('active', 'suspended', 'pending_verification');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$
