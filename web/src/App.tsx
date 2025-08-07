import { FC } from 'react'
import { Routes, Route } from 'react-router-dom'
import { Layout } from 'antd'
import Sidebar from '@/components/Sidebar'
import Header from '@/components/Header'
import Dashboard from '@/pages/Dashboard'
import Alerts from '@/pages/Alerts'
import Rules from '@/pages/Rules'
import Channels from '@/pages/Channels'
import Silences from '@/pages/Silences'
import AlertGroups from '@/pages/AlertGroups'
import Inhibitions from '@/pages/Inhibitions'
import Settings from '@/pages/Settings'
import Test from '@/pages/Test'

const { Content } = Layout

const App: FC = () => {
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
            <Route path="/silences" element={<Silences />} />
            <Route path="/alert-groups" element={<AlertGroups />} />
            <Route path="/inhibitions" element={<Inhibitions />} />
            <Route path="/settings" element={<Settings />} />
            <Route path="/test" element={<Test />} />
          </Routes>
        </Content>
      </Layout>
    </Layout>
  )
}

export default App