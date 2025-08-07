import React, { useState } from 'react'
import { Card, Table, Button, Space, Modal, Form, Input, Tag, DatePicker, Select, Alert } from 'antd'
import { PlusOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { Silence } from '@/types'
import { useSilences, useCreateSilence, useDeleteSilence, useTestSilence } from '@/hooks/useSilences'
import dayjs from 'dayjs'

const { RangePicker } = DatePicker
const { Option } = Select

const Silences: React.FC = () => {
  const [modalVisible, setModalVisible] = useState(false)
  // const [editingSilence, setEditingSilence] = useState<Silence | null>(null)
  const [testModalVisible, setTestModalVisible] = useState(false)
  const [testSilence, setTestSilence] = useState<any>(null)
  const [form] = Form.useForm()
  const [testForm] = Form.useForm()

  const { data: silences = [], isLoading, refetch } = useSilences()
  const createMutation = useCreateSilence()
  const deleteMutation = useDeleteSilence()
  const testMutation = useTestSilence()

  const getSilenceStatus = (silence: Silence) => {
    const now = dayjs()
    const startsAt = dayjs(silence.starts_at)
    const endsAt = dayjs(silence.ends_at)
    
    if (now.isBefore(startsAt)) {
      return { color: 'blue', text: '待生效' }
    } else if (now.isAfter(endsAt)) {
      return { color: 'gray', text: '已过期' }
    } else {
      return { color: 'green', text: '生效中' }
    }
  }

  const columns: ColumnsType<Silence> = [
    {
      title: '匹配条件',
      dataIndex: 'matchers',
      key: 'matchers',
      render: (matchers: any) => {
        // Handle JSONB structure - matchers might be {matchers: [...]} or directly [...]
        const matchersArray = Array.isArray(matchers) 
          ? matchers 
          : (matchers?.matchers && Array.isArray(matchers.matchers))
            ? matchers.matchers
            : []
        
        return (
          <div>
            {matchersArray.map((matcher: any, index: number) => (
              <Tag key={index} style={{ margin: '2px' }}>
                {matcher.name}{matcher.is_regex ? '~' : '='}{matcher.value}
              </Tag>
            ))}
          </div>
        )
      },
    },
    {
      title: '状态',
      key: 'status',
      width: 100,
      render: (_, record) => {
        const { color, text } = getSilenceStatus(record)
        return <Tag color={color}>{text}</Tag>
      },
    },
    {
      title: '开始时间',
      dataIndex: 'starts_at',
      key: 'starts_at',
      width: 180,
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '--',
    },
    {
      title: '结束时间',
      dataIndex: 'ends_at',
      key: 'ends_at',
      width: 180,
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '--',
    },
    {
      title: '创建者',
      dataIndex: 'creator',
      key: 'creator',
      width: 100,
    },
    {
      title: '备注',
      dataIndex: 'comment',
      key: 'comment',
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'created_at',
      key: 'created_at',
      width: 180,
      render: (time: string) => time ? dayjs(time).format('YYYY-MM-DD HH:mm:ss') : '--',
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
            icon={<DeleteOutlined />}
            danger
            onClick={() => handleDelete(record)}
            disabled={false}
          >
            删除
          </Button>
        </Space>
      ),
    },
  ]

  const handleCreate = () => {
    // setEditingSilence(null)
    form.resetFields()
    setModalVisible(true)
  }

  const handleDelete = (silence: Silence) => {
    Modal.confirm({
      title: '确认删除静默规则',
      content: `确定要删除此静默规则吗？`,
      onOk() {
        deleteMutation.mutate(silence.id)
      },
    })
  }

  const handleTest = (silence: Silence) => {
    setTestSilence(silence)
    testForm.resetFields()
    setTestModalVisible(true)
  }

  const handleSubmit = (values: any) => {
    const silenceData = {
      matchers: {
        matchers: values.matchers.map((matcher: any) => ({
          name: matcher.name,
          value: matcher.value,
          is_regex: matcher.is_regex || false
        }))
      },
      starts_at: values.timeRange[0].toISOString(),
      ends_at: values.timeRange[1].toISOString(),
      creator: values.creator,
      comment: values.comment || ''
    }
    
    createMutation.mutate(silenceData)
    setModalVisible(false)
  }

  const handleTestSubmit = (values: any) => {
    if (testSilence) {
      // Handle JSONB structure for matchers
      const matchersArray = Array.isArray(testSilence.matchers) 
        ? testSilence.matchers 
        : (testSilence.matchers?.matchers && Array.isArray(testSilence.matchers.matchers))
          ? testSilence.matchers.matchers
          : []
      
      const testData = {
        matchers: matchersArray,
        labels: values.labels || {}
      }
      testMutation.mutate(testData)
    }
  }

  return (
    <div>
      <Alert
        message="静默规则说明"
        description="静默规则用于暂时抑制匹配条件的告警通知。在静默期间内，匹配的告警不会发送通知，但仍会在告警列表中显示。"
        type="info"
        showIcon
        style={{ marginBottom: 16 }}
      />
      <Card>
        <div style={{ marginBottom: 16 }}>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleCreate}>
              创建静默规则
            </Button>
            <Button onClick={() => refetch()}>刷新</Button>
          </Space>
        </div>

        <Table
          columns={columns}
          dataSource={silences}
          loading={isLoading}
          rowKey="id"
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total, range) => `第 ${range[0]}-${range[1]} 条，共 ${total} 条`,
          }}
        />
      </Card>

      {/* 创建静默规则Modal */}
      <Modal
        title="创建静默规则"
        open={modalVisible}
        onCancel={() => setModalVisible(false)}
        footer={null}
        width={800}
      >
        <Form form={form} onFinish={handleSubmit} layout="vertical">
          <Form.List name="matchers">
            {(fields, { add, remove }) => (
              <>
                {fields.map((field, index) => (
                  <Space key={field.key} style={{ display: 'flex', marginBottom: 8 }} align="baseline">
                    <Form.Item
                      {...field}
                      name={[field.name, 'name']}
                      label={index === 0 ? '标签名' : ''}
                      rules={[{ required: true, message: '请输入标签名' }]}
                    >
                      <Input placeholder="例如: alertname" style={{ width: 150 }} />
                    </Form.Item>
                    <Form.Item
                      {...field}
                      name={[field.name, 'value']}
                      label={index === 0 ? '标签值' : ''}
                      rules={[{ required: true, message: '请输入标签值' }]}
                    >
                      <Input placeholder="例如: HighCPUUsage" style={{ width: 200 }} />
                    </Form.Item>
                    <Form.Item
                      {...field}
                      name={[field.name, 'is_regex']}
                      label={index === 0 ? '正则匹配' : ''}
                      valuePropName="checked"
                    >
                      <input type="checkbox" />
                    </Form.Item>
                    <Button type="link" onClick={() => remove(field.name)} danger>
                      删除
                    </Button>
                  </Space>
                ))}
                <Form.Item>
                  <Button type="dashed" onClick={() => add()} block icon={<PlusOutlined />}>
                    添加匹配条件
                  </Button>
                </Form.Item>
              </>
            )}
          </Form.List>
          
          <Form.Item
            name="timeRange"
            label="静默时间范围"
            rules={[{ required: true, message: '请选择静默时间范围' }]}
          >
            <RangePicker 
              showTime 
              format="YYYY-MM-DD HH:mm:ss"
              placeholder={['开始时间', '结束时间']}
              style={{ width: '100%' }}
            />
          </Form.Item>
          
          <Form.Item
            name="creator"
            label="创建者"
            rules={[{ required: true, message: '请输入创建者' }]}
          >
            <Input placeholder="请输入创建者名称" />
          </Form.Item>
          
          <Form.Item name="comment" label="备注">
            <Input.TextArea rows={3} placeholder="请输入静默原因或备注信息" />
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                创建
              </Button>
              <Button onClick={() => setModalVisible(false)}>
                取消
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>

      {/* 测试静默规则Modal */}
      <Modal
        title="测试静默规则"
        open={testModalVisible}
        onCancel={() => setTestModalVisible(false)}
        footer={null}
        width={600}
      >
        <Form form={testForm} onFinish={handleTestSubmit} layout="vertical">
          <Alert
            message="测试说明"
            description="输入一组标签来测试此静默规则是否会匹配。"
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
          
          <Form.Item name="customLabels" label="其他标签 (JSON格式)">
            <Input.TextArea 
              rows={3} 
              placeholder='{"job": "node-exporter", "team": "sre"}' 
            />
          </Form.Item>
          
          <Form.Item>
            <Space>
              <Button type="primary" htmlType="submit">
                测试匹配
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

export default Silences