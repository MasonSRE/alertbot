import React, { useState } from 'react'
import { Card, Table, Button, Space, Switch, Modal, Form, Input, Select, Tag, message } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, SendOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { NotificationChannel } from '@/types'

const Channels: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [testModalVisible, setTestModalVisible] = useState(false)
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null)
  const [testingChannel, setTestingChannel] = useState<NotificationChannel | null>(null)
  const [form] = Form.useForm()
  const [testForm] = Form.useForm()

  // 模拟数据
  const channels: NotificationChannel[] = [
    {
      id: 1,
      name: 'DingTalk On-Call',
      type: 'dingtalk',
      config: {
        webhook_url: 'https://oapi.dingtalk.com/robot/send?access_token=***',
        secret: '***',
      },
      enabled: true,
      created_at: '2025-08-05T08:00:00Z',
      updated_at: '2025-08-05T08:00:00Z',
    },
    {
      id: 2,
      name: 'Email Notifications',
      type: 'email',
      config: {
        smtp_host: 'smtp.gmail.com',
        smtp_port: 587,
        from: 'alerts@company.com',
        to: ['admin@company.com'],
      },
      enabled: true,
      created_at: '2025-08-05T08:00:00Z',
      updated_at: '2025-08-05T08:00:00Z',
    },
  ]

  const channelTypeOptions = [
    { label: '钉钉', value: 'dingtalk' },
    { label: '企业微信', value: 'wechat_work' },
    { label: '邮件', value: 'email' },
    { label: '短信', value: 'sms' },
  ]

  const getChannelTypeLabel = (type: string) => {
    const option = channelTypeOptions.find(opt => opt.value === type)
    return option?.label || type
  }

  const getChannelTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      dingtalk: 'blue',
      wechat_work: 'green',
      email: 'orange',
      sms: 'purple',
    }
    return colors[type] || 'default'
  }

  const columns: ColumnsType<NotificationChannel> = [
    {
      title: '渠道名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => <strong>{name}</strong>,
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type: string) => (
        <Tag color={getChannelTypeColor(type)}>
          {getChannelTypeLabel(type)}
        </Tag>
      ),
    },
    {
      title: '配置',
      dataIndex: 'config',
      key: 'config',
      ellipsis: true,
      render: (config: any) => {
        const keys = Object.keys(config).slice(0, 2)
        return keys.map(key => (
          <Tag key={key} style={{ margin: '2px' }}>
            {key}
          </Tag>
        ))
      },
    },
    {
      title: '状态',
      dataIndex: 'enabled',
      key: 'enabled',
      width: 80,
      render: (enabled: boolean, record) => (
        <Switch
          checked={enabled}
          onChange={(checked) => handleToggleEnabled(record.id, checked)}
        />
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 150,
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button
            size="small"
            icon={<SendOutlined />}
            onClick={() => handleTest(record)}
          >
            测试
          </Button>
          <Button
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Button
            size="small"
            icon={<DeleteOutlined />}
            danger
            onClick={() => handleDelete(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ]

  const handleCreate = () => {
    setEditingChannel(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (channel: NotificationChannel) => {
    setEditingChannel(channel)
    form.setFieldsValue({
      name: channel.name,
      type: channel.type,
      enabled: channel.enabled,
    })
    setModalVisible(true)
  }

  const handleDelete = (channel: NotificationChannel) => {
    Modal.confirm({
      title: '确认删除渠道',
      content: `确定要删除渠道 "${channel.name}" 吗？`,
      onOk() {
        message.success(`渠道 ${channel.name} 已删除`)
      },
    })
  }

  const handleTest = (channel: NotificationChannel) => {
    setTestingChannel(channel)
    testForm.resetFields()
    setTestModalVisible(true)
  }

  const handleToggleEnabled = (id: number, enabled: boolean) => {
    message.success(`渠道状态已${enabled ? '启用' : '禁用'}`)
  }

  const handleSubmit = (values: any) => {
    if (editingChannel) {
      message.success(`渠道 ${values.name} 已更新`)
    } else {
      message.success(`渠道 ${values.name} 已创建`)
    }
    setModalVisible(false)
  }

  const handleTestSubmit = (values: any) => {
    message.success(`测试消息已发送到 ${testingChannel?.name}`)
    setTestModalVisible(false)
  }

  return (
    <div>
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              创建渠道
            </Button>
            <Button onClick={() => setLoading(true)}>刷新</Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={channels}
          loading={loading}
          rowKey="id"
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
        />
      </Card>

      <Modal
        title={editingChannel ? '编辑渠道' : '创建渠道'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form form={form} onFinish={handleSubmit} layout="vertical">
          <Form.Item
            name="name"
            label="渠道名称"
            rules={[{ required: true, message: '请输入渠道名称' }]}
          >
            <Input placeholder="请输入渠道名称" />
          </Form.Item>
          
          <Form.Item
            name="type"
            label="渠道类型"
            rules={[{ required: true, message: '请选择渠道类型' }]}
          >
            <Select placeholder="请选择渠道类型">
              {channelTypeOptions.map(option => (
                <Select.Option key={option.value} value={option.value}>
                  {option.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          
          <Form.Item name="enabled" label="启用状态" valuePropName="checked">
            <Switch />
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingChannel ? '更新' : '创建'}
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      <Modal
        title={`测试渠道 - ${testingChannel?.name}`}
        open={testModalVisible}
        onCancel={() => setTestModalVisible(false)}
        footer={null}
      >
        <Form form={testForm} onFinish={handleTestSubmit} layout="vertical">
          <Form.Item
            name="message"
            label="测试消息"
            rules={[{ required: true, message: '请输入测试消息' }]}
          >
            <Input.TextArea
              rows={4}
              placeholder="请输入要发送的测试消息内容"
              defaultValue="这是来自 AlertBot 的测试消息"
            />
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                发送测试
              </Button>
              <Button onClick={() => setTestModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Channels