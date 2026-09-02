import { useState } from 'react'

type SearchResult = {
  id: string
  type: string
  title: string
  snippet: string
  url: string
  metadata: Record<string, unknown>
  score: number
}

export function SearchAdminPage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [query, setQuery] = useState('')
  const [results, setResults] = useState<SearchResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [page, setPage] = useState(1)
  const [pageSize, setPageSize] = useState(20)
  const [total, setTotal] = useState(0)

  async function search() {
    if (!tenantId || !orgId || !query.trim()) return
    setLoading(true)
    setError(null)
    try {
      const params = new URLSearchParams({
        q: query,
        page: String(page),
        page_size: String(pageSize),
      })
      const r = await fetch(`/api/v1/search/?${params}`, {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setResults(data.results as SearchResult[])
      setTotal(data.total ?? data.results?.length ?? 0)
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  function handleSearch() {
    setPage(1)
    search()
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Search Admin</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <div style={{ flex: 1, minWidth: 300 }}>
          <label style={{ display: 'block', marginBottom: 4 }}>Search Query</label>
          <input
            placeholder="Enter search terms..."
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={e => e.key === 'Enter' && handleSearch()}
            style={{ width: '100%', padding: 8 }}
          />
        </div>
        <div style={{ display: 'flex', gap: 8 }}>
          <label>
            Page Size:
            <select value={pageSize} onChange={e => { setPageSize(Number(e.target.value)); setPage(1); search() }} style={{ marginLeft: 8 }}>
              <option value="10">10</option>
              <option value="20">20</option>
              <option value="50">50</option>
              <option value="100">100</option>
            </select>
          </label>
        </div>
        <button onClick={handleSearch} disabled={!tenantId || !orgId || !query.trim() || loading}>
          {loading ? 'Searching...' : 'Search'}
        </button>
      </div>

      {loading && <p>Searching...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      {results.length > 0 && (
        <>
          <p>Showing {results.length} of {total} results</p>
          <table style={{ borderCollapse: 'collapse', width: '100%' }}>
            <thead>
              <tr>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Type</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Title</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Snippet</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Score</th>
                <th style={{ border: '1px solid #ccc', padding: 8 }}>Metadata</th>
              </tr>
            </thead>
            <tbody>
              {results.map(r => (
                <tr key={r.id}>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>{r.type}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>
                    <a href={r.url} target="_blank" rel="noopener noreferrer">{r.title}</a>
                  </td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>{r.snippet}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>{r.score.toFixed(4)}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>
                    <pre style={{ margin: 0, fontSize: 11, maxHeight: 100, overflow: 'auto' }}>{JSON.stringify(r.metadata, null, 2)}</pre>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          <div style={{ marginTop: 16, display: 'flex', gap: 8, alignItems: 'center' }}>
            <button onClick={() => { setPage(p => p - 1); search() }} disabled={page <= 1 || loading}>Previous</button>
            <span>Page {page}</span>
            <button onClick={() => { setPage(p => p + 1); search() }} disabled={page * pageSize >= total || loading}>Next</button>
          </div>
        </>
      )}

      {results.length === 0 && !loading && query.trim() && <p>No results found.</p>}
    </div>
  )
}