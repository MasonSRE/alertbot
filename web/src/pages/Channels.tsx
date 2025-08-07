import React, { useState } from 'react'
import { Card, Table, Button, Space, Switch, Modal, Form, Input, Select, Tag, InputNumber } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, SendOutlined } from '@ant-design/icons'
import { useChannels, useCreateChannel, useUpdateChannel, useDeleteChannel, useTestChannel } from '@/hooks/useChannels'
import type { ColumnsType } from 'antd/es/table'
import type { NotificationChannel } from '@/types'

const Channels: React.FC = () => {
  const [modalVisible, setModalVisible] = useState(false)
  const [testModalVisible, setTestModalVisible] = useState(false)
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null)
  const [testingChannel, setTestingChannel] = useState<NotificationChannel | null>(null)
  const [form] = Form.useForm()
  const [testForm] = Form.useForm()
  const [selectedChannelType, setSelectedChannelType] = useState<string>('')

  const { data: channels = [], isLoading, refetch } = useChannels()
  const createMutation = useCreateChannel()
  const updateMutation = useUpdateChannel()
  const deleteMutation = useDeleteChannel()
  const testMutation = useTestChannel()

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
        if (!config || typeof config !== 'object') return null
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
      render: (time: string) => time ? new Date(time).toLocaleString() : '--',
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
    setSelectedChannelType('')
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (channel: NotificationChannel) => {
    setEditingChannel(channel)
    setSelectedChannelType(channel.type)
    form.setFieldsValue({
      name: channel.name,
      type: channel.type,
      enabled: channel.enabled,
      ...channel.config
    })
    setModalVisible(true)
  }

  const handleDelete = (channel: NotificationChannel) => {
    Modal.confirm({
      title: '确认删除渠道',
      content: `确定要删除渠道 "${channel.name}" 吗？`,
      onOk() {
        deleteMutation.mutate(channel.id)
      },
    })
  }

  const handleTest = (channel: NotificationChannel) => {
    setTestingChannel(channel)
    testForm.resetFields()
    setTestModalVisible(true)
  }

  const handleToggleEnabled = (id: number, enabled: boolean) => {
    const channel = channels.find(c => c.id === id)
    if (channel) {
      updateMutation.mutate({ id, data: { ...channel, enabled } })
    }
  }

  const handleSubmit = (values: any) => {
    const configData = { ...values }
    delete configData.name
    delete configData.type
    delete configData.enabled
    
    const channelData = {
      name: values.name,
      type: values.type,
      enabled: values.enabled ?? true,
      config: configData
    }
    
    if (editingChannel) {
      updateMutation.mutate({ 
        id: editingChannel.id, 
        data: channelData 
      })
    } else {
      createMutation.mutate(channelData)
    }
    setModalVisible(false)
  }

  const handleTestSubmit = (values: any) => {
    if (testingChannel) {
      testMutation.mutate({ 
        id: testingChannel.id, 
        message: values.message 
      })
    }
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
            <Button onClick={() => refetch()}>刷新</Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={channels}
          loading={isLoading}
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
            <Select 
              placeholder="请选择渠道类型"
              onChange={(value) => {
                setSelectedChannelType(value)
                form.resetFields(['webhook_url', 'secret', 'smtp_host', 'smtp_port', 'smtp_username', 'smtp_password', 'from', 'to', 'corp_id', 'corp_secret', 'agent_id', 'api_key', 'template_id', 'sign_name'])
              }}
            >
              {channelTypeOptions.map(option => (
                <Select.Option key={option.value} value={option.value}>
                  {option.label}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          
          {/* DingTalk配置 */}
          {selectedChannelType === 'dingtalk' && (
            <>
              <Form.Item
                name="webhook_url"
                label="Webhook URL"
                rules={[{ required: true, message: '请输入DingTalk机器人Webhook URL' }]}
              >
                <Input placeholder="https://oapi.dingtalk.com/robot/send?access_token=..." />
              </Form.Item>
              <Form.Item name="secret" label="加签密钥">
                <Input.Password placeholder="SEC..." />
              </Form.Item>
            </>
          )}

          {/* WeChat Work配置 */}
          {selectedChannelType === 'wechat_work' && (
            <>
              <Form.Item
                name="webhook_url"
                label="Webhook URL"
                rules={[{ required: true, message: '请输入企业微信机器人Webhook URL' }]}
              >
                <Input placeholder="https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=..." />
              </Form.Item>
            </>
          )}

          {/* Email配置 */}
          {selectedChannelType === 'email' && (
            <>
              <Form.Item
                name="smtp_host"
                label="SMTP服务器"
                rules={[{ required: true, message: '请输入SMTP服务器地址' }]}
              >
                <Input placeholder="smtp.gmail.com" />
              </Form.Item>
              <Form.Item
                name="smtp_port"
                label="SMTP端口"
                rules={[{ required: true, message: '请输入SMTP端口' }]}
              >
                <InputNumber placeholder="587" style={{ width: '100%' }} />
              </Form.Item>
              <Form.Item
                name="smtp_username"
                label="用户名"
                rules={[{ required: true, message: '请输入SMTP用户名' }]}
              >
                <Input placeholder="your-email@gmail.com" />
              </Form.Item>
              <Form.Item
                name="smtp_password"
                label="密码"
                rules={[{ required: true, message: '请输入SMTP密码' }]}
              >
                <Input.Password placeholder="应用专用密码" />
              </Form.Item>
              <Form.Item
                name="from"
                label="发件人"
                rules={[{ required: true, message: '请输入发件人邮箱' }]}
              >
                <Input placeholder="alerts@company.com" />
              </Form.Item>
              <Form.Item
                name="to"
                label="收件人"
                rules={[{ required: true, message: '请输入收件人邮箱' }]}
              >
                <Input placeholder="admin@company.com,ops@company.com" />
              </Form.Item>
            </>
          )}

          {/* SMS配置 */}
          {selectedChannelType === 'sms' && (
            <>
              <Form.Item
                name="api_key"
                label="API密钥"
                rules={[{ required: true, message: '请输入SMS服务商API密钥' }]}
              >
                <Input.Password placeholder="您的SMS API密钥" />
              </Form.Item>
              <Form.Item
                name="template_id"
                label="短信模板ID"
                rules={[{ required: true, message: '请输入短信模板ID' }]}
              >
                <Input placeholder="SMS_123456" />
              </Form.Item>
              <Form.Item
                name="sign_name"
                label="签名"
                rules={[{ required: true, message: '请输入短信签名' }]}
              >
                <Input placeholder="【公司名称】" />
              </Form.Item>
              <Form.Item
                name="to"
                label="接收手机号"
                rules={[{ required: true, message: '请输入接收手机号' }]}
              >
                <Input placeholder="13800138000,13900139000" />
              </Form.Item>
            </>
          )}
          
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