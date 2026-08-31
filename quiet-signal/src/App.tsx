import { BrowserRouter, Routes, Route, Link } from 'react-router-dom'
import { LicensePage } from './admin/pages/LicensePage'
import { FeatureFlagsPage } from './admin/pages/FeatureFlagsPage'
import { SettingsPage } from './admin/pages/SettingsPage'
import { FieldApp } from './field/FieldApp'
import './App.css'

function AdminLayout({ children }: { children: React.ReactNode }) {
  return (
    <div style={{ display: 'flex', minHeight: '100vh' }}>
      <nav style={{ width: 220, background: '#f5f5f5', padding: 16, borderRight: '1px solid #ddd' }}>
        <h3 style={{ marginTop: 0 }}>INTEGIN</h3>
        <ul style={{ listStyle: 'none', padding: 0 }}>
          <li><Link to="/admin/licenses">Licenses</Link></li>
          <li><Link to="/admin/feature-flags">Feature Flags</Link></li>
          <li><Link to="/admin/settings">Settings</Link></li>
        </ul>
        <hr />
        <ul style={{ listStyle: 'none', padding: 0 }}>
          <li><Link to="/field">Field App</Link></li>
          <li><Link to="/">Advisory Panel</Link></li>
        </ul>
      </nav>
      <main style={{ flex: 1, padding: 0 }}>
        {children}
      </main>
    </div>
  )
}

function AdvisoryPanel() {
  return <div style={{ padding: 20 }}><h2>Advisory Panel</h2><p>Quiet Signal secondary advisor.</p></div>
}

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/admin/licenses" element={<AdminLayout><LicensePage /></AdminLayout>} />
        <Route path="/admin/feature-flags" element={<AdminLayout><FeatureFlagsPage /></AdminLayout>} />
        <Route path="/admin/settings" element={<AdminLayout><SettingsPage /></AdminLayout>} />
        <Route path="/field" element={<AdminLayout><FieldApp /></AdminLayout>} />
        <Route path="/" element={<AdminLayout><AdvisoryPanel /></AdminLayout>} />
      </Routes>
    </BrowserRouter>
  )
}
