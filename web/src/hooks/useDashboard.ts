import { useQuery } from '@tanstack/react-query'
import { statsApi } from '@/services/api'

export const useDashboardStats = () => {
  return useQuery({
    queryKey: ['dashboard-stats'],
    queryFn: async () => {
      try {
        const response = await statsApi.alerts({})
        const data = response.data?.data
        return {
          total_alerts: data?.total_alerts || 0,
          firing_alerts: data?.firing_alerts || 0,
          resolved_alerts: data?.resolved_alerts || 0,
          groups: Array.isArray(data?.groups) ? data.groups : [],
          timeline: Array.isArray(data?.timeline) ? data.timeline : []
        }
      } catch (error) {
        console.error('Failed to fetch dashboard stats:', error)
        return {
          total_alerts: 0,
          firing_alerts: 0,
          resolved_alerts: 0,
          groups: [],
          timeline: []
        }
      }
    },
    refetchInterval: 30000, // 30秒自动刷新
  })
}

export const useSystemStats = () => {
  return useQuery({
    queryKey: ['system-stats'],
    queryFn: async () => {
      // 使用健康检查接口获取系统状态
      const response = await fetch('/api/v1/health')
      const data = await response.json()
      return data.data
    },
    refetchInterval: 60000, // 1分钟刷新
  })
}