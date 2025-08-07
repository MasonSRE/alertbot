import React, { useState } from 'react'
import { Card, Table, Button, Space, Switch, Modal, Form, Input, Tag, Alert } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons'
import { useRules, useCreateRule, useUpdateRule, useDeleteRule, useTestRule } from '@/hooks/useRules'
import type { ColumnsType } from 'antd/es/table'
import type { RoutingRule } from '@/types'

const Rules: React.FC = () => {
  const [modalVisible, setModalVisible] = useState(false)
  const [editingRule, setEditingRule] = useState<RoutingRule | null>(null)
  const [form] = Form.useForm()

  const { data: rules = [], isLoading, refetch } = useRules()
  const createMutation = useCreateRule()
  const updateMutation = useUpdateRule()
  const deleteMutation = useDeleteRule()
  const testMutation = useTestRule()

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
          {conditions && typeof conditions === 'object' ? 
            Object.entries(conditions).map(([key, value]) => (
              <Tag key={key} style={{ margin: '2px' }}>
                {key}: {Array.isArray(value) ? value.join(', ') : String(value)}
              </Tag>
            )) : null
          }
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
        deleteMutation.mutate(rule.id)
      },
    })
  }

  const handleTestRule = (rule: RoutingRule) => {
    // 创建一个测试告警示例
    const sampleAlert = {
      labels: {
        alertname: 'TestAlert',
        instance: 'localhost:9100',
        job: 'node-exporter',
        severity: 'warning'
      },
      annotations: {
        description: '这是一个测试告警',
        summary: '测试告警摘要'
      }
    }
    
    testMutation.mutate({
      conditions: rule.conditions,
      sample_alert: sampleAlert
    })
  }

  const handleToggleEnabled = (ruleId: number, enabled: boolean) => {
    const rule = rules.find(r => r.id === ruleId)
    if (rule) {
      updateMutation.mutate({ id: ruleId, data: { ...rule, enabled } })
    }
  }

  const handleSubmit = (values: any) => {
    const ruleData = {
      name: values.name,
      description: values.description,
      conditions: { severity: values.severity }, // 简化的条件，实际上应该支持更复杂的条件
      receivers: { channels: [] },
      priority: values.priority,
      enabled: values.enabled ?? true
    }
    
    if (editingRule) {
      updateMutation.mutate({ id: editingRule.id, data: ruleData })
    } else {
      createMutation.mutate(ruleData)
    }
    setModalVisible(false)
  }

  return (
    <div>
      <Alert
        message="告警路由规则说明"
        description="路由规则用于将来自 Prometheus 的告警按照条件匹配，自动路由到相应的通知渠道。与 Prometheus 的 alerting rules 不同，这里的规则是用于 AlertBot 系统内部的路由逻辑。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              创建规则
            </Button>
            <Button onClick={() => refetch()}>刷新</Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={rules}
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