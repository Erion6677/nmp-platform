-- NMP Platform 数据库初始化脚本

-- 创建扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 创建用户表
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    email VARCHAR(100) UNIQUE,
    status INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建角色表
CREATE TABLE IF NOT EXISTS roles (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建权限表
CREATE TABLE IF NOT EXISTS permissions (
    id SERIAL PRIMARY KEY,
    resource VARCHAR(100) NOT NULL,
    action VARCHAR(50) NOT NULL,
    scope VARCHAR(50) DEFAULT 'global',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(resource, action, scope)
);

-- 创建用户角色关联表
CREATE TABLE IF NOT EXISTS user_roles (
    user_id INTEGER REFERENCES users(id) ON DELETE CASCADE,
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id)
);

-- 创建角色权限关联表
CREATE TABLE IF NOT EXISTS role_permissions (
    role_id INTEGER REFERENCES roles(id) ON DELETE CASCADE,
    permission_id INTEGER REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- 创建设备表
CREATE TABLE IF NOT EXISTS devices (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port INTEGER DEFAULT 22,
    protocol VARCHAR(20) DEFAULT 'ssh',
    username VARCHAR(100),
    status INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建设备分组表
CREATE TABLE IF NOT EXISTS device_groups (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建设备分组关联表
CREATE TABLE IF NOT EXISTS device_group_members (
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    device_group_id INTEGER REFERENCES device_groups(id) ON DELETE CASCADE,
    PRIMARY KEY (device_id, device_group_id)
);

-- 创建标签表
CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) UNIQUE NOT NULL,
    color VARCHAR(7) DEFAULT '#1890ff',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 创建设备标签关联表
CREATE TABLE IF NOT EXISTS device_tags (
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    tag_id INTEGER REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (device_id, tag_id)
);

-- 创建接口表
CREATE TABLE IF NOT EXISTS interfaces (
    id SERIAL PRIMARY KEY,
    device_id INTEGER REFERENCES devices(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    type VARCHAR(50),
    speed BIGINT,
    status INTEGER DEFAULT 1,
    monitor BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- 插入默认数据
INSERT INTO roles (name, description) VALUES 
    ('admin', '系统管理员'),
    ('operator', '操作员'),
    ('viewer', '查看者')
ON CONFLICT (name) DO NOTHING;

INSERT INTO permissions (resource, action) VALUES 
    ('users', 'create'),
    ('users', 'read'),
    ('users', 'update'),
    ('users', 'delete'),
    ('devices', 'create'),
    ('devices', 'read'),
    ('devices', 'update'),
    ('devices', 'delete'),
    ('monitoring', 'read'),
    ('system', 'manage')
ON CONFLICT (resource, action, scope) DO NOTHING;

-- 创建默认管理员用户 (密码: admin123)
INSERT INTO users (username, password, email) VALUES 
    ('admin', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', 'admin@nmp.local')
ON CONFLICT (username) DO NOTHING;

-- 分配管理员角色
INSERT INTO user_roles (user_id, role_id) 
SELECT u.id, r.id FROM users u, roles r 
WHERE u.username = 'admin' AND r.name = 'admin'
ON CONFLICT DO NOTHING;

-- 分配管理员权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.name = 'admin'
ON CONFLICT DO NOTHING;