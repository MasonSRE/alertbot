import React, { useState } from 'react'
import { Card, Table, Tag, Button, Space, Select, Input, DatePicker, Modal, Form } from 'antd'
import { SearchOutlined, ReloadOutlined, StopOutlined, CheckOutlined, DeleteOutlined, HistoryOutlined } from '@ant-design/icons'
import { useAlerts, useSilenceAlert, useAcknowledgeAlert, useResolveAlert, useBatchSilenceAlerts, useBatchAcknowledgeAlerts, useBatchResolveAlerts } from '@/hooks/useAlerts'
import { useAlertHistory } from '@/hooks/useAlertHistory'
import type { ColumnsType } from 'antd/es/table'
import type { Alert, AlertFilters } from '@/types'

const { Option } = Select
const { RangePicker } = DatePicker

const Alerts: React.FC = () => {
  const [filters, setFilters] = useState<AlertFilters>({ page: 1, size: 20 })
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [silenceModalVisible, setSilenceModalVisible] = useState(false)
  const [historyModalVisible, setHistoryModalVisible] = useState(false)
  const [currentAlert, setCurrentAlert] = useState<Alert | null>(null)

  const { data: alertsData, isLoading: loading, refetch } = useAlerts(filters)
  const silenceMutation = useSilenceAlert()
  const acknowledgeMutation = useAcknowledgeAlert()
  const resolveMutation = useResolveAlert()
  const batchSilenceMutation = useBatchSilenceAlerts()
  const batchAcknowledgeMutation = useBatchAcknowledgeAlerts()
  const batchResolveMutation = useBatchResolveAlerts()
  
  // 历史查看相关
  const { data: alertHistory, isLoading: historyLoading } = useAlertHistory(currentAlert?.fingerprint || '')

  const alerts = Array.isArray(alertsData?.items) ? alertsData.items : []
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
            icon={<HistoryOutlined />}
            onClick={() => handleViewHistory(record)}
          >
            历史
          </Button>
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

  const handleViewHistory = (alert: Alert) => {
    setCurrentAlert(alert)
    setHistoryModalVisible(true)
  }

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

  // 批量操作处理函数
  const getSelectedFingerprints = () => {
    return selectedRowKeys.map(key => {
      const alert = alerts.find(alert => alert.id?.toString() === key.toString())
      return alert?.fingerprint
    }).filter(Boolean) as string[]
  }

  const handleBatchSilence = () => {
    const fingerprints = getSelectedFingerprints()
    if (fingerprints.length === 0) {
      Modal.warning({ title: '提示', content: '请选择要静默的告警' })
      return
    }

    Modal.confirm({
      title: '批量静默告警',
      content: `确定要静默 ${fingerprints.length} 个告警吗？`,
      onOk() {
        Modal.confirm({
          title: '选择静默时长',
          content: (
            <Form
              onFinish={(values) => {
                batchSilenceMutation.mutate({
                  fingerprints,
                  duration: values.duration || '1h',
                  comment: values.comment || '批量静默操作'
                })
                setSelectedRowKeys([])
              }}
              layout="vertical"
            >
              <Form.Item name="duration" label="静默时长" initialValue="1h">
                <Select>
                  <Option value="1h">1小时</Option>
                  <Option value="4h">4小时</Option>
                  <Option value="24h">24小时</Option>
                  <Option value="7d">7天</Option>
                </Select>
              </Form.Item>
              <Form.Item name="comment" label="备注">
                <Input.TextArea rows={2} placeholder="批量静默原因" />
              </Form.Item>
              <Form.Item>
                <Button type="primary" htmlType="submit">确认静默</Button>
              </Form.Item>
            </Form>
          ),
          width: 500,
          footer: null
        })
      },
    })
  }

  const handleBatchAcknowledge = () => {
    const fingerprints = getSelectedFingerprints()
    if (fingerprints.length === 0) {
      Modal.warning({ title: '提示', content: '请选择要确认的告警' })
      return
    }

    Modal.confirm({
      title: '批量确认告警',
      content: `确定要确认 ${fingerprints.length} 个告警吗？`,
      onOk() {
        batchAcknowledgeMutation.mutate({
          fingerprints,
          comment: '批量确认操作'
        })
        setSelectedRowKeys([])
      },
    })
  }

  const handleBatchResolve = () => {
    const fingerprints = getSelectedFingerprints()
    if (fingerprints.length === 0) {
      Modal.warning({ title: '提示', content: '请选择要解决的告警' })
      return
    }

    Modal.confirm({
      title: '批量解决告警',
      content: `确定要解决 ${fingerprints.length} 个告警吗？这将标记这些告警为已解决状态。`,
      okText: '确认解决',
      okType: 'danger',
      onOk() {
        batchResolveMutation.mutate({
          fingerprints,
          comment: '批量解决操作'
        })
        setSelectedRowKeys([])
      },
    })
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
              value={filters.alertname || ''}
              onChange={(e) => setFilters({ ...filters, alertname: e.target.value, page: 1 })}
            />
            <Select 
              placeholder="状态" 
              style={{ width: 120 }}
              value={filters.status || ''}
              onChange={(value) => setFilters({ ...filters, status: value, page: 1 })}
            >
              <Option value="">全部</Option>
              <Option value="firing">告警中</Option>
              <Option value="resolved">已解决</Option>
              <Option value="silenced">已静默</Option>
            </Select>
            <Select 
              placeholder="严重程度" 
              style={{ width: 120 }}
              value={filters.severity || ''}
              onChange={(value) => setFilters({ ...filters, severity: value, page: 1 })}
            >
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
              <Button 
                size="small" 
                icon={<StopOutlined />}
                onClick={handleBatchSilence}
                loading={batchSilenceMutation.isPending}
              >
                批量静默
              </Button>
              <Button 
                size="small" 
                icon={<CheckOutlined />}
                onClick={handleBatchAcknowledge}
                loading={batchAcknowledgeMutation.isPending}
              >
                批量确认
              </Button>
              <Button 
                size="small" 
                icon={<DeleteOutlined />}
                danger
                onClick={handleBatchResolve}
                loading={batchResolveMutation.isPending}
              >
                批量解决
              </Button>
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

      {/* 告警历史查看Modal */}
      <Modal
        title={`告警历史 - ${currentAlert?.labels?.alertname || '未知告警'}`}
        open={historyModalVisible}
        onCancel={() => setHistoryModalVisible(false)}
        footer={[
          <Button key="close" onClick={() => setHistoryModalVisible(false)}>
            关闭
          </Button>
        ]}
        width={800}
      >
        {historyLoading ? (
          <div style={{ textAlign: 'center', padding: '20px' }}>
            加载中...
          </div>
        ) : (
          <Table
            dataSource={alertHistory || []}
            size="small"
            rowKey="id"
            pagination={false}
            columns={[
              {
                title: '操作',
                dataIndex: 'action',
                key: 'action',
                width: 100,
                render: (action: string) => {
                  const actionMap: { [key: string]: { text: string; color: string } } = {
                    'created': { text: '创建', color: 'blue' },
                    'updated': { text: '更新', color: 'orange' },
                    'silenced': { text: '静默', color: 'purple' },
                    'acknowledged': { text: '确认', color: 'green' },
                    'resolved': { text: '解决', color: 'gray' },
                  }
                  const actionInfo = actionMap[action] || { text: action, color: 'default' }
                  return <Tag color={actionInfo.color}>{actionInfo.text}</Tag>
                },
              },
              {
                title: '详情',
                dataIndex: 'details',
                key: 'details',
                render: (details: any) => {
                  if (!details || typeof details !== 'object') return '--'
                  return (
                    <div>
                      {Object.entries(details).map(([key, value]) => (
                        <div key={key} style={{ fontSize: '12px' }}>
                          <strong>{key}:</strong> {String(value)}
                        </div>
                      ))}
                    </div>
                  )
                },
              },
              {
                title: '时间',
                dataIndex: 'created_at',
                key: 'created_at',
                width: 180,
                render: (time: string) => new Date(time).toLocaleString(),
              },
            ]}
            locale={{ emptyText: '暂无历史记录' }}
          />
        )}
      </Modal>
    </div>
  )
}

export default Alerts