import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { message } from 'antd'
import { alertApi } from '@/services/api'
import type { AlertFilters } from '@/types'

export const useAlerts = (filters: AlertFilters) => {
  return useQuery({
    queryKey: ['alerts', filters],
    queryFn: async () => {
      const response = await alertApi.list(filters)
      return response.data.data
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