INSERT INTO permissions (permission_key, module, description)
VALUES ('super_admin.access', 'super_admin', 'Full super-admin access to system observability and maintenance')
ON CONFLICT (permission_key) DO NOTHING;