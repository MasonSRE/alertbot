import React from 'react'
import { Layout, Space, Button, Badge, Avatar } from 'antd'
import { BellOutlined, UserOutlined, SettingOutlined } from '@ant-design/icons'

const { Header: AntHeader } = Layout

const Header: React.FC = () => {
  return (
    <AntHeader style={{ background: '#fff', padding: '0 24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
      <div style={{ fontSize: '16px', fontWeight: 500 }}>
        告警管理平台
      </div>
      
      <Space size="middle">
        <Badge count={5} size="small">
          <Button type="text" icon={<BellOutlined />} />
        </Badge>
        
        <Button type="text" icon={<SettingOutlined />} />
        
        <Space>
          <Avatar size="small" icon={<UserOutlined />} />
          <span>管理员</span>
        </Space>
      </Space>
    </AntHeader>
  )
}

export default Header