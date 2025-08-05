import React from 'react'
import { Card, Row, Col, Statistic, Progress, Table, Tag } from 'antd'
import { AlertOutlined, CheckCircleOutlined, ExclamationCircleOutlined, ClockCircleOutlined } from '@ant-design/icons'

const Dashboard: React.FC = () => {
  // 模拟数据
  const stats = {
    total: 156,
    firing: 45,
    resolved: 111,
    silenced: 12,
  }

  const recentAlerts = [
    {
      key: '1',
      alertname: 'HighCPUUsage',
      instance: 'server1:9100',
      severity: 'critical',
      status: 'firing',
      time: '2分钟前',
    },
    {
      key: '2',
      alertname: 'DiskSpaceLow',
      instance: 'server2:9100',
      severity: 'warning',
      status: 'firing',
      time: '5分钟前',
    },
    {
      key: '3',
      alertname: 'ServiceDown',
      instance: 'api-server:8080',
      severity: 'critical',
      status: 'resolved',
      time: '10分钟前',
    },
  ]

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