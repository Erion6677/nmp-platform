/**
 * 用户管理 API
 */

import api from './client';
import type {
  User,
  Role,
  Permission,
  CreateUserRequest,
  UpdateUserRequest,
  PaginatedResponse,
} from './types';

interface ListParams {
  page?: number;
  size?: number;
  search?: string;
}

export const userApi = {
  // 获取用户列表
  list(params?: ListParams) {
    return api.get<{ success: boolean; data: PaginatedResponse<User> & { users?: User[] } }>('/api/v1/admin/users', params as Record<string, string | number | boolean | undefined>);
  },

  // 获取用户详情
  get(id: number) {
    return api.get<{ success: boolean; data: User }>(`/api/v1/admin/users/${id}`);
  },

  // 创建用户
  create(data: CreateUserRequest) {
    return api.post<{ success: boolean; data: User }>('/api/v1/admin/users', data);
  },

  // 更新用户
  update(id: number, data: UpdateUserRequest) {
    return api.put<{ success: boolean; data: User }>(`/api/v1/admin/users/${id}`, data);
  },

  // 删除用户
  delete(id: number) {
    return api.delete<{ success: boolean; message: string }>(`/api/v1/admin/users/${id}`);
  },

  // 更新用户状态
  updateStatus(id: number, status: string) {
    return api.put<{ success: boolean; message: string }>(`/api/v1/admin/users/${id}/status`, { status });
  },

  // 更新用户角色
  updateRoles(id: number, roleIds: number[]) {
    return api.put<{ success: boolean; message: string }>(`/api/v1/admin/users/${id}/roles`, { role_ids: roleIds });
  },
};

export const roleApi = {
  // 获取角色列表
  list(params?: ListParams) {
    return api.get<{ success: boolean; data: PaginatedResponse<Role> & { roles?: Role[] } }>('/api/v1/admin/roles', params as Record<string, string | number | boolean | undefined>);
  },

  // 获取角色详情
  get(id: number) {
    return api.get<{ success: boolean; data: Role }>(`/api/v1/admin/roles/${id}`);
  },

  // 创建角色
  create(data: { name: string; display_name: string; description?: string; permission_ids?: number[] }) {
    return api.post<{ success: boolean; data: Role }>('/api/v1/admin/roles', data);
  },

  // 更新角色
  update(id: number, data: { display_name?: string; description?: string }) {
    return api.put<{ success: boolean; data: Role }>(`/api/v1/admin/roles/${id}`, data);
  },

  // 删除角色
  delete(id: number) {
    return api.delete<{ success: boolean; message: string }>(`/api/v1/admin/roles/${id}`);
  },

  // 更新角色权限
  updatePermissions(id: number, permissionIds: number[]) {
    return api.put<{ success: boolean; message: string }>(`/api/v1/admin/roles/${id}/permissions`, { permission_ids: permissionIds });
  },
};

export const permissionApi = {
  // 获取权限列表
  list(params?: ListParams) {
    return api.get<{ success: boolean; data: PaginatedResponse<Permission> & { permissions?: Permission[] } }>('/api/v1/admin/permissions', params as Record<string, string | number | boolean | undefined>);
  },
};

export default { userApi, roleApi, permissionApi };
