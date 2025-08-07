import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { message } from 'antd'
import { inhibitionApi } from '@/services/api'

// 获取抑制规则列表
export const useInhibitions = () => {
  return useQuery({
    queryKey: ['inhibitions'],
    queryFn: async () => {
      const response = await inhibitionApi.list()
      return response.data?.data || []
    },
  })
}

// 获取单个抑制规则
export const useInhibition = (id: number) => {
  return useQuery({
    queryKey: ['inhibition', id],
    queryFn: async () => {
      const response = await inhibitionApi.get(id)
      return response.data.data
    },
    enabled: !!id,
  })
}

// 创建抑制规则
export const useCreateInhibition = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (rule: any) => inhibitionApi.create(rule),
    onSuccess: () => {
      message.success('抑制规则创建成功')
      queryClient.invalidateQueries({ queryKey: ['inhibitions'] })
    },
    onError: (error: any) => {
      message.error(`创建失败: ${error.message}`)
    },
  })
}

// 更新抑制规则
export const useUpdateInhibition = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: ({ id, data }: { id: number; data: any }) =>
      inhibitionApi.update(id, data),
    onSuccess: () => {
      message.success('抑制规则更新成功')
      queryClient.invalidateQueries({ queryKey: ['inhibitions'] })
    },
    onError: (error: any) => {
      message.error(`更新失败: ${error.message}`)
    },
  })
}

// 删除抑制规则
export const useDeleteInhibition = () => {
  const queryClient = useQueryClient()
  
  return useMutation({
    mutationFn: (id: number) => inhibitionApi.delete(id),
    onSuccess: () => {
      message.success('抑制规则删除成功')
      queryClient.invalidateQueries({ queryKey: ['inhibitions'] })
    },
    onError: (error: any) => {
      message.error(`删除失败: ${error.message}`)
    },
  })
}

// 测试抑制规则
export const useTestInhibition = () => {
  return useMutation({
    mutationFn: (data: { rule: any; source_alert: any; target_alert: any }) =>
      inhibitionApi.test(data),
    onSuccess: (data) => {
      const result = data?.data?.data
      if (result?.inhibited) {
        message.success(`测试通过：规则 "${result.test_rule}" 将抑制目标告警`)
      } else {
        message.info(`测试结果：规则 "${result?.test_rule}" 不会抑制目标告警`)
      }
    },
    onError: (error: any) => {
      message.error(`测试失败: ${error.message}`)
    },
  })
}