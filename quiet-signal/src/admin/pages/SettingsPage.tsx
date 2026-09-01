import { useState } from 'react'

type Setting = {
  id: string
  setting_key: string
  setting_value: Record<string, unknown>
  scope: string
  is_editable: boolean
  description: string
}

export function SettingsPage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [settings, setSettings] = useState<Setting[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function fetchSettings() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/v1/admin/settings', {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setSettings(data as Setting[])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function updateSetting(key: string, scope: string, value: Record<string, unknown>) {
    if (!tenantId || !orgId) return
    try {
      const r = await fetch('/api/v1/admin/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
        body: JSON.stringify({ setting_key: key, setting_value: value, scope, created_by: 'admin-ui' }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchSettings()
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Tenant Settings</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20 }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchSettings} disabled={!tenantId || !orgId}>Load Settings</button>
      </div>

      {loading && <p>Loading...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      {settings.length > 0 && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Key</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Scope</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Value</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Editable</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {settings.map(s => (
              <tr key={s.id}>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{s.setting_key}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{s.scope}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  <pre style={{ margin: 0, fontSize: 12 }}>{JSON.stringify(s.setting_value, null, 2)}</pre>
                </td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{s.is_editable ? 'Yes' : 'No'}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  {s.is_editable && (
                    <button onClick={() => {
                      const newVal = prompt('Enter new JSON value:', JSON.stringify(s.setting_value))
                      if (newVal) {
                        try {
                          updateSetting(s.setting_key, s.scope, JSON.parse(newVal))
                        } catch { alert('Invalid JSON') }
                        }
                    }}>Edit</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
