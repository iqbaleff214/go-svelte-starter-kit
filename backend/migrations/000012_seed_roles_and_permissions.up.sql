-- Seed default roles
INSERT INTO roles (id, name, description) VALUES
    (uuid_generate_v4(), 'superadmin', 'Full system access with all permissions'),
    (uuid_generate_v4(), 'admin',      'Administrative access to manage users and content'),
    (uuid_generate_v4(), 'user',       'Standard authenticated user');

-- Seed default permissions
INSERT INTO permissions (id, name, description) VALUES
    (uuid_generate_v4(), 'users:read',       'View user list and user details'),
    (uuid_generate_v4(), 'users:write',      'Create and update users'),
    (uuid_generate_v4(), 'users:delete',     'Delete users'),
    (uuid_generate_v4(), 'roles:manage',     'Manage roles and permissions'),
    (uuid_generate_v4(), 'notifications:read',  'Read own notifications'),
    (uuid_generate_v4(), 'notifications:write', 'Create and manage notifications'),
    (uuid_generate_v4(), 'profile:read',     'Read own profile'),
    (uuid_generate_v4(), 'profile:write',    'Update own profile'),
    (uuid_generate_v4(), 'api_keys:manage',  'Create and revoke API keys'),
    (uuid_generate_v4(), 'ai:access',        'Access AI assistant'),
    (uuid_generate_v4(), 'admin:access',     'Access admin panel');

-- Assign all permissions to superadmin
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r, permissions p
WHERE r.name = 'superadmin';

-- Assign admin permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN (
    'users:read', 'users:write',
    'notifications:read', 'notifications:write',
    'profile:read', 'profile:write',
    'api_keys:manage', 'ai:access', 'admin:access'
)
WHERE r.name = 'admin';

-- Assign user permissions
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.name IN (
    'notifications:read',
    'profile:read', 'profile:write',
    'ai:access'
)
WHERE r.name = 'user';
