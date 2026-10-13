import { useState } from 'react'

export type DeviceRecord = {
  device_id: string
  platform: string
  status: 'PENDING' | 'APPROVED' | 'REJECTED' | 'REVOKED'
  public_key_fingerprint: string
  enrolled_at: string
}

export type AuthorityPackage = {
  epoch: number
  valid_until: string
  key_id: string
}

export function DeviceWorkbenchPage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [devices, setDevices] = useState<DeviceRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [approveTarget, setApproveTarget] = useState<DeviceRecord | null>(null)
  const [approveEpoch, setApproveEpoch] = useState('')
  const [approveDuration, setApproveDuration] = useState('')
  const [revokeTarget, setRevokeTarget] = useState<DeviceRecord | null>(null)
  const [revokeReason, setRevokeReason] = useState('')
  const [authorityFor, setAuthorityFor] = useState<DeviceRecord | null>(null)
  const [authority, setAuthority] = useState<AuthorityPackage | null>(null)

  function authHeaders(json = false): Record<string, string> {
    const h: Record<string, string> = {
      'X-Tenant-ID': tenantId,
      'X-Organization-ID': orgId,
      'Authorization': `Bearer ${localStorage.getItem('token') || ''}`,
    }
    if (json) h['Content-Type'] = 'application/json'
    return h
  }

  async function fetchDevices() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/v1/devices', {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setDevices((data.devices ?? data) as DeviceRecord[])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function approveDevice() {
    if (!approveTarget) return
    const epoch = Number(approveEpoch)
    const duration = Number(approveDuration)
    if (Number.isNaN(epoch) || Number.isNaN(duration) || duration <= 0) {
      setError('Epoch must be numeric and duration must be a positive number of seconds.')
      return
    }
    try {
      const r = await fetch('/api/v1/devices/approve', {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({ device_id: approveTarget.device_id, epoch, duration_seconds: duration }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      setApproveTarget(null)
      setApproveEpoch('')
      setApproveDuration('')
      await fetchDevices()
    } catch (e) {
      setError(String(e))
    }
  }

  async function revokeDevice() {
    if (!revokeTarget) return
    if (!revokeReason.trim()) {
      setError('A mandatory reason is required to revoke a device.')
      return
    }
    try {
      const r = await fetch('/api/v1/devices/revoke', {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({ device_id: revokeTarget.device_id, reason: revokeReason.trim() }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      setRevokeTarget(null)
      setRevokeReason('')
      await fetchDevices()
    } catch (e) {
      setError(String(e))
    }
  }

  async function inspectAuthority(device: DeviceRecord) {
    if (!device || device.status !== 'APPROVED') return
    setError(null)
    try {
      const r = await fetch(`/api/v1/devices/${device.device_id}/authority-package`, {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setAuthorityFor(device)
      setAuthority(data as AuthorityPackage)
    } catch (e) {
      setError(String(e))
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Device Workbench</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchDevices} disabled={!tenantId || !orgId || loading}>
          {loading ? 'Loading...' : 'Load Devices'}
        </button>
      </div>

      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      {devices.length > 0 && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>Device ID</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Platform</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Status</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Public Key Fingerprint</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Enrolled At</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {devices.map(d => (
              <tr key={d.device_id}>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{d.device_id}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{d.platform}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{d.status}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 11, fontFamily: 'monospace' }}>{d.public_key_fingerprint}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12, whiteSpace: 'nowrap' }}>{formatDate(d.enrolled_at)}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  {d.status === 'PENDING' && (
                    <button onClick={() => { setApproveTarget(d); setApproveEpoch(String(Math.floor(Date.now() / 1000))); setApproveDuration('86400') }}>Approve</button>
                  )}
                  {d.status !== 'REVOKED' && (
                    <button onClick={() => { setRevokeTarget(d); setRevokeReason('') }} style={{ marginLeft: 8, background: '#ff4444', color: 'white' }}>Revoke</button>
                  )}
                  {d.status === 'APPROVED' && (
                    <button onClick={() => inspectAuthority(d)} style={{ marginLeft: 8 }}>Inspect Authority</button>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {devices.length === 0 && !loading && <p>No devices found. Enter tenant/organization and load.</p>}

      {approveTarget && (
        <div style={{ marginTop: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
          <h3>Approve Enrollment: {approveTarget.device_id}</h3>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'flex-end' }}>
            <div>
              <label style={{ display: 'block', marginBottom: 4 }}>Epoch</label>
              <input type="number" value={approveEpoch} onChange={e => setApproveEpoch(e.target.value)} style={{ width: 160 }} />
            </div>
            <div>
              <label style={{ display: 'block', marginBottom: 4 }}>Duration (seconds)</label>
              <input type="number" value={approveDuration} onChange={e => setApproveDuration(e.target.value)} style={{ width: 160 }} />
            </div>
            <button onClick={approveDevice}>Approve</button>
            <button onClick={() => setApproveTarget(null)} style={{ marginLeft: 8 }}>Cancel</button>
          </div>
        </div>
      )}

      {revokeTarget && (
        <div style={{ marginTop: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
          <h3>Revoke Device: {revokeTarget.device_id}</h3>
          <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'flex-end' }}>
            <div style={{ flex: 1, minWidth: 300 }}>
              <label style={{ display: 'block', marginBottom: 4 }}>Reason (required)</label>
              <input
                placeholder="Mandatory justification for revocation"
                value={revokeReason}
                onChange={e => setRevokeReason(e.target.value)}
                style={{ width: '100%', padding: 8 }}
              />
            </div>
            <button onClick={revokeDevice} style={{ background: '#ff4444', color: 'white' }}>Revoke</button>
            <button onClick={() => setRevokeTarget(null)} style={{ marginLeft: 8 }}>Cancel</button>
          </div>
          {!revokeReason.trim() && <p style={{ fontSize: 12, color: '#d97706' }}>A reason is mandatory; the API will reject revocation without one.</p>}
        </div>
      )}

      {authorityFor && authority && (
        <div style={{ marginTop: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
          <h3>Authority Package: {authorityFor.device_id}</h3>
          <table style={{ borderCollapse: 'collapse' }}>
            <tbody>
              <tr>
                <td style={{ border: '1px solid #ccc', padding: 8, fontWeight: 'bold' }}>Epoch</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12, fontFamily: 'monospace' }}>{authority.epoch}</td>
              </tr>
              <tr>
                <td style={{ border: '1px solid #ccc', padding: 8, fontWeight: 'bold' }}>Valid Until</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{formatDate(authority.valid_until)}</td>
              </tr>
              <tr>
                <td style={{ border: '1px solid #ccc', padding: 8, fontWeight: 'bold' }}>Key ID</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12, fontFamily: 'monospace' }}>{authority.key_id}</td>
              </tr>
            </tbody>
          </table>
          <button onClick={() => { setAuthorityFor(null); setAuthority(null) }} style={{ marginTop: 12 }}>Close</button>
        </div>
      )}
    </div>
  )
}