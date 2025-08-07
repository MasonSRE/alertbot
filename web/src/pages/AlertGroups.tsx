import React, { useState } from 'react'
import { Card, Table, Button, Space, Modal, Form, Input, Tag, Alert, Tabs, Select, InputNumber } from 'antd'
import { PlusOutlined, EditOutlined, DeleteOutlined, PlayCircleOutlined, EyeOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useAlertGroups, useAlertGroupRules, useCreateAlertGroupRule, useUpdateAlertGroupRule, useDeleteAlertGroupRule, useTestAlertGroupRule } from '@/hooks/useAlertGroups'
import dayjs from 'dayjs'

const { Option } = Select
const { TabPane } = Tabs
const { TextArea } = Input

interface AlertGroup {
  id: number
  group_key: string
  group_by: Record<string, any>
  common_labels: Record<string, any>
  alert_count: number
  status: string
  severity: string
  first_alert_at: string
  last_alert_at: string
  created_at: string
  updated_at: string
}

interface AlertGroupRule {
  id: number
  name: string
  description: string
  group_by: Record<string, any>
  group_wait: number
  group_interval: number
  repeat_interval: number
  matchers: Record<string, any>
  priority: number
  enabled: boolean
  created_at: string
  updated_at: string
}

const AlertGroups: React.FC = () => {
  const [activeTab, setActiveTab] = useState('groups')
  const [modalVisible, setModalVisible] = useState(false)
  const [editingRule, setEditingRule] = useState<AlertGroupRule | null>(null)
  const [testModalVisible, setTestModalVisible] = useState(false)
  const [form] = Form.useForm()
  const [testForm] = Form.useForm()

  const { data: alertGroups = [], isLoading: groupsLoading, refetch: refetchGroups } = useAlertGroups()
  const { data: groupRules = [], isLoading: rulesLoading, refetch: refetchRules } = useAlertGroupRules()
  const createRuleMutation = useCreateAlertGroupRule()
  const updateRuleMutation = useUpdateAlertGroupRule()
  const deleteRuleMutation = useDeleteAlertGroupRule()
  const testRuleMutation = useTestAlertGroupRule()

  // Alert Groups Table
  const groupColumns: ColumnsType<AlertGroup> = [
    {
      title: '分组键',
      dataIndex: 'group_key',
      key: 'group_key',
      width: 200,
      render: (key: string) => <code style={{ fontSize: '12px' }}>{key}</code>,
    },
    {
      title: '分组条件',
      dataIndex: 'common_labels',
      key: 'common_labels',
      render: (labels: Record<string, any>) => (
        <div>
          {Object.entries(labels || {}).map(([key, value]) => (
            <Tag key={key} style={{ margin: '2px' }}>
              {key}={String(value)}
            </Tag>
          ))}
        </div>
      ),
    },
    {
      title: '告警数量',
      dataIndex: 'alert_count',
      key: 'alert_count',
      width: 100,
      render: (count: number) => <strong>{count}</strong>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => (
        <Tag color={status === 'firing' ? 'red' : status === 'resolved' ? 'green' : 'gray'}>
          {status === 'firing' ? '告警中' : status === 'resolved' ? '已解决' : '其他'}
        </Tag>
      ),
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      width: 100,
      render: (severity: string) => (
        <Tag color={severity === 'critical' ? 'red' : severity === 'warning' ? 'orange' : 'blue'}>
          {severity?.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '首次告警',
      dataIndex: 'first_alert_at',
      key: 'first_alert_at',
      width: 180,
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '--',
    },
    {
      title: '最后告警',
      dataIndex: 'last_alert_at',
      key: 'last_alert_at',
      width: 180,
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '--',
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Button
            size="small"
            icon={<EyeOutlined />}
            onClick={() => handleViewGroup(record)}
          >
            查看
          </Button>
        </Space>
      ),
    },
  ]

  // Alert Group Rules Table
  const ruleColumns: ColumnsType<AlertGroupRule> = [
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
      title: '分组字段',
      dataIndex: 'group_by',
      key: 'group_by',
      render: (groupBy: Record<string, any>) => {
        const labels = groupBy?.labels || []
        return (
          <div>
            {Array.isArray(labels) ? labels.map((label: string, index: number) => (
              <Tag key={index} style={{ margin: '2px' }}>
                {label}
              </Tag>
            )) : null}
          </div>
        )
      },
    },
    {
      title: '时间配置',
      key: 'timing',
      render: (_, record) => (
        <div>
          <div>等待: {record.group_wait}s</div>
          <div>间隔: {record.group_interval}s</div>
          <div>重复: {record.repeat_interval}s</div>
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
      render: (enabled: boolean) => (
        <Tag color={enabled ? 'green' : 'red'}>
          {enabled ? '启用' : '禁用'}
        </Tag>
      ),
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
            onClick={() => handleEditRule(record)}
          >
            编辑
          </Button>
          <Button
            size="small"
            icon={<DeleteOutlined />}
            danger
            onClick={() => handleDeleteRule(record)}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ]

  const handleViewGroup = (group: AlertGroup) => {
    Modal.info({
      title: `告警分组详情 - ${group.group_key}`,
      width: 600,
      content: (
        <div style={{ marginTop: 16 }}>
          <p><strong>分组键:</strong> <code>{group.group_key}</code></p>
          <p><strong>告警数量:</strong> {group.alert_count}</p>
          <p><strong>状态:</strong> {group.status}</p>
          <p><strong>严重程度:</strong> {group.severity}</p>
          <p><strong>首次告警:</strong> {dayjs(group.first_alert_at).format('YYYY-MM-DD HH:mm:ss')}</p>
          <p><strong>最后告警:</strong> {dayjs(group.last_alert_at).format('YYYY-MM-DD HH:mm:ss')}</p>
          <p><strong>通用标签:</strong></p>
          <div>
            {Object.entries(group.common_labels || {}).map(([key, value]) => (
              <Tag key={key} style={{ margin: '2px' }}>
                {key}={String(value)}
              </Tag>
            ))}
          </div>
        </div>
      ),
    })
  }

  const handleCreateRule = () => {
    setEditingRule(null)
    form.resetFields()
    form.setFieldsValue({
      group_wait: 10,
      group_interval: 300,
      repeat_interval: 3600,
      priority: 0,
      enabled: true,
    })
    setModalVisible(true)
  }

  const handleEditRule = (rule: AlertGroupRule) => {
    setEditingRule(rule)
    form.setFieldsValue({
      ...rule,
      group_by_labels: rule.group_by?.labels || [],
    })
    setModalVisible(true)
  }

  const handleDeleteRule = (rule: AlertGroupRule) => {
    Modal.confirm({
      title: '确认删除分组规则',
      content: `确定要删除分组规则 "${rule.name}" 吗？`,
      onOk() {
        deleteRuleMutation.mutate(rule.id)
      },
    })
  }

  const handleTestRule = (rule: AlertGroupRule) => {
    testForm.resetFields()
    testForm.setFieldsValue({
      group_by: rule.group_by,
      matchers: rule.matchers,
    })
    setTestModalVisible(true)
  }

  const handleSubmit = (values: any) => {
    const ruleData = {
      name: values.name,
      description: values.description,
      group_by: {
        labels: values.group_by_labels || [],
      },
      group_wait: values.group_wait,
      group_interval: values.group_interval,
      repeat_interval: values.repeat_interval,
      matchers: values.matchers ? JSON.parse(values.matchers) : {},
      priority: values.priority,
      enabled: values.enabled,
    }
    
    if (editingRule) {
      updateRuleMutation.mutate({ id: editingRule.id, data: ruleData })
    } else {
      createRuleMutation.mutate(ruleData)
    }
    setModalVisible(false)
  }

  const handleTestSubmit = (values: any) => {
    const testData = {
      group_by: values.group_by,
      matchers: values.matchers,
      test_alert: {
        alertname: values.alertname || 'TestAlert',
        instance: values.instance || 'localhost:9100',
        severity: values.severity || 'warning',
        ...((values.custom_labels && values.custom_labels.trim()) ? JSON.parse(values.custom_labels) : {}),
      },
    }
    testRuleMutation.mutate(testData)
  }

  return (
    <div>
      <Alert
        message="告警分组说明"
        description="告警分组将相似的告警聚合在一起，减少通知噪音。分组规则定义了如何根据告警标签进行分组，以及分组后的通知策略。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />

      <Tabs activeKey={activeTab} onChange={setActiveTab}>
        <TabPane tab="告警分组" key="groups">
          <Card>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Button onClick={() => refetchGroups()}>刷新</Button>
              </Space>
            </div>

            <Table
              columns={groupColumns}
              dataSource={alertGroups}
              loading={groupsLoading}
              rowKey="id"
              pagination={{
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              }}
            />
          </Card>
        </TabPane>

        <TabPane tab="分组规则" key="rules">
          <Card>
            <div style={{ marginBottom: 16 }}>
              <Space>
                <Button type="primary" icon={<PlusOutlined />} onClick={handleCreateRule}>
                  创建分组规则
                </Button>
                <Button onClick={() => refetchRules()}>刷新</Button>
              </Space>
            </div>

            <Table
              columns={ruleColumns}
              dataSource={groupRules}
              loading={rulesLoading}
              rowKey="id"
              pagination={{
                showSizeChanger: true,
                showQuickJumper: true,
                showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
              }}
            />
          </Card>
        </TabPane>
      </Tabs>

      {/* 创建/编辑分组规则Modal */}
      <Modal
        title={editingRule ? '编辑分组规则' : '创建分组规则'}
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
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
            name="group_by_labels"
            label="分组字段"
            rules={[{ required: true, message: '请选择至少一个分组字段' }]}
          >
            <Select
              mode="tags"
              placeholder="输入标签名进行分组，如: alertname, instance, service"
              style={{ width: '100%' }}
            >
              <Option value="alertname">alertname</Option>
              <Option value="instance">instance</Option>
              <Option value="service">service</Option>
              <Option value="severity">severity</Option>
              <Option value="job">job</Option>
              <Option value="team">team</Option>
            </Select>
          </Form.Item>
          
          <div style={{ display: 'flex', gap: '16px' }}>
            <Form.Item
              name="group_wait"
              label="组等待时间 (秒)"
              rules={[{ required: true, message: '请输入组等待时间' }]}
              style={{ flex: 1 }}
            >
              <InputNumber min={0} max={3600} placeholder="10" style={{ width: '100%' }} />
            </Form.Item>
            
            <Form.Item
              name="group_interval"
              label="组间隔时间 (秒)"
              rules={[{ required: true, message: '请输入组间隔时间' }]}
              style={{ flex: 1 }}
            >
              <InputNumber min={60} max={86400} placeholder="300" style={{ width: '100%' }} />
            </Form.Item>
            
            <Form.Item
              name="repeat_interval"
              label="重复间隔 (秒)"
              rules={[{ required: true, message: '请输入重复间隔' }]}
              style={{ flex: 1 }}
            >
              <InputNumber min={300} max={604800} placeholder="3600" style={{ width: '100%' }} />
            </Form.Item>
          </div>
          
          <Form.Item name="matchers" label="匹配条件 (JSON格式)">
            <TextArea 
              rows={3} 
              placeholder='{"matchers": [{"name": "team", "value": "frontend", "is_regex": false}]}' 
            />
          </Form.Item>
          
          <div style={{ display: 'flex', gap: '16px' }}>
            <Form.Item name="priority" label="优先级" style={{ flex: 1 }}>
              <InputNumber placeholder="数值越大优先级越高" style={{ width: '100%' }} />
            </Form.Item>
            
            <Form.Item name="enabled" label="启用状态" valuePropName="checked" style={{ flex: 1 }}>
              <Select style={{ width: '100%' }}>
                <Option value={true}>启用</Option>
                <Option value={false}>禁用</Option>
              </Select>
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

      {/* 测试分组规则Modal */}
      <Modal
        title="测试分组规则"
        open={testModalVisible}
        onCancel={() => setTestModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form form={testForm} onFinish={handleTestSubmit} layout="vertical">
          <Alert
            message="测试说明"
            description="输入一组告警标签来测试分组规则的匹配效果。"
            type="info"
            style={{ marginBottom: 16 }}
          />
          
          <Form.Item name="alertname" label="告警名称">
            <Input placeholder="例如: HighCPUUsage" />
          </Form.Item>
          
          <Form.Item name="instance" label="实例">
            <Input placeholder="例如: server1:9100" />
          </Form.Item>
          
          <Form.Item name="severity" label="严重程度">
            <Select placeholder="选择严重程度">
              <Option value="critical">Critical</Option>
              <Option value="warning">Warning</Option>
              <Option value="info">Info</Option>
            </Select>
          </Form.Item>
          
          <Form.Item name="custom_labels" label="其他标签 (JSON格式)">
            <TextArea 
              rows={3} 
              placeholder='{"job": "node-exporter", "team": "sre"}' 
            />
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                测试分组
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

export default AlertGroups