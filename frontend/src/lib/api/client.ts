/**
 * API 客户端 - 基于 fetch 的请求封装
 */

// API 响应类型
export interface ApiResponse<T = unknown> {
  success: boolean;
  data?: T;
  error?: string;
  message?: string;
}

// 请求配置
interface RequestConfig extends RequestInit {
  params?: Record<string, string | number | boolean | undefined>;
}

// API 基础 URL
const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || '';

// Token 存储 key
const TOKEN_KEY = 'nmp_token';

// 获取 Token
export function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem(TOKEN_KEY);
}

// 设置 Token
export function setToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem(TOKEN_KEY, token);
}

// 清除 Token
export function clearToken(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem(TOKEN_KEY);
}

// 构建 URL 带查询参数
function buildUrl(url: string, params?: Record<string, string | number | boolean | undefined>): string {
  const fullUrl = url.startsWith('http') ? url : `${API_BASE_URL}${url}`;
  
  if (!params) return fullUrl;
  
  const searchParams = new URLSearchParams();
  Object.entries(params).forEach(([key, value]) => {
    if (value !== undefined && value !== null) {
      searchParams.append(key, String(value));
    }
  });
  
  const queryString = searchParams.toString();
  return queryString ? `${fullUrl}?${queryString}` : fullUrl;
}

// 基础请求函数
async function request<T>(url: string, config: RequestConfig = {}): Promise<T> {
  const { params, ...fetchConfig } = config;
  
  const token = getToken();
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
    ...(config.headers || {}),
  };
  
  if (token) {
    (headers as Record<string, string>)['Authorization'] = `Bearer ${token}`;
  }
  
  try {
    const response = await fetch(buildUrl(url, params), {
      ...fetchConfig,
      headers,
      credentials: 'include',
    });
    
    // 处理 401 未授权
    if (response.status === 401) {
      clearToken();
      if (typeof window !== 'undefined' && !url.includes('/auth/login')) {
        window.location.href = '/login';
      }
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || '未授权，请重新登录');
    }
    
    // 解析响应
    const data = await response.json();
    
    // 处理其他错误状态
    if (!response.ok) {
      throw new Error(data.error || data.message || `请求失败 (${response.status})`);
    }
    
    // 处理业务错误
    if (data.success === false) {
      throw new Error(data.error || data.message || '请求失败');
    }
    
    return data;
  } catch (error) {
    if (error instanceof TypeError && error.message.includes('fetch')) {
      throw new Error('网络连接失败，请检查后端服务是否启动');
    }
    throw error;
  }
}

// 导出请求方法
export const api = {
  get<T>(url: string, params?: Record<string, string | number | boolean | undefined>): Promise<T> {
    return request<T>(url, { method: 'GET', params });
  },
  
  post<T>(url: string, data?: unknown): Promise<T> {
    return request<T>(url, {
      method: 'POST',
      body: data ? JSON.stringify(data) : undefined,
    });
  },
  
  put<T>(url: string, data?: unknown): Promise<T> {
    return request<T>(url, {
      method: 'PUT',
      body: data ? JSON.stringify(data) : undefined,
    });
  },
  
  delete<T>(url: string): Promise<T> {
    return request<T>(url, { method: 'DELETE' });
  },
  
  patch<T>(url: string, data?: unknown): Promise<T> {
    return request<T>(url, {
      method: 'PATCH',
      body: data ? JSON.stringify(data) : undefined,
    });
  },
};

export default api;
