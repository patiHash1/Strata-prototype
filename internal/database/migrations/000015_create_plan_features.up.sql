CREATE TABLE IF NOT EXISTS plan_features (
					plan_id UUID NOT NULL REFERENCES subscription_plans(id) ON DELETE CASCADE,
					feature_key VARCHAR(100) NOT NULL,
					PRIMARY KEY (plan_id, feature_key)
				)
