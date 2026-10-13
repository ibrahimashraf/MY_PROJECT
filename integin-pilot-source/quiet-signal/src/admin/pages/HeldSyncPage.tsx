import { useState } from 'react'

export type HeldTransaction = {
  transaction_id: string
  device_id: string
  sequence_number: number
  status: 'HELD' | 'CONFLICT' | 'QUEUED'
  error_reason: string
  held_at: string
}

export type HeldTransactionDetail = HeldTransaction & {
  payload_hash: string
  inspection_id: string
  conflict_analysis: Record<string, unknown>
}

export function HeldSyncPage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [transactions, setTransactions] = useState<HeldTransaction[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [selected, setSelected] = useState<HeldTransactionDetail | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  async function fetchHeld() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/v1/sync/held', {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setTransactions((data.transactions ?? data) as HeldTransaction[])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function loadDetail(tx: HeldTransaction) {
    setDetailLoading(true)
    setError(null)
    try {
      const r = await fetch(`/api/v1/sync/held/${tx.transaction_id}`, {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setSelected(data as HeldTransactionDetail)
    } catch (e) {
      setError(String(e))
    } finally {
      setDetailLoading(false)
    }
  }

  async function reconcile(tx: HeldTransaction) {
    if (!tx) return
    setError(null)
    try {
      const r = await fetch('/api/v1/sync/reconcile', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`,
          'X-Tenant-ID': tenantId,
          'X-Organization-ID': orgId,
        },
        body: JSON.stringify({ transaction_id: tx.transaction_id }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchHeld()
      if (selected?.transaction_id === tx.transaction_id) await loadDetail(tx)
    } catch (e) {
      setError(String(e))
    }
  }

  async function reprovision(tx: HeldTransaction) {
    if (!tx) return
    setError(null)
    try {
      const r = await fetch('/work-orders/provisional', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${localStorage.getItem('token') || ''}`,
          'X-Tenant-ID': tenantId,
          'X-Organization-ID': orgId,
        },
        body: JSON.stringify({ transaction_id: tx.transaction_id }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchHeld()
    } catch (e) {
      setError(String(e))
    }
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Held Sync Workbench</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchHeld} disabled={!tenantId || !orgId || loading}>
          {loading ? 'Loading...' : 'Load Held Transactions'}
        </button>
      </div>

      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      {transactions.length > 0 && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>Transaction ID</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Device ID</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Sequence</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Status</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Error Reason</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Held At</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {transactions.map(tx => (
              <tr key={tx.transaction_id}>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{tx.transaction_id}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{tx.device_id}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{tx.sequence_number}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{tx.status}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 11 }}>{tx.error_reason}</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12, whiteSpace: 'nowrap' }}>{formatDate(tx.held_at)}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  <button onClick={() => loadDetail(tx)} disabled={detailLoading}>Detail</button>
                  <button onClick={() => reconcile(tx)} style={{ marginLeft: 8 }}>Reconcile</button>
                  <button onClick={() => reprovision(tx)} style={{ marginLeft: 8 }}>Reprovision</button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {transactions.length === 0 && !loading && <p>No held transactions found.</p>}

      {selected && (
        <div style={{ marginTop: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
          <h3>Transaction Detail: {selected.transaction_id}</h3>
          <table style={{ borderCollapse: 'collapse' }}>
            <tbody>
              <tr>
                <td style={{ border: '1px solid #ccc', padding: 8, fontWeight: 'bold' }}>Payload Hash</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 11, fontFamily: 'monospace' }}>{selected.payload_hash}</td>
              </tr>
              <tr>
                <td style={{ border: '1px solid #ccc', padding: 8, fontWeight: 'bold' }}>Inspection ID</td>
                <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{selected.inspection_id}</td>
              </tr>
              <tr>
                <td style={{ border: '1px solid #ccc', padding: 8, fontWeight: 'bold' }}>Conflict Analysis</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  <pre style={{ margin: 0, fontSize: 11, maxHeight: 200, overflow: 'auto' }}>{JSON.stringify(selected.conflict_analysis, null, 2)}</pre>
                </td>
              </tr>
            </tbody>
          </table>
          <div style={{ marginTop: 12, display: 'flex', gap: 8 }}>
            <button onClick={() => reconcile(selected)}>Reconcile</button>
            <button onClick={() => setSelected(null)}>Close</button>
          </div>
        </div>
      )}
    </div>
  )
}