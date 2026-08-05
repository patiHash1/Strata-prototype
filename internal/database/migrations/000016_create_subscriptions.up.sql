CREATE TABLE IF NOT EXISTS subscriptions (
					id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
					org_id UUID UNIQUE NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					plan_id UUID NOT NULL REFERENCES subscription_plans(id),
					stripe_customer_id VARCHAR(255),
					stripe_subscription_id VARCHAR(255),
					status subscription_status DEFAULT 'trialing',
					current_period_start TIMESTAMPTZ NOT NULL DEFAULT NOW(),
					current_period_end TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days'),
					created_at TIMESTAMPTZ DEFAULT NOW()
				)
