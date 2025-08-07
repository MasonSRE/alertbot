import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { message } from 'antd'
import { alertApi } from '@/services/api'
import type { AlertFilters } from '@/types'

export const useAlerts = (filters: AlertFilters) => {
  return useQuery({
    queryKey: ['alerts', filters],
    queryFn: async () => {
      try {
        const response = await alertApi.list(filters)
        const data = response.data?.data
        // 确保返回的数据有正确的结构
        return {
          items: Array.isArray(data?.items) ? data.items : [],
          total: data?.total || 0,
          page: data?.page || 1,
          size: data?.size || 20,
          pages: data?.pages || 1
        }
      } catch (error) {
        console.error('Failed to fetch alerts:', error)
        return {
          items: [],
          total: 0,
          page: 1,
          size: 20,
          pages: 1
        }
      }
    },
    refetchInterval: 30000, // 30秒自动刷新
  })
}

export const useAlert = (fingerprint: string) => {
  return useQuery({
    queryKey: ['alert', fingerprint],
    queryFn: async () => {
      const response = await alertApi.get(fingerprint)
      return response.data.data
    },
    enabled: !!fingerprint,
  })
}

export const useSilenceAlert = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ fingerprint, data }: { fingerprint: string; data: { duration: string; comment?: string } }) =>
      alertApi.silence(fingerprint, data),
    onSuccess: () => {
      message.success('告警已静默')
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
    onError: (error: any) => {
      message.error(`静默失败: ${error.message}`)
    },
  })
}

export const useAcknowledgeAlert = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ fingerprint, data }: { fingerprint: string; data: { comment?: string } }) =>
      alertApi.acknowledge(fingerprint, data),
    onSuccess: () => {
      message.success('告警已确认')
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
    onError: (error: any) => {
      message.error(`确认失败: ${error.message}`)
    },
  })
}

export const useResolveAlert = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ fingerprint, data }: { fingerprint: string; data?: { comment?: string } }) =>
      alertApi.resolve(fingerprint, data),
    onSuccess: () => {
      message.success('告警已解决')
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
    onError: (error: any) => {
      message.error(`解决失败: ${error.message}`)
    },
  })
}

// 批量操作 hooks
export const useBatchSilenceAlerts = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ fingerprints, duration, comment }: { fingerprints: string[]; duration: string; comment?: string }) =>
      alertApi.batchSilence({ fingerprints, duration, comment }),
    onSuccess: (data) => {
      message.success(`批量静默成功，处理了 ${data?.data?.data?.processed || 0} 个告警`)
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
    onError: (error: any) => {
      message.error(`批量静默失败: ${error.message}`)
    },
  })
}

export const useBatchAcknowledgeAlerts = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ fingerprints, comment }: { fingerprints: string[]; comment?: string }) =>
      alertApi.batchAcknowledge({ fingerprints, comment }),
    onSuccess: (data) => {
      message.success(`批量确认成功，处理了 ${data?.data?.data?.processed || 0} 个告警`)
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
    onError: (error: any) => {
      message.error(`批量确认失败: ${error.message}`)
    },
  })
}

export const useBatchResolveAlerts = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ fingerprints, comment }: { fingerprints: string[]; comment?: string }) =>
      alertApi.batchResolve({ fingerprints, comment }),
    onSuccess: (data) => {
      message.success(`批量解决成功，处理了 ${data?.data?.data?.processed || 0} 个告警`)
      queryClient.invalidateQueries({ queryKey: ['alerts'] })
    },
    onError: (error: any) => {
      message.error(`批量解决失败: ${error.message}`)
    },
  })
}