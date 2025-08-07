import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { message } from 'antd'
import { channelApi } from '@/services/api'
import type { NotificationChannel } from '@/types'

export const useChannels = () => {
  return useQuery({
    queryKey: ['channels'],
    queryFn: async () => {
      try {
        const response = await channelApi.list()
        return response.data?.data?.items || []
      } catch (error) {
        console.error('Failed to fetch channels:', error)
        return []
      }
    },
  })
}

export const useChannel = (id: number) => {
  return useQuery({
    queryKey: ['channel', id],
    queryFn: async () => {
      const response = await channelApi.get(id)
      return response.data.data
    },
    enabled: !!id,
  })
}

export const useCreateChannel = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (data: Partial<NotificationChannel>) =>
      channelApi.create(data),
    onSuccess: () => {
      message.success('通知渠道已创建')
      queryClient.invalidateQueries({ queryKey: ['channels'] })
    },
    onError: (error: any) => {
      message.error(`创建失败: ${error.message}`)
    },
  })
}

export const useUpdateChannel = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<NotificationChannel> }) =>
      channelApi.update(id, data),
    onSuccess: () => {
      message.success('通知渠道已更新')
      queryClient.invalidateQueries({ queryKey: ['channels'] })
    },
    onError: (error: any) => {
      message.error(`更新失败: ${error.message}`)
    },
  })
}

export const useDeleteChannel = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (id: number) => channelApi.delete(id),
    onSuccess: () => {
      message.success('通知渠道已删除')
      queryClient.invalidateQueries({ queryKey: ['channels'] })
    },
    onError: (error: any) => {
      message.error(`删除失败: ${error.message}`)
    },
  })
}

export const useTestChannel = () => {
  return useMutation({
    mutationFn: ({ id, message }: { id: number; message: string }) =>
      channelApi.test(id, { message }),
    onSuccess: () => {
      message.success('测试消息已发送')
    },
    onError: (error: any) => {
      message.error(`测试失败: ${error.message}`)
    },
  })
}