import React, { useState, useEffect } from 'react'
import { Card, Form, Input, Switch, Button, Space, message, Tabs, Divider, Alert, InputNumber, Select } from 'antd'
import { SaveOutlined, ReloadOutlined, LinkOutlined } from '@ant-design/icons'
import { settingsApi } from '@/services/api'

const { TabPane } = Tabs

const Settings: React.FC = () => {
  const [form] = Form.useForm()
  const [prometheusForm] = Form.useForm()
  const [loading, setLoading] = useState(false)
  const [testingConnection, setTestingConnection] = useState(false)

  const handleSave = async (values: any) => {
    setLoading(true)
    try {
      await settingsApi.updateSystemSettings(values)
      message.success('系统设置已保存')
    } catch (error: any) {
      message.error(`保存失败: ${error.message}`)
    } finally {
      setLoading(false)
    }
  }

  const handlePrometheusSubmit = async (values: any) => {
    setLoading(true)
    try {
      await settingsApi.updatePrometheusSettings(values)
      message.success('Prometheus 配置已保存')
    } catch (error: any) {
      message.error(`保存失败: ${error.message}`)
    } finally {
      setLoading(false)
    }
  }

  const testPrometheusConnection = async () => {
    const values = prometheusForm.getFieldsValue()
    if (!values.url) {
      message.error('请先输入 Prometheus URL')
      return
    }
    
    setTestingConnection(true)
    try {
      await settingsApi.testPrometheusConnection({
        url: values.url,
        timeout: values.timeout || 30
      })
      message.success('Prometheus 连接测试成功')
    } catch (error: any) {
      message.error(`连接测试失败: ${error.message}`)
    } finally {
      setTestingConnection(false)
    }
  }

  const loadSettings = async () => {
    try {
      // 加载系统设置
      const systemResponse = await settingsApi.getSystemSettings()
      form.setFieldsValue(systemResponse.data.data)
      
      // 加载Prometheus设置
      const prometheusResponse = await settingsApi.getPrometheusSettings()
      prometheusForm.setFieldsValue(prometheusResponse.data.data)
    } catch (error) {
      // 如果加载失败，使用默认值
      prometheusForm.setFieldsValue({
        enabled: true,
        url: 'http://localhost:9090',
        timeout: 30,
        query_timeout: 30,
        scrape_interval: '15s',
        evaluation_interval: '15s'
      })
    }
  }

  useEffect(() => {
    loadSettings()
  }, [])

  return (
    <div>
      <Tabs defaultActiveKey="system" type="card">
        <TabPane tab="系统设置" key="system">
          <Card>
            <Form
              form={form}
              layout="vertical"
              onFinish={handleSave}
              initialValues={{
                system_name: 'AlertBot',
                admin_email: 'admin@company.com',
                retention_days: 30,
                enable_notifications: true,
                enable_webhooks: true,
                webhook_timeout: 30,
              }}
            >
              <Form.Item
                name="system_name"
                label="系统名称"
                rules={[{ required: true, message: '请输入系统名称' }]}
              >
                <Input placeholder="请输入系统名称" />
              </Form.Item>

              <Form.Item
                name="admin_email"
                label="管理员邮箱"
                rules={[
                  { required: true, message: '请输入管理员邮箱' },
                  { type: 'email', message: '请输入有效的邮箱地址' },
                ]}
              >
                <Input placeholder="请输入管理员邮箱" />
              </Form.Item>

              <Form.Item
                name="retention_days"
                label="数据保留天数"
                rules={[{ required: true, message: '请输入数据保留天数' }]}
              >
                <InputNumber min={1} max={365} style={{ width: '100%' }} placeholder="默认30天" />
              </Form.Item>

              <Form.Item name="enable_notifications" label="启用通知" valuePropName="checked">
                <Switch />
              </Form.Item>

              <Form.Item name="enable_webhooks" label="启用Webhook" valuePropName="checked">
                <Switch />
              </Form.Item>

              <Form.Item
                name="webhook_timeout"
                label="Webhook超时时间(秒)"
              >
                <InputNumber min={1} max={300} style={{ width: '100%' }} placeholder="默认30秒" />
              </Form.Item>

              <Form.Item>
                <Space>
                  <Button type="primary" htmlType="submit" icon={<SaveOutlined />}>
                    保存设置
                  </Button>
                  <Button onClick={() => form.resetFields()} icon={<ReloadOutlined />}>
                    重置
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        </TabPane>
        
        <TabPane tab="Prometheus 配置" key="prometheus">
          <Card>
            <Alert
              message="Prometheus 集成说明"
              description="配置 Prometheus 服务器地址，用于接收告警和查询指标数据。AlertBot 作为 Prometheus Alertmanager 的替代品，需要与 Prometheus 服务器连接。"
              type="info"
              showIcon
              style={{ marginBottom: 24 }}
            />
            
            <Form
              form={prometheusForm}
              layout="vertical"
              onFinish={handlePrometheusSubmit}
              initialValues={{
                enabled: true,
                url: 'http://localhost:9090',
                timeout: 30,
                query_timeout: 30,
                scrape_interval: '15s',
                evaluation_interval: '15s'
              }}
            >
              <Form.Item 
                name="enabled" 
                label="启用 Prometheus 集成" 
                valuePropName="checked"
                extra="启用后可以接收 Prometheus 的告警通知"
              >
                <Switch />
              </Form.Item>
              
              <Form.Item
                name="url"
                label="Prometheus 服务器 URL"
                rules={[
                  { required: true, message: '请输入 Prometheus URL' },
                  { type: 'url', message: '请输入正确的 URL 格式' }
                ]}
                extra="例如：http://localhost:9090 或 http://prometheus.company.com:9090"
              >
                <Input placeholder="http://localhost:9090" />
              </Form.Item>
              
              <Form.Item
                name="timeout"
                label="连接超时时间（秒）"
                rules={[{ required: true, message: '请输入超时时间' }]}
              >
                <InputNumber min={5} max={300} style={{ width: '100%' }} />
              </Form.Item>
              
              <Form.Item
                name="query_timeout"
                label="查询超时时间（秒）"
                rules={[{ required: true, message: '请输入查询超时时间' }]}
              >
                <InputNumber min={5} max={300} style={{ width: '100%' }} />
              </Form.Item>
              
              <Divider orientation="left">高级设置</Divider>
              
              <Form.Item
                name="scrape_interval"
                label="采集间隔"
                extra="Prometheus 采集数据的默认间隔时间"
              >
                <Select>
                  <Select.Option value="5s">5秒</Select.Option>
                  <Select.Option value="10s">10秒</Select.Option>
                  <Select.Option value="15s">15秒</Select.Option>
                  <Select.Option value="30s">30秒</Select.Option>
                  <Select.Option value="1m">1分钟</Select.Option>
                </Select>
              </Form.Item>
              
              <Form.Item
                name="evaluation_interval"
                label="规则评估间隔"
                extra="Prometheus 评估告警规则的默认间隔时间"
              >
                <Select>
                  <Select.Option value="5s">5秒</Select.Option>
                  <Select.Option value="10s">10秒</Select.Option>
                  <Select.Option value="15s">15秒</Select.Option>
                  <Select.Option value="30s">30秒</Select.Option>
                  <Select.Option value="1m">1分钟</Select.Option>
                </Select>
              </Form.Item>
              
              <Form.Item>
                <Space>
                  <Button 
                    type="primary" 
                    htmlType="submit" 
                    icon={<SaveOutlined />}
                    loading={loading}
                  >
                    保存配置
                  </Button>
                  <Button 
                    icon={<LinkOutlined />}
                    onClick={testPrometheusConnection}
                    loading={testingConnection}
                  >
                    测试连接
                  </Button>
                  <Button 
                    icon={<ReloadOutlined />}
                    onClick={() => prometheusForm.resetFields()}
                  >
                    重置
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        </TabPane>
        
        <TabPane tab="通知设置" key="notification">
          <Card>
            <Alert
              message="全局通知设置"
              description="设置全局的通知默认参数，包括重试策略、限流设置等。"
              type="info"
              showIcon
              style={{ marginBottom: 24 }}
            />
            
            <Form layout="vertical" onFinish={handleSave}>
              <Form.Item name="max_retries" label="最大重试次数" initialValue={3}>
                <InputNumber min={0} max={10} style={{ width: '100%' }} />
              </Form.Item>
              
              <Form.Item name="retry_interval" label="重试间隔（秒）" initialValue={30}>
                <InputNumber min={1} max={3600} style={{ width: '100%' }} />
              </Form.Item>
              
              <Form.Item name="rate_limit" label="限流设置（每分钟最大发送数）" initialValue={100}>
                <InputNumber min={1} max={1000} style={{ width: '100%' }} />
              </Form.Item>
              
              <Form.Item name="batch_size" label="批量发送大小" initialValue={10}>
                <InputNumber min={1} max={100} style={{ width: '100%' }} />
              </Form.Item>
              
              <Form.Item>
                <Space>
                  <Button type="primary" htmlType="submit" icon={<SaveOutlined />}>
                    保存设置
                  </Button>
                  <Button icon={<ReloadOutlined />}>
                    重置
                  </Button>
                </Space>
              </Form.Item>
            </Form>
          </Card>
        </TabPane>
      </Tabs>
    </div>
  )
}

export default Settings