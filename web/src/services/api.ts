import axios, { AxiosResponse } from 'axios'
import type { Alert, AlertFilters, RoutingRule, NotificationChannel, Silence, ApiResponse, PaginatedResponse, Stats } from '@/types'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

// 请求拦截器
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token')
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
api.interceptors.response.use(
  (response: AxiosResponse<ApiResponse<any>>) => {
    if (response.data.success) {
      return response
    } else {
      return Promise.reject(new Error(response.data.error?.message || 'Unknown error'))
    }
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

export const alertApi = {
  // 告警相关API
  list: (filters: AlertFilters) => 
    api.get<ApiResponse<PaginatedResponse<Alert>>>('/alerts', { params: filters }),
  
  get: (fingerprint: string) =>
    api.get<ApiResponse<Alert>>(`/alerts/${fingerprint}`),
  
  silence: (fingerprint: string, data: { duration: string; comment?: string }) =>
    api.put<ApiResponse<any>>(`/alerts/${fingerprint}/silence`, data),
  
  acknowledge: (fingerprint: string, data: { comment?: string }) =>
    api.put<ApiResponse<any>>(`/alerts/${fingerprint}/ack`, data),
  
  resolve: (fingerprint: string, data?: { comment?: string }) =>
    api.delete<ApiResponse<any>>(`/alerts/${fingerprint}`, { data }),
  
  send: (alerts: any[]) =>
    api.post<ApiResponse<any>>('/alerts', alerts),
}

export const ruleApi = {
  // 规则相关API
  list: () =>
    api.get<ApiResponse<RoutingRule[]>>('/rules'),
  
  get: (id: number) =>
    api.get<ApiResponse<RoutingRule>>(`/rules/${id}`),
  
  create: (rule: Partial<RoutingRule>) =>
    api.post<ApiResponse<RoutingRule>>('/rules', rule),
  
  update: (id: number, rule: Partial<RoutingRule>) =>
    api.put<ApiResponse<RoutingRule>>(`/rules/${id}`, rule),
  
  delete: (id: number) =>
    api.delete<ApiResponse<any>>(`/rules/${id}`),
  
  test: (data: { conditions: any; sample_alert: any }) =>
    api.post<ApiResponse<{ matched: boolean; matched_rules: RoutingRule[] }>>('/rules/test', data),
}

export const channelApi = {
  // 通知渠道相关API
  list: () =>
    api.get<ApiResponse<NotificationChannel[]>>('/channels'),
  
  get: (id: number) =>
    api.get<ApiResponse<NotificationChannel>>(`/channels/${id}`),
  
  create: (channel: Partial<NotificationChannel>) =>
    api.post<ApiResponse<NotificationChannel>>('/channels', channel),
  
  update: (id: number, channel: Partial<NotificationChannel>) =>
    api.put<ApiResponse<NotificationChannel>>(`/channels/${id}`, channel),
  
  delete: (id: number) =>
    api.delete<ApiResponse<any>>(`/channels/${id}`),
  
  test: (id: number, data: { message: string }) =>
    api.post<ApiResponse<any>>(`/channels/${id}/test`, data),
}

export const silenceApi = {
  // 静默相关API
  list: () =>
    api.get<ApiResponse<Silence[]>>('/silences'),
  
  get: (id: number) =>
    api.get<ApiResponse<Silence>>(`/silences/${id}`),
  
  create: (silence: Partial<Silence>) =>
    api.post<ApiResponse<Silence>>('/silences', silence),
  
  delete: (id: number) =>
    api.delete<ApiResponse<any>>(`/silences/${id}`),
}

export const statsApi = {
  // 统计相关API
  alerts: (params: { start_time?: string; end_time?: string; group_by?: string }) =>
    api.get<ApiResponse<Stats>>('/stats/alerts', { params }),
  
  notifications: (params: { start_time?: string; end_time?: string }) =>
    api.get<ApiResponse<any>>('/stats/notifications', { params }),
}

export const authApi = {
  // 认证相关API
  login: (data: { username: string; password: string }) =>
    api.post<ApiResponse<{ token: string; user: any }>>('/auth/login', data),
  
  logout: () =>
    api.post<ApiResponse<any>>('/auth/logout'),
  
  refresh: () =>
    api.post<ApiResponse<{ token: string }>>('/auth/refresh'),
}

export default api