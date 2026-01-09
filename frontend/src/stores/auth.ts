/**
 * 认证状态管理
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { authApi } from '@/lib/api/auth';
import { getToken, clearToken } from '@/lib/api/client';
import type { UserInfo } from '@/lib/api/types';

interface AuthState {
  user: UserInfo | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  
  // Actions
  login: (username: string, password: string) => Promise<boolean>;
  logout: () => Promise<void>;
  fetchUser: () => Promise<void>;
  clearError: () => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      login: async (username: string, password: string) => {
        set({ isLoading: true, error: null });
        try {
          const response = await authApi.login({ username, password });
          if (response.success && response.data) {
            // 后端返回的是 user 字段
            const user = response.data.user || response.data.userInfo || null;
            set({
              user,
              isAuthenticated: true,
              isLoading: false,
            });
            return true;
          }
          set({ isLoading: false, error: '登录失败：服务器响应异常' });
          return false;
        } catch (error) {
          const message = error instanceof Error ? error.message : '登录失败';
          console.error('Login error:', error);
          set({ isLoading: false, error: message });
          return false;
        }
      },

      logout: async () => {
        try {
          await authApi.logout();
        } catch {
          // 忽略登出错误
        } finally {
          clearToken();
          set({ user: null, isAuthenticated: false });
        }
      },

      fetchUser: async () => {
        const token = getToken();
        if (!token) {
          set({ user: null, isAuthenticated: false });
          return;
        }

        set({ isLoading: true });
        try {
          const response = await authApi.getCurrentUser();
          if (response.success && response.data) {
            set({
              user: response.data,
              isAuthenticated: true,
              isLoading: false,
            });
          } else {
            clearToken();
            set({ user: null, isAuthenticated: false, isLoading: false });
          }
        } catch {
          clearToken();
          set({ user: null, isAuthenticated: false, isLoading: false });
        }
      },

      clearError: () => set({ error: null }),
    }),
    {
      name: 'auth-storage',
      partialize: (state) => ({
        user: state.user,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
);

export default useAuthStore;
