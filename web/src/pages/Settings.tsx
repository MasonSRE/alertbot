import React from 'react'
import { Card, Form, Input, Switch, Button, Space, message } from 'antd'

const Settings: React.FC = () => {
  const [form] = Form.useForm()

  const handleSave = (values: any) => {
    message.success('设置已保存')
    console.log('Settings saved:', values)
  }

  return (
    <div>
      <Card title="系统设置">
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
            <Input type="number" placeholder="默认30天" />
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
            <Input type="number" placeholder="默认30秒" />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                保存设置
              </Button>
              <Button onClick={() => form.resetFields()}>
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}

export default Settings