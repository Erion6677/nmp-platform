-- Migration: 001_add_monitoring_tables
-- Description: 添加监控相关数据表
-- Date: 2026-01-06

-- =====================================================
-- 1. 创建 proxies 代理表（必须先创建，因为 devices 表引用它）
-- =====================================================
CREATE TABLE IF NOT EXISTS proxies (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL,
    
    -- SSH 代理字段
    ssh_host VARCHAR(255),
    ssh_port INTEGER DEFAULT 22,
    ssh_username VARCHAR(100),
    ssh_password VARCHAR(255),
    
    -- SOCKS5 代理字段
    socks5_host VARCHAR(255),
    socks5_port INTEGER DEFAULT 1080,
    socks5_username VARCHAR(100),
    socks5_password VARCHAR(255),
    
    -- 链式代理
    parent_proxy_id BIGINT REFERENCES proxies(id),
    
    enabled BOOLEAN DEFAULT TRUE,
    status VARCHAR(20) DEFAULT 'disconnected',
    last_error TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- 创建代理索引
CREATE INDEX IF NOT EXISTS idx_proxies_type ON proxies(type);
CREATE INDEX IF NOT EXISTS idx_proxies_enabled ON proxies(enabled);
CREATE INDEX IF NOT EXISTS idx_proxies_status ON proxies(status);
CREATE INDEX IF NOT EXISTS idx_proxies_deleted_at ON proxies(deleted_at);

-- =====================================================
-- 2. 扩展 devices 表（添加 api_port、os_type 和 proxy_id 字段）
-- =====================================================
ALTER TABLE devices ADD COLUMN IF NOT EXISTS api_port INTEGER DEFAULT 8728;
ALTER TABLE devices ADD COLUMN IF NOT EXISTS os_type VARCHAR(20) DEFAULT 'mikrotik';
ALTER TABLE devices ADD COLUMN IF NOT EXISTS proxy_id BIGINT REFERENCES proxies(id);

-- 创建设备索引
CREATE INDEX IF NOT EXISTS idx_devices_os_type ON devices(os_type);
CREATE INDEX IF NOT EXISTS idx_devices_proxy_id ON devices(proxy_id);

-- =====================================================
-- 3. 扩展 interfaces 表
-- =====================================================
ALTER TABLE interfaces ADD COLUMN IF NOT EXISTS monitored BOOLEAN DEFAULT FALSE;
CREATE INDEX IF NOT EXISTS idx_interfaces_monitored ON interfaces(monitored);

-- =====================================================
-- 4. 创建 collector_scripts 采集器配置表
-- =====================================================
CREATE TABLE IF NOT EXISTS collector_scripts (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL UNIQUE REFERENCES devices(id) ON DELETE CASCADE,
    
    enabled BOOLEAN DEFAULT FALSE,
    interval_ms INTEGER DEFAULT 1000,
    push_batch_size INTEGER DEFAULT 10,
    script_name VARCHAR(64) DEFAULT 'nmp-collector',
    scheduler_name VARCHAR(64) DEFAULT 'nmp-scheduler',
    
    deployed_at TIMESTAMPTZ,
    last_push_at TIMESTAMPTZ,
    push_count BIGINT DEFAULT 0,
    
    status VARCHAR(32) DEFAULT 'not_deployed',
    error_message TEXT,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 创建采集器索引
CREATE INDEX IF NOT EXISTS idx_collector_scripts_status ON collector_scripts(status);
CREATE INDEX IF NOT EXISTS idx_collector_scripts_enabled ON collector_scripts(enabled);

-- =====================================================
-- 5. 创建 ping_targets Ping 目标表
-- =====================================================
CREATE TABLE IF NOT EXISTS ping_targets (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    
    target_address VARCHAR(255) NOT NULL,
    target_name VARCHAR(100) NOT NULL,
    source_interface VARCHAR(100) DEFAULT '',
    enabled BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- 创建 Ping 目标索引
CREATE INDEX IF NOT EXISTS idx_ping_targets_device_id ON ping_targets(device_id);
CREATE INDEX IF NOT EXISTS idx_ping_targets_enabled ON ping_targets(enabled);
CREATE INDEX IF NOT EXISTS idx_ping_targets_deleted_at ON ping_targets(deleted_at);

-- =====================================================
-- 6. 创建 system_settings 系统设置表
-- =====================================================
CREATE TABLE IF NOT EXISTS system_settings (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 插入默认设置
INSERT INTO system_settings (key, value, description) VALUES
    ('default_push_interval', '1000', '默认推送间隔（毫秒）'),
    ('data_retention_days', '10', '数据保留天数'),
    ('frontend_refresh_interval', '10', '前端刷新间隔（秒）'),
    ('device_offline_timeout', '60', '设备离线超时时间（秒）')
ON CONFLICT (key) DO NOTHING;

-- =====================================================
-- 7. 创建 user_device_permissions 用户设备权限表
-- =====================================================
CREATE TABLE IF NOT EXISTS user_device_permissions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id BIGINT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, device_id)
);

-- 创建用户设备权限索引
CREATE INDEX IF NOT EXISTS idx_user_device_permissions_user_id ON user_device_permissions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_device_permissions_device_id ON user_device_permissions(device_id);

-- =====================================================
-- 8. 添加监控相关权限
-- =====================================================
INSERT INTO permissions (resource, action, scope, description, created_at, updated_at) VALUES
    ('proxy', 'create', 'all', '创建代理', NOW(), NOW()),
    ('proxy', 'read', 'all', '查看代理', NOW(), NOW()),
    ('proxy', 'update', 'all', '更新代理', NOW(), NOW()),
    ('proxy', 'delete', 'all', '删除代理', NOW(), NOW()),
    ('collector', 'deploy', 'all', '部署采集器', NOW(), NOW()),
    ('collector', 'manage', 'all', '管理采集器', NOW(), NOW()),
    ('monitoring', 'read', 'all', '查看监控数据', NOW(), NOW()),
    ('monitoring', 'config', 'all', '配置监控设置', NOW(), NOW())
ON CONFLICT DO NOTHING;

-- =====================================================
-- 完成
-- =====================================================
