DO $$ BEGIN
					CREATE TYPE shipment_status AS ENUM ('pending', 'assigned', 'in_transit', 'delivered', 'delayed');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$
