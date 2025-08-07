export interface Alert {
  id: number
  fingerprint: string
  labels: Record<string, string>
  annotations: Record<string, string>
  status: 'firing' | 'resolved' | 'silenced' | 'acknowledged'
  severity: 'critical' | 'warning' | 'info'
  starts_at: string
  ends_at?: string
  created_at: string
  updated_at: string
}

export interface AlertFilters {
  status?: string
  severity?: string
  alertname?: string
  instance?: string
  page?: number
  size?: number
  sort?: string
  order?: string
}

export interface RoutingRule {
  id: number
  name: string
  description: string
  conditions: Record<string, any>
  receivers: {
    channels: number[]
  }
  priority: number
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface NotificationChannel {
  id: number
  name: string
  type: 'dingtalk' | 'wechat_work' | 'email' | 'sms'
  config: Record<string, any>
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface Silence {
  id: number
  matchers: Array<{
    name: string
    value: string
    is_regex: boolean
  }>
  starts_at: string
  ends_at: string
  creator: string
  comment: string
  created_at: string
}

export interface ApiResponse<T> {
  success: boolean
  data: T
  message?: string
  error?: {
    code: string
    message: string
    details?: any
  }
}

export interface PaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  size: number
  pages: number
}

export interface Stats {
  total_alerts: number
  firing_alerts: number
  resolved_alerts: number
  groups: Array<{
    key: string
    count: number
    percentage: number
  }>
  timeline: Array<{
    timestamp: string
    count: number
  }>
}