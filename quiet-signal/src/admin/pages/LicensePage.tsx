import { useState } from 'react'

type License = {
  id: string
  tier: string
  status: string
  max_inspectors: number
  max_inspections_per_month: number
  features: Record<string, boolean>
  created_at: string
}

export function LicensePage() {
  const [licenses, setLicenses] = useState<License[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [showForm, setShowForm] = useState(false)
  const [newLicense, setNewLicense] = useState({
    tier: 'starter',
    max_inspectors: 5,
    max_inspections_per_month: 100,
    features: {},
  })
  const [selectedLicenseId, setSelectedLicenseId] = useState<string | null>(null)
  const [renewExpiresAt, setRenewExpiresAt] = useState('')

  async function fetchLicenses() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/v1/licenses', {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setLicenses(data as License[])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function issueLicense() {
    if (!tenantId || !orgId) return
    try {
      const r = await fetch('/api/v1/licenses', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('token') || ''}`, 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
        body: JSON.stringify(newLicense),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchLicenses()
      setShowForm(false)
      setNewLicense({ tier: 'starter', max_inspectors: 5, max_inspections_per_month: 100, features: {} })
    } catch (e) {
      setError(String(e))
    }
  }

  async function revokeLicense(id: string) {
    if (!tenantId || !orgId || !confirm('Revoke this license?')) return
    try {
      const r = await fetch('/api/v1/licenses/revoke', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('token') || ''}`, 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
        body: JSON.stringify({ license_id: id }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchLicenses()
    } catch (e) {
      setError(String(e))
    }
  }

  async function renewLicense(id: string) {
    if (!tenantId || !orgId || !renewExpiresAt) return
    try {
      const r = await fetch('/api/v1/licenses/renew', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Authorization': `Bearer ${localStorage.getItem('token') || ''}`, 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
        body: JSON.stringify({ license_id: id, new_expires_at: renewExpiresAt }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchLicenses()
      setRenewExpiresAt('')
    } catch (e) {
      setError(String(e))
    }
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>License Management</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchLicenses} disabled={!tenantId || !orgId}>Load Licenses</button>
      </div>

      {loading && <p>Loading...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      <div style={{ marginBottom: 20 }}>
        <button onClick={() => setShowForm(!showForm)}>
          {showForm ? 'Hide Form' : 'Issue New License'}
        </button>
      </div>

      {showForm && (
        <div style={{ marginBottom: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
          <h3>Issue New License</h3>
          <div style={{ display: 'flex', gap: 12, marginBottom: 12, flexWrap: 'wrap', alignItems: 'flex-end' }}>
            <div>
              <label style={{ display: 'block', marginBottom: 4 }}>Tier</label>
              <select value={newLicense.tier} onChange={e => setNewLicense({ ...newLicense, tier: e.target.value })}>
                <option value="starter">Starter</option>
                <option value="professional">Professional</option>
                <option value="enterprise">Enterprise</option>
              </select>
            </div>
            <div>
              <label style={{ display: 'block', marginBottom: 4 }}>Max Inspectors</label>
              <input type="number" value={newLicense.max_inspectors} onChange={e => setNewLicense({ ...newLicense, max_inspectors: Number(e.target.value) })} style={{ width: 120 }} />
            </div>
            <div>
              <label style={{ display: 'block', marginBottom: 4 }}>Max Inspections/Month</label>
              <input type="number" value={newLicense.max_inspections_per_month} onChange={e => setNewLicense({ ...newLicense, max_inspections_per_month: Number(e.target.value) })} style={{ width: 120 }} />
            </div>
            <button onClick={issueLicense}>Issue License</button>
          </div>
        </div>
      )}

      {licenses.length > 0 && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>ID</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Tier</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Status</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Max Inspectors</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Max Inspections/Month</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Created</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {licenses.map(lic => (
              <tr key={lic.id}>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{lic.id}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{lic.tier}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{lic.status}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{lic.max_inspectors}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{lic.max_inspections_per_month}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{new Date(lic.created_at).toLocaleDateString()}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  <button onClick={() => { setSelectedLicenseId(lic.id); setRenewExpiresAt(new Date(Date.now() + 365*24*60*60*1000).toISOString().slice(0, 16)) }}>Renew</button>
                  <button onClick={() => revokeLicense(lic.id)} style={{ marginLeft: 8, background: '#ff4444', color: 'white' }}>Revoke</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {licenses.length === 0 && !loading && (
        <p>No licenses found. Click "Issue New License" to create one.</p>
      )}

      {selectedLicenseId && (
        <div style={{ marginTop: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
          <h3>Renew License {selectedLicenseId}</h3>
          <div style={{ display: 'flex', gap: 12, marginBottom: 12, alignItems: 'flex-end' }}>
            <div>
              <label style={{ display: 'block', marginBottom: 4 }}>New Expires At</label>
              <input type="datetime-local" value={renewExpiresAt} onChange={e => setRenewExpiresAt(e.target.value)} style={{ width: 250 }} />
            </div>
            <button onClick={() => { renewLicense(selectedLicenseId!); setSelectedLicenseId(null); }}>Renew</button>
            <button onClick={() => setSelectedLicenseId(null)} style={{ marginLeft: 8 }}>Cancel</button>
          </div>
        </div>
      )}

      {loading && <p>Loading...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}
    </div>
  )
}
