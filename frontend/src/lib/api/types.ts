/**
 * API 类型定义
 */

// ==================== 通用类型 ====================

export interface PaginatedResponse<T> {
  items: T[];
  total: number;
  page: number;
  size: number;
  total_pages?: number;
}

// ==================== 认证相关 ====================

export interface LoginParams {
  username: string;
  password: string;
}

export interface LoginResult {
  token: string;
  user?: UserInfo;
  userInfo?: UserInfo;
}

export interface UserInfo {
  id: number;
  username: string;
  email?: string;
  full_name?: string;
  roles?: string[];
  avatar?: string;
}

// ==================== 设备相关 ====================

export type DeviceType = 'mikrotik' | 'linux' | 'switch' | 'firewall';
export type DeviceStatus = 'online' | 'offline' | 'unknown' | 'error';
export type InterfaceStatus = 'up' | 'down' | 'unknown';

export interface Device {
  id: number;
  name: string;
  type: DeviceType;
  os_type?: 'mikrotik' | 'linux';
  host: string;
  port: number;
  api_port?: number;
  protocol?: string;
  username: string;
  version?: string;
  description?: string;
  status: DeviceStatus;
  last_seen?: string;
  proxy_id?: number;
  tags?: Tag[];
  interfaces?: DeviceInterface[];
  groups?: DeviceGroup[];
  created_at: string;
  updated_at: string;
}

export interface DeviceInterface {
  id: number;
  device_id: number;
  name: string;
  description?: string;
  type?: string;
  speed?: number;
  mtu?: number;
  mac_address?: string;
  status: InterfaceStatus;
  monitored: boolean;
  created_at: string;
  updated_at: string;
}

export interface Tag {
  id: number;
  name: string;
  color: string;
  description?: string;
}

export interface DeviceGroup {
  id: number;
  name: string;
  description?: string;
  parent_id?: number;
}

export interface CreateDeviceRequest {
  name: string;
  type: DeviceType;
  os_type?: 'mikrotik' | 'linux';
  host: string;
  port: number;
  api_port?: number;
  protocol?: string;
  username: string;
  password: string;
  description?: string;
}

export interface UpdateDeviceRequest {
  name?: string;
  type?: DeviceType;
  os_type?: 'mikrotik' | 'linux';
  host?: string;
  port?: number;
  api_port?: number;
  protocol?: string;
  username?: string;
  password?: string;
  description?: string;
}

export interface TestConnectionRequest {
  host: string;
  port: number;
  type: 'api' | 'ssh';
  username: string;
  password: string;
  device_type: 'mikrotik' | 'linux';
}

export interface TestConnectionResponse {
  success: boolean;
  message: string;
  latency?: number;
  error_type?: 'network' | 'auth' | 'port' | 'timeout';
}

export interface SystemInfoResponse {
  device_name: string;
  device_ip: string;
  cpu_count: number;
  version: string;
  license: string;
  uptime: number;
  cpu_usage: number;
  memory_usage: number;
  memory_total: number;
  memory_free: number;
}

// ==================== Ping 目标 ====================

export interface PingTarget {
  id: number;
  device_id: number;
  target_address: string;
  target_name: string;
  source_interface: string;
  enabled: boolean;
}

export interface CreatePingTargetRequest {
  target_address: string;
  target_name: string;
  source_interface?: string;
  enabled?: boolean;
}

export interface UpdatePingTargetRequest {
  target_address?: string;
  target_name?: string;
  source_interface?: string;
  enabled?: boolean;
}

// ==================== 采集器 ====================

export interface CollectorStatus {
  device_id: number;
  interval_ms: number;
  push_batch_size: number;
  enabled: boolean;
  status: string;
  deployed_at: string | null;
  last_push_at: string | null;
  push_count: number;
}

export interface CollectorDeviceStatus {
  script_exists: boolean;
  scheduler_exists: boolean;
  scheduler_enabled: boolean;
}

// ==================== 监控数据 ====================

export interface BandwidthPoint {
  timestamp: string;
  rx_rate: number;
  tx_rate: number;
}

export interface BandwidthQueryResponse {
  device_id: string;
  start_time: string;
  end_time: string;
  interfaces: Record<string, BandwidthPoint[]>;
}

export interface PingPoint {
  timestamp: string;
  latency: number;
  status: string;
  is_loss: boolean;
}

export interface PingStats {
  total_count: number;
  loss_count: number;
  loss_rate: number;
  avg_latency: number;
  min_latency: number;
  max_latency: number;
}

export interface PingQueryResponse {
  device_id: string;
  start_time: string;
  end_time: string;
  targets: Record<string, PingPoint[]>;
  stats: Record<string, PingStats>;
}

export interface TotalTrafficPoint {
  timestamp: string;
  inbound: number;
  outbound: number;
}

export interface TotalTrafficResponse {
  start_time: string;
  end_time: string;
  points: TotalTrafficPoint[];
}

export type TimeRangeType = '10m' | '30m' | '1h' | '3h' | '6h' | '12h' | '24h' | 'custom';

// ==================== 用户管理 ====================

export interface User {
  id: number;
  username: string;
  email: string;
  full_name: string;
  status: 'active' | 'disabled';
  roles: Role[];
  last_login?: string;
  created_at: string;
  updated_at: string;
}

export interface Role {
  id: number;
  name: string;
  display_name: string;
  description?: string;
  is_system: boolean;
  permissions?: Permission[];
}

export interface Permission {
  id: number;
  resource: string;
  action: string;
  scope: string;
  description?: string;
}

export interface CreateUserRequest {
  username: string;
  password: string;
  email?: string;
  full_name?: string;
  status?: string;
  role_ids?: number[];
}

export interface UpdateUserRequest {
  email?: string;
  full_name?: string;
  status?: string;
  password?: string;
}

// ==================== 系统设置 ====================

export interface CollectionSettings {
  default_push_interval: number;
  data_retention_days: number;
  frontend_refresh_interval: number;
  device_offline_timeout: number;
  follow_push_interval: boolean;
}

export interface UpdateCollectionSettingsRequest {
  default_push_interval?: number;
  data_retention_days?: number;
  frontend_refresh_interval?: number;
  device_offline_timeout?: number;
  follow_push_interval?: boolean;
}

// ==================== 概览统计 ====================

export interface OverviewStats {
  total_devices: number;
  online_devices: number;
  offline_devices: number;
  warning_devices: number;
}
