CREATE TABLE IF NOT EXISTS super_admin_maintenance_rules (
    id BIGSERIAL PRIMARY KEY,
    scope VARCHAR(50) NOT NULL,
    target_id VARCHAR(255) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    reason TEXT NOT NULL DEFAULT '',
    allowed_roles TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(scope, target_id)
);

CREATE TABLE IF NOT EXISTS super_admin_system_errors (
    id BIGSERIAL PRIMARY KEY,
    module VARCHAR(100) NOT NULL DEFAULT 'system',
    error_message TEXT NOT NULL,
    stack_trace TEXT NOT NULL DEFAULT '',
    status_code INTEGER NOT NULL DEFAULT 500,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS super_admin_ci_health_reports (
    id BIGSERIAL PRIMARY KEY,
    module VARCHAR(100) NOT NULL,
    coverage_percent NUMERIC(5,2) NOT NULL DEFAULT 0,
    linter_issues INTEGER NOT NULL DEFAULT 0,
    vulnerabilities_count INTEGER NOT NULL DEFAULT 0,
    commit_sha VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_maintenance_active ON super_admin_maintenance_rules(is_active, scope);
CREATE INDEX IF NOT EXISTS idx_system_errors_module ON super_admin_system_errors(module, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_ci_health_module ON super_admin_ci_health_reports(module, created_at DESC);