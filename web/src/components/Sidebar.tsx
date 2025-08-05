import React from 'react'
import { Layout, Menu } from 'antd'
import { useNavigate, useLocation } from 'react-router-dom'
import {
  DashboardOutlined,
  AlertOutlined,
  SettingOutlined,
  BellOutlined,
  NotificationOutlined,
} from '@ant-design/icons'

const { Sider } = Layout

const Sidebar: React.FC = () => {
  const navigate = useNavigate()
  const location = useLocation()

  const menuItems = [
    {
      key: '/',
      icon: <DashboardOutlined />,
      label: '仪表板',
    },
    {
      key: '/alerts',
      icon: <AlertOutlined />,
      label: '告警管理',
    },
    {
      key: '/rules',
      icon: <SettingOutlined />,
      label: '规则管理',
    },
    {
      key: '/channels',
      icon: <NotificationOutlined />,
      label: '通知渠道',
    },
    {
      key: '/settings',
      icon: <BellOutlined />,
      label: '系统设置',
    },
    {
      key: '/test',
      icon: <SettingOutlined />,
      label: 'API测试',
    },
  ]

  const handleMenuClick = ({ key }: { key: string }) => {
    navigate(key)
  }

  return (
    <Sider width={200} style={{ background: '#fff' }}>
      <div style={{ padding: '16px', textAlign: 'center', fontSize: '20px', fontWeight: 'bold' }}>
        AlertBot
      </div>
      <Menu
        mode="inline"
        selectedKeys={[location.pathname]}
        items={menuItems}
        onClick={handleMenuClick}
        style={{ height: '100%', borderRight: 0 }}
      />
    </Sider>
  )
}

export default Sidebar