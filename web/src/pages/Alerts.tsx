import React, { useState } from 'react'
import { Card, Table, Tag, Button, Space, Select, Input, DatePicker, Modal, Form, message } from 'antd'
import { SearchOutlined, ReloadOutlined, StopOutlined, CheckOutlined, DeleteOutlined } from '@ant-design/icons'
import { useAlerts, useSilenceAlert, useAcknowledgeAlert, useResolveAlert } from '@/hooks/useAlerts'
import type { ColumnsType } from 'antd/es/table'
import type { Alert, AlertFilters } from '@/types'

const { Option } = Select
const { RangePicker } = DatePicker

const Alerts: React.FC = () => {
  const [filters, setFilters] = useState<AlertFilters>({ page: 1, size: 20 })
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [silenceModalVisible, setSilenceModalVisible] = useState(false)
  const [currentAlert, setCurrentAlert] = useState<Alert | null>(null)

  const { data: alertsData, isLoading: loading, refetch } = useAlerts(filters)
  const silenceMutation = useSilenceAlert()
  const acknowledgeMutation = useAcknowledgeAlert()
  const resolveMutation = useResolveAlert()

  const alerts = alertsData?.items || []
  const total = alertsData?.total || 0
  const page = alertsData?.page || 1
  const size = alertsData?.size || 20

  // 注释掉模拟数据
  /*const alerts: Alert[] = [
    {
      id: 1,
      fingerprint: 'f1a2b3c4d5e6f7g8',
      labels: {
        alertname: 'HighCPUUsage',
        instance: 'server1:9100',
        severity: 'critical',
        job: 'node-exporter',
      },
      annotations: {
        description: 'CPU usage is above 90% for more than 5 minutes',
        summary: 'High CPU usage detected',
      },
      status: 'firing',
      severity: 'critical',
      starts_at: '2025-08-05T10:30:00Z',
      created_at: '2025-08-05T10:30:15Z',
      updated_at: '2025-08-05T10:35:20Z',
    },
    {
      id: 2,
      fingerprint: 'a1b2c3d4e5f6g7h8',
      labels: {
        alertname: 'DiskSpaceLow',
        instance: 'server2:9100',
        severity: 'warning',
        job: 'node-exporter',
      },
      annotations: {
        description: 'Disk space is below 10%',
        summary: 'Low disk space',
      },
      status: 'firing',
      severity: 'warning',
      starts_at: '2025-08-05T10:25:00Z',
      created_at: '2025-08-05T10:25:15Z',
      updated_at: '2025-08-05T10:30:20Z',
    },
  ]*/

  const columns: ColumnsType<Alert> = [
    {
      title: '告警名称',
      dataIndex: ['labels', 'alertname'],
      key: 'alertname',
      render: (alertname: string) => <strong>{alertname}</strong>,
    },
    {
      title: '实例',
      dataIndex: ['labels', 'instance'],
      key: 'instance',
    },
    {
      title: '严重程度',
      dataIndex: 'severity',
      key: 'severity',
      render: (severity: string) => (
        <Tag color={severity === 'critical' ? 'red' : severity === 'warning' ? 'orange' : 'blue'}>
          {severity.toUpperCase()}
        </Tag>
      ),
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: string) => (
        <Tag color={status === 'firing' ? 'red' : status === 'resolved' ? 'green' : 'gray'}>
          {status === 'firing' ? '告警中' : status === 'resolved' ? '已解决' : '已静默'}
        </Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: ['annotations', 'summary'],
      key: 'summary',
      ellipsis: true,
    },
    {
      title: '开始时间',
      dataIndex: 'starts_at',
      key: 'starts_at',
      render: (time: string) => new Date(time).toLocaleString(),
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space size="small">
          <Button
            size="small"
            icon={<StopOutlined />}
            onClick={() => handleSilence(record)}
          >
            静默
          </Button>
          <Button
            size="small"
            icon={<CheckOutlined />}
            onClick={() => handleAcknowledge(record)}
          >
            确认
          </Button>
          <Button
            size="small"
            icon={<DeleteOutlined />}
            danger
            onClick={() => handleResolve(record)}
          >
            解决
          </Button>
        </Space>
      ),
    },
  ]

  const handleSilence = (alert: Alert) => {
    setCurrentAlert(alert)
    setSilenceModalVisible(true)
  }

  const handleAcknowledge = (alert: Alert) => {
    acknowledgeMutation.mutate({
      fingerprint: alert.fingerprint,
      data: { comment: '手动确认' }
    })
  }

  const handleResolve = (alert: Alert) => {
    Modal.confirm({
      title: '确认解决告警',
      content: `确定要解决告警 "${alert.labels.alertname}" 吗？`,
      onOk() {
        resolveMutation.mutate({
          fingerprint: alert.fingerprint,
          data: { comment: '手动解决' }
        })
      },
    })
  }

  const handleSilenceSubmit = (values: any) => {
    if (currentAlert) {
      silenceMutation.mutate({
        fingerprint: currentAlert.fingerprint,
        data: {
          duration: values.duration,
          comment: values.comment || '手动静默'
        }
      })
      setSilenceModalVisible(false)
    }
  }

  const rowSelection = {
    selectedRowKeys,
    onChange: (selectedRowKeys: React.Key[]) => {
      setSelectedRowKeys(selectedRowKeys)
    },
  }

  return (
    <div>
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Space wrap>
            <Input
              placeholder="搜索告警名称"
              prefix={<SearchOutlined />}
              style={{ width: 200 }}
            />
            <Select placeholder="状态" style={{ width: 120 }}>
              <Option value="">全部</Option>
              <Option value="firing">告警中</Option>
              <Option value="resolved">已解决</Option>
              <Option value="silenced">已静默</Option>
            </Select>
            <Select placeholder="严重程度" style={{ width: 120 }}>
              <Option value="">全部</Option>
              <Option value="critical">严重</Option>
              <Option value="warning">警告</Option>
              <Option value="info">信息</Option>
            </Select>
            <RangePicker placeholder={['开始时间', '结束时间']} />
            <Button type="primary" icon={<SearchOutlined />}>
              搜索
            </Button>
            <Button icon={<ReloadOutlined />} onClick={() => refetch()}>
              刷新
            </Button>
          </Space>
        </div>

        {selectedRowKeys.length > 0 && (
          <div style={{ marginBottom: 16 }}>
            <Space>
              <span>已选择 {selectedRowKeys.length} 项</span>
              <Button size="small">批量静默</Button>
              <Button size="small">批量确认</Button>
              <Button size="small" danger>批量解决</Button>
            </Space>
          </div>
        )}

        <Table
          rowSelection={rowSelection}
          columns={columns}
          dataSource={alerts}
          loading={loading}
          rowKey="id"
          pagination={{
            current: page,
            pageSize: size,
            total: total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
            onChange: (page, pageSize) => {
              setFilters({ ...filters, page, size: pageSize })
            },
          }}
        />
      </Card>

      <Modal
        title="静默告警"
        open={silenceModalVisible}
        onCancel={() => setSilenceModalVisible(false)}
        footer={null}
      >
        <Form onFinish={handleSilenceSubmit} layout="vertical">
          <Form.Item
            name="duration"
            label="静默时长"
            rules={[{ required: true, message: '请选择静默时长' }]}
          >
            <Select>
              <Option value="1h">1小时</Option>
              <Option value="4h">4小时</Option>
              <Option value="24h">24小时</Option>
              <Option value="7d">7天</Option>
            </Select>
          </Form.Item>
          <Form.Item name="comment" label="备注">
            <Input.TextArea rows={3} placeholder="请输入静默原因" />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                确认静默
              </Button>
              <Button onClick={() => setSilenceModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}

export default Alerts