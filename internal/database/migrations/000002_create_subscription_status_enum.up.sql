DO $$ BEGIN
					CREATE TYPE subscription_status AS ENUM ('active', 'past_due', 'canceled', 'trialing');
				EXCEPTION WHEN duplicate_object THEN NULL;
				END $$
