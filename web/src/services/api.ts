import axios, { AxiosResponse } from 'axios'
import type { Alert, AlertFilters, RoutingRule, NotificationChannel, Silence, ApiResponse, PaginatedResponse, Stats } from '@/types'

const api = axios.create({
  baseURL: 'http://localhost:8080/api/v1',  // Direct to backend
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
  
  // 批量操作API
  batchSilence: (data: { fingerprints: string[]; duration: string; comment?: string }) =>
    api.put<ApiResponse<{ processed: number; action: string }>>('/alerts/batch/silence', data),
  
  batchAcknowledge: (data: { fingerprints: string[]; comment?: string }) =>
    api.put<ApiResponse<{ processed: number; action: string }>>('/alerts/batch/ack', data),
  
  batchResolve: (data: { fingerprints: string[]; comment?: string }) =>
    api.delete<ApiResponse<{ processed: number; action: string }>>('/alerts/batch/resolve', { data }),
  
  // 告警历史API
  getHistory: (fingerprint: string) =>
    api.get<ApiResponse<any[]>>(`/alerts/${fingerprint}/history`),
}

export const alertHistoryApi = {
  // 告警历史相关API
  list: (filters: any) =>
    api.get<ApiResponse<PaginatedResponse<any>>>('/alert-history', { params: filters }),
}

export const ruleApi = {
  // 规则相关API
  list: () =>
    api.get<ApiResponse<PaginatedResponse<RoutingRule>>>('/rules'),
  
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
    api.get<ApiResponse<PaginatedResponse<NotificationChannel>>>('/channels'),
  
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
    
  test: (data: { matchers: any[]; labels: Record<string, string> }) =>
    api.post<ApiResponse<{ matched: boolean }>>('/silences/test', data),
}

export const alertGroupApi = {
  // 告警分组API
  listGroups: (filters?: any) =>
    api.get<ApiResponse<any>>('/alert-groups', { params: filters }),
  
  getGroup: (id: number) =>
    api.get<ApiResponse<any>>(`/alert-groups/${id}`),
  
  // 告警分组规则API
  listRules: () =>
    api.get<ApiResponse<any>>('/alert-group-rules'),
  
  getRule: (id: number) =>
    api.get<ApiResponse<any>>(`/alert-group-rules/${id}`),
  
  createRule: (data: any) =>
    api.post<ApiResponse<any>>('/alert-group-rules', data),
  
  updateRule: (id: number, data: any) =>
    api.put<ApiResponse<any>>(`/alert-group-rules/${id}`, data),
  
  deleteRule: (id: number) =>
    api.delete<ApiResponse<any>>(`/alert-group-rules/${id}`),
  
  testRule: (data: any) =>
    api.post<ApiResponse<any>>('/alert-group-rules/test', data),
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

export const inhibitionApi = {
  // 抑制规则相关API
  list: () =>
    api.get<ApiResponse<any[]>>('/inhibitions'),
  
  get: (id: number) =>
    api.get<ApiResponse<any>>(`/inhibitions/${id}`),
  
  create: (rule: any) =>
    api.post<ApiResponse<any>>('/inhibitions', rule),
  
  update: (id: number, rule: any) =>
    api.put<ApiResponse<any>>(`/inhibitions/${id}`, rule),
  
  delete: (id: number) =>
    api.delete<ApiResponse<any>>(`/inhibitions/${id}`),
  
  test: (data: { rule: any; source_alert: any; target_alert: any }) =>
    api.post<ApiResponse<{ inhibited: boolean; test_rule: string }>>('/inhibitions/test', data),
}

export const settingsApi = {
  // 系统设置API
  getSystemSettings: () =>
    api.get<ApiResponse<any>>('/settings/system'),
  
  updateSystemSettings: (data: any) =>
    api.put<ApiResponse<any>>('/settings/system', data),
  
  // Prometheus设置API
  getPrometheusSettings: () =>
    api.get<ApiResponse<any>>('/settings/prometheus'),
  
  updatePrometheusSettings: (data: any) =>
    api.put<ApiResponse<any>>('/settings/prometheus', data),
  
  testPrometheusConnection: (data: { url: string; timeout: number }) =>
    api.post<ApiResponse<any>>('/settings/prometheus/test', data),
  
  // 通知设置API
  getNotificationSettings: () =>
    api.get<ApiResponse<any>>('/settings/notification'),
  
  updateNotificationSettings: (data: any) =>
    api.put<ApiResponse<any>>('/settings/notification', data),
}

export default api