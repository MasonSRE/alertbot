import React, { useState } from 'react'
import { Card, Table, Button, Space, Modal, Form, Input, Tag, Alert, Switch, InputNumber } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useInhibitions, useCreateInhibition, useUpdateInhibition, useDeleteInhibition, useTestInhibition } from '@/hooks/useInhibitions'
import dayjs from 'dayjs'

const { TextArea } = Input

interface InhibitionRule {
  id: number
  name: string
  description: string
  source_matchers: any
  target_matchers: any
  equal_labels: any
  duration: number
  priority: number
  enabled: boolean
  created_at: string
  updated_at: string
}

const Inhibitions: React.FC = () => {
  const [modalVisible, setModalVisible] = useState(false)
  const [testModalVisible, setTestModalVisible] = useState(false)
  const [editingRule, setEditingRule] = useState<InhibitionRule | null>(null)
  const [form] = Form.useForm()
  const [testForm] = Form.useForm()

  const { data: rules = [], isLoading, refetch } = useInhibitions()
  const createMutation = useCreateInhibition()
  const updateMutation = useUpdateInhibition()
  const deleteMutation = useDeleteInhibition()
  const testMutation = useTestInhibition()

  const columns: ColumnsType<InhibitionRule> = [
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
      title: '源匹配器',
      dataIndex: 'source_matchers',
      key: 'source_matchers',
      render: (matchers: any) => (
        <div>
          {Array.isArray(matchers?.matchers) ? matchers.matchers.map((matcher: any, index: number) => (
            <Tag key={index} style={{ margin: '2px' }}>
              {matcher.name}{matcher.is_regex ? '~' : '='}{matcher.value}
            </Tag>
          )) : <Tag color="gray">未配置</Tag>}
        </div>
      ),
    },
    {
      title: '目标匹配器',
      dataIndex: 'target_matchers',
      key: 'target_matchers',
      render: (matchers: any) => (
        <div>
          {Array.isArray(matchers?.matchers) ? matchers.matchers.map((matcher: any, index: number) => (
            <Tag key={index} style={{ margin: '2px' }}>
              {matcher.name}{matcher.is_regex ? '~' : '='}{matcher.value}
            </Tag>
          )) : <Tag color="gray">未配置</Tag>}
        </div>
      ),
    },
    {
      title: '持续时间',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: (duration: number) => `${duration}s`,
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
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'red'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => dayjs(time).format('YYYY-MM-DD HH:mm:ss'),
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
    setEditingRule(null)
    form.resetFields()
    form.setFieldsValue({
      duration: 300,
      priority: 0,
      enabled: true,
    })
    setModalVisible(true)
  }

  const handleEdit = (rule: InhibitionRule) => {
    setEditingRule(rule)
    form.setFieldsValue({
      ...rule,
      source_matchers: JSON.stringify(rule.source_matchers || {}, null, 2),
      target_matchers: JSON.stringify(rule.target_matchers || {}, null, 2),
      equal_labels: JSON.stringify(rule.equal_labels || [], null, 2),
    })
    setModalVisible(true)
  }

  const handleDelete = (rule: InhibitionRule) => {
    Modal.confirm({
      title: '确认删除抑制规则',
      content: `确定要删除抑制规则 "${rule.name}" 吗？`,
      onOk() {
        deleteMutation.mutate(rule.id)
      },
    })
  }

  const handleTest = (rule: InhibitionRule) => {
    testForm.resetFields()
    testForm.setFieldsValue({
      rule_name: rule.name,
    })
    setTestModalVisible(true)
  }

  const handleSubmit = (values: any) => {
    try {
      const ruleData = {
        name: values.name,
        description: values.description,
        source_matchers: values.source_matchers ? JSON.parse(values.source_matchers) : {},
        target_matchers: values.target_matchers ? JSON.parse(values.target_matchers) : {},
        equal_labels: values.equal_labels ? JSON.parse(values.equal_labels) : [],
        duration: values.duration,
        priority: values.priority,
        enabled: values.enabled,
      }

      if (editingRule) {
        updateMutation.mutate({ id: editingRule.id, data: ruleData })
      } else {
        createMutation.mutate(ruleData)
      }
      setModalVisible(false)
    } catch (error) {
      Modal.error({
        title: 'JSON 格式错误',
        content: '请检查匹配器和相等标签的JSON格式是否正确',
      })
    }
  }

  const handleTestSubmit = (values: any) => {
    try {
      const testData = {
        rule: {
          name: values.rule_name,
          source_matchers: JSON.parse(values.source_matchers || '{}'),
          target_matchers: JSON.parse(values.target_matchers || '{}'),
          equal_labels: JSON.parse(values.equal_labels || '[]'),
        },
        source_alert: JSON.parse(values.source_alert || '{}'),
        target_alert: JSON.parse(values.target_alert || '{}'),
      }
      testMutation.mutate(testData)
    } catch (error) {
      Modal.error({
        title: 'JSON 格式错误',
        content: '请检查输入的JSON格式是否正确',
      })
    }
  }

  return (
    <div>
      <Alert
        message="抑制规则说明"
        description="抑制规则用于在某些告警存在时抑制其他相关告警，减少告警噪音。源匹配器定义触发抑制的告警，目标匹配器定义被抑制的告警。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Card>
        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              创建抑制规则
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

      {/* 创建/编辑抑制规则Modal */}
      <Modal
        title={editingRule ? '编辑抑制规则' : '创建抑制规则'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
        destroyOnClose
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
            <TextArea rows={2} placeholder="请输入规则描述" />
          </Form.Item>

          <Form.Item
            name="source_matchers"
            label="源匹配器 (JSON格式)"
            rules={[{ required: true, message: '请输入源匹配器' }]}
          >
            <TextArea
              rows={4}
              placeholder='{"matchers": [{"name": "alertname", "value": "InstanceDown", "is_regex": false}]}'
            />
          </Form.Item>

          <Form.Item
            name="target_matchers"
            label="目标匹配器 (JSON格式)"
            rules={[{ required: true, message: '请输入目标匹配器' }]}
          >
            <TextArea
              rows={4}
              placeholder='{"matchers": [{"name": "severity", "value": "warning", "is_regex": false}]}'
            />
          </Form.Item>

          <Form.Item name="equal_labels" label="相等标签 (JSON数组)">
            <TextArea
              rows={2}
              placeholder='["instance", "job"]'
            />
          </Form.Item>

          <div style={{ display: 'flex', gap: '16px' }}>
            <Form.Item
              name="duration"
              label="持续时间 (秒)"
              style={{ flex: 1 }}
              rules={[{ required: true, message: '请输入持续时间' }]}
            >
              <InputNumber min={0} style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item name="priority" label="优先级" style={{ flex: 1 }}>
              <InputNumber style={{ width: '100%' }} />
            </Form.Item>

            <Form.Item name="enabled" label="启用状态" valuePropName="checked" style={{ flex: 1 }}>
              <Switch />
            </Form.Item>
          </div>

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

      {/* 测试抑制规则Modal */}
      <Modal
        title="测试抑制规则"
        open={testModalVisible}
        onCancel={() => setTestModalVisible(false)}
        footer={null}
        width={700}
        destroyOnClose
      >
        <Form form={testForm} onFinish={handleTestSubmit} layout="vertical">
          <Alert
            message="测试说明"
            description="输入源告警和目标告警的标签来测试抑制规则是否生效。"
            type="info"
            style={{ marginBottom: 16 }}
          />

          <Form.Item name="rule_name" label="规则名称">
            <Input disabled />
          </Form.Item>

          <Form.Item
            name="source_matchers"
            label="源匹配器 (JSON)"
            rules={[{ required: true, message: '请输入源匹配器' }]}
          >
            <TextArea
              rows={3}
              placeholder='{"matchers": [{"name": "alertname", "value": "InstanceDown", "is_regex": false}]}'
            />
          </Form.Item>

          <Form.Item
            name="target_matchers"
            label="目标匹配器 (JSON)"
            rules={[{ required: true, message: '请输入目标匹配器' }]}
          >
            <TextArea
              rows={3}
              placeholder='{"matchers": [{"name": "severity", "value": "warning", "is_regex": false}]}'
            />
          </Form.Item>

          <Form.Item
            name="source_alert"
            label="源告警标签 (JSON)"
            rules={[{ required: true, message: '请输入源告警标签' }]}
          >
            <TextArea
              rows={3}
              placeholder='{"alertname": "InstanceDown", "instance": "localhost:9100", "job": "node"}'
            />
          </Form.Item>

          <Form.Item
            name="target_alert"
            label="目标告警标签 (JSON)"
            rules={[{ required: true, message: '请输入目标告警标签' }]}
          >
            <TextArea
              rows={3}
              placeholder='{"alertname": "HighCPU", "instance": "localhost:9100", "severity": "warning"}'
            />
          </Form.Item>

          <Form.Item name="equal_labels" label="相等标签 (JSON数组)">
            <TextArea
              rows={2}
              placeholder='["instance", "job"]'
            />
          </Form.Item>

          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit" loading={testMutation.isPending}>
                测试规则
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

export default Inhibitions