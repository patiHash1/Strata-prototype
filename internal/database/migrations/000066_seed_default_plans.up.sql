INSERT INTO subscription_plans (id, plan_code, name, max_tenants_limit, max_vehicles_limit, max_ai_credits_per_month, monthly_price)
				VALUES
					(gen_random_uuid(), 'starter', 'Starter', 1, 5, 1000, 49.00),
					(gen_random_uuid(), 'professional', 'Professional', 3, 25, 5000, 149.00),
					(gen_random_uuid(), 'enterprise', 'Enterprise', 10, 100, 25000, 499.00)
				ON CONFLICT (plan_code) DO NOTHING
