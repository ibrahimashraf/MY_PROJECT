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
  const [license, setLicense] = useState<License | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')

  async function fetchLicense() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/v1/licenses/validate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
        body: JSON.stringify({ tenant_id: tenantId, organization_id: orgId }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setLicense(data as License)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>License Management</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20 }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchLicense} disabled={!tenantId || !orgId}>Load License</button>
      </div>
      {loading && <p>Loading...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}
      {license && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>Field</th>
              <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>Value</th>
            </tr>
          </thead>
          <tbody>
            <tr><td style={{ border: '1px solid #ccc', padding: 8 }}>Tier</td><td style={{ border: '1px solid #ccc', padding: 8 }}>{license.tier}</td></tr>
            <tr><td style={{ border: '1px solid #ccc', padding: 8 }}>Status</td><td style={{ border: '1px solid #ccc', padding: 8 }}>{license.status}</td></tr>
            <tr><td style={{ border: '1px solid #ccc', padding: 8 }}>Max Inspectors</td><td style={{ border: '1px solid #ccc', padding: 8 }}>{license.max_inspectors}</td></tr>
            <tr><td style={{ border: '1px solid #ccc', padding: 8 }}>Max Inspections/Month</td><td style={{ border: '1px solid #ccc', padding: 8 }}>{license.max_inspections_per_month}</td></tr>
            <tr><td style={{ border: '1px solid #ccc', padding: 8 }}>Created</td><td style={{ border: '1px solid #ccc', padding: 8 }}>{new Date(license.created_at).toLocaleDateString()}</td></tr>
          </tbody>
        </table>
      )}
    </div>
  )
}
