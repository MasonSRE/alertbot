import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { message } from 'antd'
import { ruleApi } from '@/services/api'
import type { RoutingRule } from '@/types'

export const useRules = () => {
  return useQuery({
    queryKey: ['rules'],
    queryFn: async () => {
      try {
        const response = await ruleApi.list()
        return response.data?.data?.items || []
      } catch (error) {
        console.error('Failed to fetch rules:', error)
        return []
      }
    },
  })
}

export const useRule = (id: number) => {
  return useQuery({
    queryKey: ['rule', id],
    queryFn: async () => {
      const response = await ruleApi.get(id)
      return response.data.data
    },
    enabled: !!id,
  })
}

export const useCreateRule = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (data: Partial<RoutingRule>) =>
      ruleApi.create(data),
    onSuccess: () => {
      message.success('路由规则已创建')
      queryClient.invalidateQueries({ queryKey: ['rules'] })
    },
    onError: (error: any) => {
      message.error(`创建失败: ${error.message}`)
    },
  })
}

export const useUpdateRule = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: Partial<RoutingRule> }) =>
      ruleApi.update(id, data),
    onSuccess: () => {
      message.success('路由规则已更新')
      queryClient.invalidateQueries({ queryKey: ['rules'] })
    },
    onError: (error: any) => {
      message.error(`更新失败: ${error.message}`)
    },
  })
}

export const useDeleteRule = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (id: number) => ruleApi.delete(id),
    onSuccess: () => {
      message.success('路由规则已删除')
      queryClient.invalidateQueries({ queryKey: ['rules'] })
    },
    onError: (error: any) => {
      message.error(`删除失败: ${error.message}`)
    },
  })
}

export const useTestRule = () => {
  return useMutation({
    mutationFn: (data: { conditions: any; sample_alert: any }) =>
      ruleApi.test(data),
    onSuccess: (response) => {
      const result = response.data.data
      if (result.matched) {
        message.success(`规则匹配成功，匹配到 ${result.matched_rules.length} 个规则`)
      } else {
        message.warning('规则不匹配')
      }
    },
    onError: (error: any) => {
      message.error(`测试失败: ${error.message}`)
    },
  })
}