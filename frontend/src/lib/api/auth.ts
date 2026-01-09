/**
 * 认证 API
 */

import { api, setToken, clearToken } from './client';
import type { LoginParams, LoginResult, UserInfo } from './types';

export const authApi = {
  // 登录
  async login(params: LoginParams) {
    const response = await api.post<{ success: boolean; data: LoginResult }>('/api/v1/auth/login', params);
    if (response.success && response.data?.token) {
      setToken(response.data.token);
    }
    return response;
  },

  // 登出
  async logout() {
    try {
      await api.post<{ success: boolean }>('/api/v1/auth/logout');
    } finally {
      clearToken();
    }
  },

  // 获取当前用户信息
  async getCurrentUser() {
    return api.get<{ success: boolean; data: UserInfo }>('/api/v1/auth/me');
  },

  // 刷新 Token
  async refreshToken() {
    const response = await api.post<{ success: boolean; data: { token: string } }>('/api/v1/auth/refresh-token');
    if (response.success && response.data?.token) {
      setToken(response.data.token);
    }
    return response;
  },

  // 修改密码
  async changePassword(oldPassword: string, newPassword: string) {
    return api.post<{ success: boolean; message: string }>('/api/v1/auth/change-password', {
      oldPassword,
      newPassword,
    });
  },
};

export default authApi;
