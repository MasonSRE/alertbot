import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { message } from 'antd'
import { alertGroupApi } from '@/services/api'

export const useAlertGroups = (filters?: any) => {
  return useQuery({
    queryKey: ['alert-groups', filters],
    queryFn: async () => {
      try {
        const response = await alertGroupApi.listGroups(filters)
        return response.data?.data?.items || []
      } catch (error) {
        console.error('Failed to fetch alert groups:', error)
        return []
      }
    },
    refetchInterval: 30000, // 30秒自动刷新
  })
}

export const useAlertGroup = (id: number) => {
  return useQuery({
    queryKey: ['alert-group', id],
    queryFn: async () => {
      const response = await alertGroupApi.getGroup(id)
      return response.data.data
    },
    enabled: !!id,
  })
}

export const useAlertGroupRules = () => {
  return useQuery({
    queryKey: ['alert-group-rules'],
    queryFn: async () => {
      try {
        const response = await alertGroupApi.listRules()
        return response.data?.data?.items || []
      } catch (error) {
        console.error('Failed to fetch alert group rules:', error)
        return []
      }
    },
  })
}

export const useAlertGroupRule = (id: number) => {
  return useQuery({
    queryKey: ['alert-group-rule', id],
    queryFn: async () => {
      const response = await alertGroupApi.getRule(id)
      return response.data.data
    },
    enabled: !!id,
  })
}

export const useCreateAlertGroupRule = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (data: any) => alertGroupApi.createRule(data),
    onSuccess: () => {
      message.success('分组规则创建成功')
      queryClient.invalidateQueries({ queryKey: ['alert-group-rules'] })
    },
    onError: (error: any) => {
      message.error(`创建失败: ${error.message}`)
    },
  })
}

export const useUpdateAlertGroupRule = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: any }) => 
      alertGroupApi.updateRule(id, data),
    onSuccess: () => {
      message.success('分组规则更新成功')
      queryClient.invalidateQueries({ queryKey: ['alert-group-rules'] })
    },
    onError: (error: any) => {
      message.error(`更新失败: ${error.message}`)
    },
  })
}

export const useDeleteAlertGroupRule = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (id: number) => alertGroupApi.deleteRule(id),
    onSuccess: () => {
      message.success('分组规则删除成功')
      queryClient.invalidateQueries({ queryKey: ['alert-group-rules'] })
    },
    onError: (error: any) => {
      message.error(`删除失败: ${error.message}`)
    },
  })
}

export const useTestAlertGroupRule = () => {
  return useMutation({
    mutationFn: (data: any) => alertGroupApi.testRule(data),
    onSuccess: (response) => {
      const result = response.data?.data
      if (result?.matched) {
        message.success(`匹配成功！分组键: ${result.group_key}`)
      } else {
        message.warning('不匹配。此规则不会匹配提供的告警标签')
      }
    },
    onError: (error: any) => {
      message.error(`测试失败: ${error.message}`)
    },
  })
}