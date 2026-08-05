DO $$ BEGIN
					CREATE TYPE ticket_priority AS ENUM ('low', 'medium', 'high', 'urgent');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$
