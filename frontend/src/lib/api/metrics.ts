/**
 * 监控数据 API
 */

import api from './client';
import type {
  BandwidthQueryResponse,
  PingQueryResponse,
  TotalTrafficResponse,
  TimeRangeType,
} from './types';

export const metricsApi = {
  // 查询带宽数据
  queryBandwidth(deviceId: number | string, range: TimeRangeType, interfaces?: string[]) {
    const params: Record<string, string | number | boolean | undefined> = { range };
    if (interfaces && interfaces.length > 0) {
      params.interfaces = interfaces.join(',');
    }
    return api.get<BandwidthQueryResponse>(`/api/v1/metrics/bandwidth/${deviceId}`, params);
  },

  // 查询 Ping 数据
  queryPing(deviceId: number | string, range: TimeRangeType, target?: string) {
    const params: Record<string, string | number | boolean | undefined> = { range };
    if (target) {
      params.target = target;
    }
    return api.get<PingQueryResponse>(`/api/v1/metrics/ping/${deviceId}`, params);
  },

  // 查询总流量数据
  queryTotalTraffic(range: TimeRangeType) {
    return api.get<TotalTrafficResponse>('/api/v1/metrics/traffic/total', { range });
  },
};

export default metricsApi;
