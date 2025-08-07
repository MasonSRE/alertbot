import { useQuery } from '@tanstack/react-query'
import { alertApi, alertHistoryApi } from '@/services/api'

interface AlertHistoryFilters {
  alert_fingerprint?: string
  action?: string
  page?: number
  size?: number
  sort?: string
  order?: string
}

// 获取特定告警的历史记录
export const useAlertHistory = (fingerprint: string) => {
  return useQuery({
    queryKey: ['alert-history', fingerprint],
    queryFn: async () => {
      if (!fingerprint) return []
      const response = await alertApi.getHistory(fingerprint)
      return response.data?.data || []
    },
    enabled: !!fingerprint,
  })
}

// 获取所有告警历史记录（分页）
export const useAlertHistoryList = (filters: AlertHistoryFilters = {}) => {
  return useQuery({
    queryKey: ['alert-history-list', filters],
    queryFn: async () => {
      try {
        const response = await alertHistoryApi.list(filters)
        const data = response.data?.data
        return {
          items: Array.isArray(data?.items) ? data.items : [],
          total: data?.total || 0,
          page: data?.page || 1,
          size: data?.size || 50,
          pages: data?.pages || 1
        }
      } catch (error) {
        console.error('Failed to fetch alert history:', error)
        return {
          items: [],
          total: 0,
          page: 1,
          size: 50,
          pages: 1
        }
      }
    },
  })
}