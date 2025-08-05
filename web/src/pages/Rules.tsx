import React, { useState } from 'react'
import { Card, Table, Button, Space, Switch, Modal, Form, Input, Select, Tag, message } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { RoutingRule } from '@/types'

const Rules: React.FC = () => {
  const [loading, setLoading] = useState(false)
  const [modalVisible, setModalVisible] = useState(false)
  const [editingRule, setEditingRule] = useState<RoutingRule | null>(null)
  const [form] = Form.useForm()

  // 模拟数据
  const rules: RoutingRule[] = [
    {
      id: 1,
      name: 'Critical Alerts',
      description: '严重告警路由规则',
      conditions: { severity: 'critical' },
      receivers: [{ channel_id: 1, template: 'critical_template' }],
      priority: 100,
      enabled: true,
      created_at: '2025-08-05T09:00:00Z',
      updated_at: '2025-08-05T09:00:00Z',
    },
    {
      id: 2,
      name: 'Database Alerts',
      description: '数据库告警路由规则',
      conditions: { job: 'mysql-exporter', severity: ['warning', 'critical'] },
      receivers: [{ channel_id: 2, template: 'database_template' }],
      priority: 80,
      enabled: true,
      created_at: '2025-08-05T09:00:00Z',
      updated_at: '2025-08-05T09:00:00Z',
    },
  ]

  const columns: ColumnsType<RoutingRule> = [
    {
      title: '规则名称',
      dataIndex: 'name',
      key: 'name',
      render: (name: string) => <strong>{name}</strong>,
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '条件',
      dataIndex: 'conditions',
      key: 'conditions',
      render: (conditions: any) => (
        <div>
          {Object.entries(conditions).map(([key, value]) => (
            <Tag key={key} style={{ margin: '2px' }}>
              {key}: {Array.isArray(value) ? value.join(', ') : String(value)}
            </Tag>
          ))}
        </div>
      ),
    },
    {
      title: '优先级',
      dataIndex: 'priority',
      key: 'priority',
      width: 80,
      sorter: (a, b) => a.priority - b.priority,
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
            icon={<PlayCircleOutlined />}
            onClick={() => handleTestRule(record)}
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
    setEditingRule(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleEdit = (rule: RoutingRule) => {
    setEditingRule(rule)
    form.setFieldsValue({
      name: rule.name,
      description: rule.description,
      priority: rule.priority,
      enabled: rule.enabled,
    })
    setModalVisible(true)
  }

  const handleDelete = (rule: RoutingRule) => {
    Modal.confirm({
      title: '确认删除规则',
      content: `确定要删除规则 "${rule.name}" 吗？`,
      onOk() {
        message.success(`规则 ${rule.name} 已删除`)
      },
    })
  }

  const handleTestRule = (rule: RoutingRule) => {
    message.info(`正在测试规则 ${rule.name}...`)
  }

  const handleToggleEnabled = (id: number, enabled: boolean) => {
    message.success(`规则状态已${enabled ? '启用' : '禁用'}`)
  }

  const handleSubmit = (values: any) => {
    if (editingRule) {
      message.success(`规则 ${values.name} 已更新`)
    } else {
      message.success(`规则 ${values.name} 已创建`)
    }
    setModalVisible(false)
  }

  return (
    <div>
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              创建规则
            </Button>
            <Button onClick={() => setLoading(true)}>刷新</Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={rules}
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
        title={editingRule ? '编辑规则' : '创建规则'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form form={form} onFinish={handleSubmit} layout="vertical">
          <Form.Item
            name="name"
            label="规则名称"
            rules={[{ required: true, message: '请输入规则名称' }]}
          >
            <Input placeholder="请输入规则名称" />
          </Form.Item>
          
          <Form.Item name="description" label="描述">
            <Input.TextArea rows={2} placeholder="请输入规则描述" />
          </Form.Item>
          
          <Form.Item
            name="priority"
            label="优先级"
            rules={[{ required: true, message: '请输入优先级' }]}
          >
            <Input type="number" placeholder="数值越大优先级越高" />
          </Form.Item>
          
          <Form.Item name="enabled" label="启用状态" valuePropName="checked">
            <Switch />
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                {editingRule ? '更新' : '创建'}
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Rules