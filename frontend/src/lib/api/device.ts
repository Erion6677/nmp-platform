/**
 * 设备 API
 */

import api from './client';
import type {
  Device,
  DeviceInterface,
  CreateDeviceRequest,
  UpdateDeviceRequest,
  TestConnectionRequest,
  TestConnectionResponse,
  SystemInfoResponse,
  PingTarget,
  CreatePingTargetRequest,
  UpdatePingTargetRequest,
  CollectorStatus,
  DeviceStatus,
} from './types';

interface ListDevicesParams {
  page?: number;
  page_size?: number;
  type?: string;
  status?: DeviceStatus;
  group_id?: number;
  tag_id?: number;
  search?: string;
}

interface ListDevicesResponse {
  devices: Device[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export const deviceApi = {
  // 测试设备连接
  testConnection(data: TestConnectionRequest) {
    return api.post<{ success: boolean; data: TestConnectionResponse }>('/api/v1/devices/test-connection', data);
  },

  // 创建设备
  createDevice(data: CreateDeviceRequest) {
    return api.post<{ success: boolean; data: Device }>('/api/v1/devices', data);
  },

  // 获取设备详情
  getDevice(id: number) {
    return api.get<{ success: boolean; data: Device }>(`/api/v1/devices/${id}`);
  },

  // 更新设备
  updateDevice(id: number, data: UpdateDeviceRequest) {
    return api.put<{ success: boolean; data: Device }>(`/api/v1/devices/${id}`, data);
  },

  // 删除设备
  deleteDevice(id: number) {
    return api.delete<{ success: boolean; message: string }>(`/api/v1/devices/${id}`);
  },

  // 获取设备列表
  listDevices(params?: ListDevicesParams) {
    return api.get<{ success: boolean; data: ListDevicesResponse }>('/api/v1/devices', params as Record<string, string | number | boolean | undefined>);
  },

  // 获取设备接口列表
  getDeviceInterfaces(id: number) {
    return api.get<{ success: boolean; data: DeviceInterface[] }>(`/api/v1/devices/${id}/interfaces`);
  },

  // 更新接口监控状态
  updateInterfaceMonitorStatus(interfaceId: number, monitor: boolean) {
    return api.put<{ success: boolean; message: string }>(`/api/v1/devices/interfaces/${interfaceId}/monitor`, { monitor });
  },

  // 同步设备接口
  syncInterfaces(id: number) {
    return api.post<{ success: boolean; data: { interfaces: DeviceInterface[]; synced_count: number }; message: string }>(`/api/v1/devices/${id}/interfaces/sync`);
  },

  // 批量设置监控接口
  setMonitoredInterfaces(id: number, interfaceNames: string[]) {
    return api.put<{ success: boolean; message: string }>(`/api/v1/devices/${id}/interfaces/monitored`, { interface_names: interfaceNames });
  },

  // 获取设备系统信息
  getSystemInfo(id: number) {
    return api.get<{ success: boolean; data: SystemInfoResponse }>(`/api/v1/devices/${id}/info`);
  },

  // 获取设备 Ping 目标列表
  getPingTargets(id: number) {
    return api.get<{ success: boolean; data: PingTarget[] }>(`/api/v1/devices/${id}/ping-targets`);
  },

  // 创建 Ping 目标
  createPingTarget(deviceId: number, data: CreatePingTargetRequest) {
    return api.post<{ success: boolean; data: PingTarget; message: string }>(`/api/v1/devices/${deviceId}/ping-targets`, data);
  },

  // 更新 Ping 目标
  updatePingTarget(deviceId: number, targetId: number, data: UpdatePingTargetRequest) {
    return api.put<{ success: boolean; data: PingTarget; message: string }>(`/api/v1/devices/${deviceId}/ping-targets/${targetId}`, data);
  },

  // 删除 Ping 目标
  deletePingTarget(deviceId: number, targetId: number) {
    return api.delete<{ success: boolean; message: string }>(`/api/v1/devices/${deviceId}/ping-targets/${targetId}`);
  },

  // 切换 Ping 目标启用状态
  togglePingTarget(deviceId: number, targetId: number) {
    return api.put<{ success: boolean; data: { enabled: boolean }; message: string }>(`/api/v1/devices/${deviceId}/ping-targets/${targetId}/toggle`);
  },

  // 获取采集器状态
  getCollectorStatus(id: number) {
    return api.get<{ success: boolean; data: { config: CollectorStatus; device_status: { script_exists: boolean; scheduler_exists: boolean; scheduler_enabled: boolean } } }>(`/api/v1/devices/${id}/collector/status`);
  },

  // 部署采集器
  deployCollector(id: number, intervalMs: number) {
    return api.post<{ success: boolean; message: string }>(`/api/v1/devices/${id}/collector/deploy`, { interval_ms: intervalMs });
  },

  // 卸载采集器
  undeployCollector(id: number) {
    return api.delete<{ success: boolean; message: string }>(`/api/v1/devices/${id}/collector/deploy`);
  },

  // 更新采集器配置
  updateCollectorConfig(id: number, intervalMs: number, batchSize?: number) {
    return api.put<{ success: boolean; message: string }>(`/api/v1/devices/${id}/collector`, { interval_ms: intervalMs, push_batch_size: batchSize });
  },

  // 切换采集器推送状态
  toggleCollectorPush(id: number, enabled: boolean) {
    return api.post<{ success: boolean; message: string }>(`/api/v1/devices/${id}/collector/toggle`, { enabled });
  },

  // 清除设备数据
  clearDeviceData(id: number) {
    return api.post<{ success: boolean; message: string }>(`/api/v1/devices/${id}/collector/clear`);
  },
};

export default deviceApi;
