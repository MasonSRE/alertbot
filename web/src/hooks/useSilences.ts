import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { message } from 'antd'
import { silenceApi } from '@/services/api'
import type { Silence } from '@/types'

export const useSilences = () => {
  return useQuery({
    queryKey: ['silences'],
    queryFn: async () => {
      try {
        const response = await silenceApi.list()
        return response.data?.data || []
      } catch (error) {
        console.error('Failed to fetch silences:', error)
        return []
      }
    },
    refetchInterval: 60000, // 60秒自动刷新
  })
}

export const useSilence = (id: number) => {
  return useQuery({
    queryKey: ['silence', id],
    queryFn: async () => {
      const response = await silenceApi.get(id)
      return response.data.data
    },
    enabled: !!id,
  })
}

export const useCreateSilence = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (data: Partial<Silence>) => silenceApi.create(data),
    onSuccess: () => {
      message.success('静默规则创建成功')
      queryClient.invalidateQueries({ queryKey: ['silences'] })
    },
    onError: (error: any) => {
      message.error(`创建失败: ${error.message}`)
    },
  })
}

export const useDeleteSilence = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (id: number) => silenceApi.delete(id),
    onSuccess: () => {
      message.success('静默规则删除成功')
      queryClient.invalidateQueries({ queryKey: ['silences'] })
    },
    onError: (error: any) => {
      message.error(`删除失败: ${error.message}`)
    },
  })
}

export const useTestSilence = () => {
  return useMutation({
    mutationFn: (data: { matchers: any[]; labels: Record<string, string> }) =>
      silenceApi.test(data),
    onSuccess: (response) => {
      const matched = response.data?.data?.matched
      if (matched) {
        message.success('匹配成功！此静默规则将匹配提供的标签')
      } else {
        message.warning('不匹配。此静默规则不会匹配提供的标签')
      }
    },
    onError: (error: any) => {
      message.error(`测试失败: ${error.message}`)
    },
  })
}