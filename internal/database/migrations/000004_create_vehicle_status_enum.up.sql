DO $$ BEGIN
					CREATE TYPE vehicle_status AS ENUM ('active', 'in_transit', 'maintenance', 'decommissioned');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$
