import React from 'react'
import { Card, Row, Col, Statistic, Progress, Table, Tag, Spin } from 'antd'
import { AlertOutlined, CheckCircleOutlined, ExclamationCircleOutlined, ClockCircleOutlined } from '@ant-design/icons'
import { useDashboardStats } from '@/hooks/useDashboard'
import { useAlerts } from '@/hooks/useAlerts'

const Dashboard: React.FC = () => {
  const { data: statsData, isLoading: statsLoading } = useDashboardStats()
  const { data: alertsData, isLoading: alertsLoading } = useAlerts({ page: 1, size: 10, sort: 'created_at', order: 'desc' })

  // 从API数据中提取统计信息
  const stats = {
    total: statsData?.total_alerts || 0,
    firing: statsData?.firing_alerts || 0,
    resolved: statsData?.resolved_alerts || 0,
    silenced: 0, // 静默数据可能需要单独接口
  }

  // 使用真实告警数据
  const recentAlerts = Array.isArray(alertsData?.items) 
    ? alertsData.items.map((alert: any, index: number) => ({
        key: alert.id?.toString() || index.toString(),
        alertname: alert.labels?.alertname || '未知告警',
        instance: alert.labels?.instance || '未知实例',
        severity: alert.severity || 'info',
        status: alert.status || 'unknown',
        time: alert.created_at ? new Date(alert.created_at).toLocaleString() : '未知时间',
      }))
    : []

  if (statsLoading || alertsLoading) {
    return (
      <div style={{ textAlign: 'center', padding: '50px' }}>
        <Spin size="large" />
      </div>
    )
  }

  const columns = [
    {
      title: '告警名称',
      dataIndex: 'alertname',
      key: 'alertname',
    },
    {
      title: '实例',
      dataIndex: 'instance',
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
      title: '时间',
      dataIndex: 'time',
      key: 'time',
    },
  ]

  return (
    <div>
      <Row gutter={[16, 16]} style={{ marginBottom: 24 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总告警数"
              value={stats.total}
              prefix={<AlertOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="告警中"
              value={stats.firing}
              prefix={<ExclamationCircleOutlined />}
              valueStyle={{ color: '#ff4d4f' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已解决"
              value={stats.resolved}
              prefix={<CheckCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="已静默"
              value={stats.silenced}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ color: '#faad14' }}
            />
          </Card>
        </Col>
      </Row>

      <Row gutter={[16, 16]}>
        <Col span={16}>
          <Card title="最近告警">
            <Table
              dataSource={recentAlerts}
              columns={columns}
              pagination={false}
              size="small"
              locale={{ emptyText: '暂无告警数据' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card title="告警分布">
            <div style={{ textAlign: 'center' }}>
              <Progress
                type="circle"
                percent={Math.round((stats.firing / stats.total) * 100)}
                format={() => `${stats.firing}/${stats.total}`}
                strokeColor="#ff4d4f"
                size={120}
              />
              <div style={{ marginTop: 16, fontSize: '14px', color: '#666' }}>
                当前告警比例
              </div>
            </div>
          </Card>
        </Col>
      </Row>
    </div>
  )
}

export default Dashboard