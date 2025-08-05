import React from 'react'
import { Routes, Route } from 'react-router-dom'
import { Layout } from 'antd'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'
import Dashboard from '@/pages/Dashboard'
import Alerts from '@/pages/Alerts'
import Rules from '@/pages/Rules'
import Channels from '@/pages/Channels'
import Settings from '@/pages/Settings'
import Test from '@/pages/Test'

const { Content } = Layout

function App() {
  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Sidebar />
      <Layout>
        <Header />
        <Content style={{ margin: '16px', padding: '24px', background: '#fff' }}>
          <Routes>
            <Route path="/" element={<Dashboard />} />
            <Route path="/alerts" element={<Alerts />} />
            <Route path="/rules" element={<Rules />} />
            <Route path="/channels" element={<Channels />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/test" element={<Test />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}

export default App