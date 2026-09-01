import { useState } from 'react'

type FlagOverride = {
  id: string
  flag_key: string
  scope: string
  scope_id: string
  state: string
  reason: string
  expires_at: string | null
  created_by: string
  created_at: string
}

export function FeatureFlagsPage() {
  const [overrides, setOverrides] = useState<FlagOverride[]>([])
  const [loading, setLoading] = useState(true)
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [newFlag, setNewFlag] = useState({ flag_key: '', scope: 'ORGANIZATION', scope_id: '', state: 'ENABLED', reason: '' })

  async function fetchFlags() {
    if (!tenantId || !orgId) return
    setLoading(true)
    try {
      const r = await fetch('/api/v1/admin/feature-flags', {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setOverrides(data as FlagOverride[])
    } catch (e) {
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  async function upsertFlag() {
    if (!tenantId || !orgId || !newFlag.flag_key || !newFlag.scope_id) return
    try {
      const r = await fetch('/api/v1/admin/feature-flags', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
        body: JSON.stringify({ ...newFlag, actor_id: 'admin-ui' }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchFlags()
      setNewFlag({ flag_key: '', scope: 'ORGANIZATION', scope_id: '', state: 'ENABLED', reason: '' })
    } catch (e) {
      console.error(e)
    }
  }

  async function deleteFlag(key: string, scope: string, scopeId: string) {
    if (!tenantId || !orgId) return
    try {
      const r = await fetch(`/api/v1/admin/feature-flags/${key}/${scope}/${scopeId}`, {
        method: 'DELETE',
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchFlags()
    } catch (e) {
      console.error(e)
    }
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Feature Flag Management</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20 }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchFlags} disabled={!tenantId || !orgId}>Load Flags</button>
      </div>

      <h3>Add/Update Override</h3>
      <div style={{ display: 'flex', gap: 8, marginBottom: 20, flexWrap: 'wrap' }}>
        <input placeholder="Flag Key" value={newFlag.flag_key} onChange={e => setNewFlag({ ...newFlag, flag_key: e.target.value })} style={{ width: 200 }} />
        <select value={newFlag.scope} onChange={e => setNewFlag({ ...newFlag, scope: e.target.value })}>
          <option value="ORGANIZATION">Organization</option>
          <option value="USER">User</option>
          <option value="CLIENT">Client</option>
          <option value="PROJECT">Project</option>
          <option value="DEVICE">Device</option>
        </select>
        <input placeholder="Scope ID" value={newFlag.scope_id} onChange={e => setNewFlag({ ...newFlag, scope_id: e.target.value })} style={{ width: 200 }} />
        <select value={newFlag.state} onChange={e => setNewFlag({ ...newFlag, state: e.target.value })}>
          <option value="ENABLED">Enabled</option>
          <option value="DISABLED">Disabled</option>
          <option value="INHERITED">Inherited</option>
          <option value="EXPIRED">Expired</option>
        </select>
        <input placeholder="Reason" value={newFlag.reason} onChange={e => setNewFlag({ ...newFlag, reason: e.target.value })} style={{ width: 200 }} />
        <button onClick={upsertFlag}>Save</button>
      </div>

      {loading ? <p>Loading...</p> : (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Flag</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Scope</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Scope ID</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>State</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Reason</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {overrides.map(o => (
              <tr key={o.id}>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{o.flag_key}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{o.scope}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{o.scope_id}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{o.state}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{o.reason}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  <button onClick={() => deleteFlag(o.flag_key, o.scope, o.scope_id)}>Delete</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  )
}
