import React, { useState } from 'react'
import { Card, Button, Space, message, Divider } from 'antd'
import { alertApi } from '@/services/api'

const Test: React.FC = () => {
  const [loading, setLoading] = useState(false)
  
  const testHealthCheck = async () => {
    try {
      setLoading(true)
      const response = await fetch('/health')
      const data = await response.json()
      message.success(`健康检查成功: ${data.status}`)
      console.log('Health check:', data)
    } catch (error) {
      message.error(`健康检查失败: ${error}`)
    } finally {
      setLoading(false)
    }
  }

  const testSendAlert = async () => {
    try {
      setLoading(true)
      const testAlert = [{
        labels: {
          alertname: 'TestAlert',
          instance: 'test-server:9100',
          severity: 'warning',
          job: 'test',
        },
        annotations: {
          description: '这是一个测试告警',
          summary: '测试告警摘要',
        },
        startsAt: new Date().toISOString(),
        endsAt: '0001-01-01T00:00:00Z',
      }]
      
      const response = await alertApi.send(testAlert)
      message.success('测试告警发送成功')
      console.log('Send alert response:', response.data)
    } catch (error: any) {
      message.error(`发送告警失败: ${error.message}`)
      console.error('Send alert error:', error)
    } finally {
      setLoading(false)
    }
  }

  const testListAlerts = async () => {
    try {
      setLoading(true)
      const response = await alertApi.list({ page: 1, size: 10 })
      message.success(`获取告警列表成功，共 ${response.data.data.total} 条`)
      console.log('List alerts response:', response.data)
    } catch (error: any) {
      message.error(`获取告警列表失败: ${error.message}`)
      console.error('List alerts error:', error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <div>
      <Card title="API 测试工具">
        <Space direction="vertical" style={{ width: '100%' }}>
          <div>
            <h3>基础测试</h3>
            <Space>
              <Button 
                type="primary" 
                onClick={testHealthCheck}
                loading={loading}
              >
                健康检查
              </Button>
            </Space>
          </div>

          <Divider />

          <div>
            <h3>告警API测试</h3>
            <Space>
              <Button 
                onClick={testSendAlert}
                loading={loading}
              >
                发送测试告警
              </Button>
              <Button 
                onClick={testListAlerts}
                loading={loading}
              >
                获取告警列表
              </Button>
            </Space>
          </div>

          <Divider />

          <div>
            <h3>说明</h3>
            <p>1. 首先点击"健康检查"确保后端服务正常运行</p>
            <p>2. 点击"发送测试告警"创建一个测试告警</p>
            <p>3. 点击"获取告警列表"查看所有告警</p>
            <p>4. 打开浏览器开发者工具查看详细的API响应</p>
          </div>
        </Space>
      </Card>
    </div>
  )
}

export default Test