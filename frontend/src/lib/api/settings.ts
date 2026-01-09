/**
 * 系统设置 API
 */

import api from './client';
import type { CollectionSettings, UpdateCollectionSettingsRequest } from './types';

export const settingsApi = {
  // 获取采集设置
  getCollectionSettings() {
    return api.get<{ success: boolean; data: CollectionSettings }>('/api/v1/settings/collection');
  },

  // 更新采集设置
  updateCollectionSettings(data: UpdateCollectionSettingsRequest) {
    return api.put<{ success: boolean; data: CollectionSettings; message: string }>('/api/v1/settings/collection', data);
  },

  // 获取所有设置
  getAllSettings() {
    return api.get<{ success: boolean; data: Record<string, string> }>('/api/v1/settings/all');
  },

  // 初始化默认设置
  initDefaults() {
    return api.post<{ success: boolean; message: string }>('/api/v1/settings/init');
  },
};

export default settingsApi;
