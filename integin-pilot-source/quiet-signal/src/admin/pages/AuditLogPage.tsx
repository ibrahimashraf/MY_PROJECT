import { useState } from 'react'

type AuditLogEntry = {
  id: string
  timestamp: string
  actor_type: string
  actor_id: string
  action: string
  resource_type: string
  resource_id: string
  details: Record<string, unknown>
  ip_address: string | null
  user_agent: string | null
}

export function AuditLogPage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [logs, setLogs] = useState<AuditLogEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(50)
  const [total, setTotal] = useState(0)
  const [filters, setFilters] = useState({
    actor_type: '',
    action: '',
    resource_type: '',
    from: '',
    to: '',
  })

  async function fetchLogs() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({
        page: String(page),
        page_size: String(pageSize),
        ...Object.fromEntries(
          Object.entries(filters).filter(([, v]) => v !== '')
        ),
      })
      const r = await fetch(`/api/v1/audit-log?${params}`, {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setLogs(data.logs as AuditLogEntry[])
      setTotal(data.total ?? data.logs?.length ?? 0)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  function handleFilterChange(key: keyof typeof filters, value: string) {
    setFilters(f => ({ ...f, [key]: value }))
    setPage(1)
  }

  function applyFilters() {
    setPage(1)
    fetchLogs()
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Audit Log</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchLogs} disabled={!tenantId || !orgId || loading}>
          {loading ? 'Loading...' : 'Load Logs'}
        </button>
      </div>

      <div style={{ marginBottom: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4, display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <h3 style={{ margin: 0 }}>Filters</h3>
        <div>
          <label style={{ display: 'block', marginBottom: 4 }}>Actor Type</label>
          <input
            placeholder="e.g., USER, SYSTEM"
            value={filters.actor_type}
            onChange={e => handleFilterChange('actor_type', e.target.value)}
            style={{ width: 180, padding: 8 }}
          />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 4 }}>Action</label>
          <input
            placeholder="e.g., CREATE, UPDATE, DELETE"
            value={filters.action}
            onChange={e => handleFilterChange('action', e.target.value)}
            style={{ width: 180, padding: 8 }}
          />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 4 }}>Resource Type</label>
          <input
            placeholder="e.g., LICENSE, USER, PROJECT"
            value={filters.resource_type}
            onChange={e => handleFilterChange('resource_type', e.target.value)}
            style={{ width: 180, padding: 8 }}
          />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 4 }}>From Date</label>
          <input
            type="datetime-local"
            value={filters.from}
            onChange={e => handleFilterChange('from', e.target.value)}
            style={{ width: 200, padding: 8 }}
          />
        </div>
        <div>
          <label style={{ display: 'block', marginBottom: 4 }}>To Date</label>
          <input
            type="datetime-local"
            value={filters.to}
            onChange={e => handleFilterChange('to', e.target.value)}
            style={{ width: 200, padding: 8 }}
          />
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <button onClick={applyFilters} disabled={loading}>Apply</button>
          <button onClick={() => { setFilters({ actor_type: '', action: '', resource_type: '', from: '', to: '' }); setPage(1); fetchLogs() }} disabled={loading}>Clear</button>
        </div>
        <div style={{ marginLeft: 'auto' }}>
          <label>
            Page Size:
            <select value={pageSize} onChange={e => { setPageSize(Number(e.target.value)); setPage(1); fetchLogs() }} style={{ marginLeft: 8 }}>
              <option value="25">25</option>
              <option value="50">50</option>
              <option value="100">100</option>
              <option value="200">200</option>
            </select>
          </label>
        </div>
      </div>

      {loading && <p>Loading...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      {logs.length > 0 && (
        <>
          <p>Showing {logs.length} of {total} entries</p>
          <table style={{ borderCollapse: 'collapse', width: '100%' }}>
            <thead>
              <tr>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Timestamp</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Actor</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Action</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Resource</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Details</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>IP / UA</th>
              </tr>
            </thead>
            <tbody>
              {logs.map(log => (
                <tr key={log.id}>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12, whiteSpace: 'nowrap' }}>{formatDate(log.timestamp)}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>
                    <span style={{ fontWeight: 'bold' }}>{log.actor_type}</span>: {log.actor_id}
                  </td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{log.action}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>
                    <span style={{ fontWeight: 'bold' }}>{log.resource_type}</span>: {log.resource_id}
                  </td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>
                    <pre style={{ margin: 0, fontSize: 10, maxHeight: 120, overflow: 'auto' }}>{JSON.stringify(log.details, null, 2)}</pre>
                  </td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 11 }}>
                    {log.ip_address && <div>IP: {log.ip_address}</div>}
                    {log.user_agent && <div title={log.user_agent} style={{ maxWidth: 200, textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap' }}>UA: {log.user_agent}</div>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div style={{ marginTop: 16, display: 'flex', gap: 8, alignItems: 'center' }}>
            <button onClick={() => { setPage(p => p - 1); fetchLogs() }} disabled={page <= 1 || loading}>Previous</button>
            <span>Page {page}</span>
            <button onClick={() => { setPage(p => p + 1); fetchLogs() }} disabled={page * pageSize >= total || loading}>Next</button>
          </div>
        </>
      )}

      {logs.length === 0 && !loading && <p>No audit log entries found.</p>}
    </div>
  )
}