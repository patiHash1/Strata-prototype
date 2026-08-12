CREATE TABLE IF NOT EXISTS super_admin_soc_events (
    id VARCHAR(36) PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL DEFAULT 'info',
    message TEXT NOT NULL,
    ip_address VARCHAR(45) NOT NULL DEFAULT '',
    user_id VARCHAR(36) NOT NULL DEFAULT '',
    org_id VARCHAR(36) NOT NULL DEFAULT '',
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_soc_events_type ON super_admin_soc_events(event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_soc_events_created ON super_admin_soc_events(created_at DESC);
