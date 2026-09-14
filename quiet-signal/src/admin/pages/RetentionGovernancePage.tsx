import { useState } from 'react'

export type LegalHold = {
  id: string
  entity_type: string
  entity_id: string
  reason: string
  placed_by: string
  placed_at: string
}

export type ExportApproval = {
  export_id: string
  requested_by: string
  status: 'PENDING' | 'APPROVED' | 'REJECTED'
}

export type RetentionPolicy = {
  entity_class: string
  retention_days: number
  archive_after_days: number
  purge_after_days: number
  immutable: boolean
}

export function RetentionGovernancePage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [holds, setHolds] = useState<LegalHold[]>([])
  const [approvals, setApprovals] = useState<ExportApproval[]>([])
  const [policies, setPolicies] = useState<RetentionPolicy[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [showPlaceForm, setShowPlaceForm] = useState(false)
  const [newHold, setNewHold] = useState({ entity_type: '', entity_id: '', reason: '' })

  function authHeaders(json = false): Record<string, string> {
    const h: Record<string, string> = {
      'X-Tenant-ID': tenantId,
      'X-Organization-ID': orgId,
      'Authorization': `Bearer ${localStorage.getItem('token') || ''}`,
    }
    if (json) h['Content-Type'] = 'application/json'
    return h
  }

  async function loadAll() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const [holdsR, approvalsR, policiesR] = await Promise.all([
        fetch('/api/v1/legal-holds', { headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId } }),
        fetch('/api/v1/exports/approvals', { headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId } }),
        fetch('/api/v1/retention-policies', { headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId } }),
      ])
      for (const r of [holdsR, approvalsR, policiesR]) {
        if (!r.ok) throw new Error(`HTTP ${r.status}`)
      }
      const [holdsData, approvalsData, policiesData] = await Promise.all([
        holdsR.json(), approvalsR.json(), policiesR.json(),
      ])
      setHolds((holdsData.legal_holds ?? holdsData) as LegalHold[])
      setApprovals((approvalsData.approvals ?? approvalsData) as ExportApproval[])
      setPolicies((policiesData.policies ?? policiesData) as RetentionPolicy[])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function placeHold() {
    if (!newHold.entity_type.trim() || !newHold.entity_id.trim() || !newHold.reason.trim()) {
      setError('Entity type, entity ID and reason are required to place a legal hold.')
      return
    }
    try {
      const r = await fetch('/api/v1/legal-holds', {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify(newHold),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      setShowPlaceForm(false)
      setNewHold({ entity_type: '', entity_id: '', reason: '' })
      await loadAll()
    } catch (e) {
      setError(String(e))
    }
  }

  async function releaseHold(hold: LegalHold) {
    try {
      const r = await fetch(`/api/v1/legal-holds/${hold.id}/release`, {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({ legal_hold_id: hold.id }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await loadAll()
    } catch (e) {
      setError(String(e))
    }
  }

  async function decideExport(approval: ExportApproval, decision: 'approve' | 'reject') {
    try {
      const r = await fetch(`/api/v1/exports/approvals/${approval.export_id}/${decision}`, {
        method: 'POST',
        headers: authHeaders(true),
        body: JSON.stringify({ export_id: approval.export_id }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await loadAll()
    } catch (e) {
      setError(String(e))
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Retention &amp; Governance</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={loadAll} disabled={!tenantId || !orgId || loading}>
          {loading ? 'Loading...' : 'Load Governance Data'}
        </button>
      </div>

      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      <section style={{ marginBottom: 32 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 12 }}>
          <h3 style={{ margin: 0 }}>Active Legal Holds</h3>
          <button onClick={() => setShowPlaceForm(v => !v)}>{showPlaceForm ? 'Hide Form' : 'Place New Hold'}</button>
        </div>

        {showPlaceForm && (
          <div style={{ marginBottom: 16, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
            <h4 style={{ marginTop: 0 }}>Place New Legal Hold</h4>
            <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'flex-end' }}>
              <div>
                <label style={{ display: 'block', marginBottom: 4 }}>Entity Type</label>
                <input placeholder="e.g., INSPECTION, EVIDENCE, PERSON" value={newHold.entity_type} onChange={e => setNewHold({ ...newHold, entity_type: e.target.value })} style={{ padding: 8, width: 180 }} />
              </div>
              <div>
                <label style={{ display: 'block', marginBottom: 4 }}>Entity ID</label>
                <input placeholder="Entity identifier" value={newHold.entity_id} onChange={e => setNewHold({ ...newHold, entity_id: e.target.value })} style={{ padding: 8, width: 220 }} />
              </div>
              <div style={{ flex: 1, minWidth: 260 }}>
                <label style={{ display: 'block', marginBottom: 4 }}>Reason</label>
                <input placeholder="Justification for the legal hold" value={newHold.reason} onChange={e => setNewHold({ ...newHold, reason: e.target.value })} style={{ padding: 8, width: '100%' }} />
              </div>
              <button onClick={placeHold} disabled={!newHold.entity_type || !newHold.entity_id || !newHold.reason}>Place Hold</button>
            </div>
          </div>
        )}

        {holds.length > 0 ? (
          <table style={{ borderCollapse: 'collapse', width: '100%' }}>
            <thead>
              <tr>
                <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>ID</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Entity</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Reason</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Placed By</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Placed At</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {holds.map(h => (
                <tr key={h.id}>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{h.id}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>
                    <strong>{h.entity_type}</strong>: {h.entity_id}
                  </td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{h.reason}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{h.placed_by}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12, whiteSpace: 'nowrap' }}>{formatDate(h.placed_at)}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>
                    <button onClick={() => releaseHold(h)} style={{ background: '#d97706', color: 'white' }}>Release</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          !loading && <p>No active legal holds.</p>
        )}
      </section>

      <section style={{ marginBottom: 32 }}>
        <h3 style={{ marginBottom: 12 }}>Export Approvals Queue</h3>
        {approvals.length > 0 ? (
          <table style={{ borderCollapse: 'collapse', width: '100%' }}>
            <thead>
              <tr>
                <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>Export ID</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Requested By</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Status</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
              </tr>
            </thead>
            <tbody>
              {approvals.map(a => (
                <tr key={a.export_id}>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{a.export_id}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{a.requested_by}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{a.status}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>
                    {a.status === 'PENDING' && (
                      <>
                        <button onClick={() => decideExport(a, 'approve')}>Approve</button>
                        <button onClick={() => decideExport(a, 'reject')} style={{ marginLeft: 8, background: '#ff4444', color: 'white' }}>Reject</button>
                      </>
                    )}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          !loading && <p>No export approvals pending.</p>
        )}
      </section>

      <section>
        <h3 style={{ marginBottom: 12 }}>Retention Policy Schedule</h3>
        {policies.length > 0 ? (
          <table style={{ borderCollapse: 'collapse', width: '100%' }}>
            <thead>
              <tr>
                <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>Entity Class</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Retention (days)</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Archive After</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Purge After</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Immutable</th>
              </tr>
            </thead>
            <tbody>
              {policies.map(p => (
                <tr key={p.entity_class}>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{p.entity_class}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{p.retention_days}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{p.archive_after_days}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{p.purge_after_days}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{p.immutable ? 'Yes' : 'No'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          !loading && <p>No retention policies configured.</p>
        )}
      </section>
    </div>
  )
}