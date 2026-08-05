CREATE TABLE IF NOT EXISTS subscription_plans (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					plan_code VARCHAR(50) UNIQUE NOT NULL,
					name VARCHAR(100) NOT NULL,
					max_tenants_limit INT NOT NULL,
					max_vehicles_limit INT NOT NULL,
					max_ai_credits_per_month INT NOT NULL,
					monthly_price DECIMAL(10, 2) NOT NULL
				)
