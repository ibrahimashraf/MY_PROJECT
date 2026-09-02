import { useState } from 'react'

type ShortLink = {
  code: string
  target_url: string
  created_at: string
  expires_at: string | null
  revoked_at: string | null
  scan_count: number
}

export function ShortLinksPage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [links, setLinks] = useState<ShortLink[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [newTargetUrl, setNewTargetUrl] = useState('')
  const [newTtl, setNewTtl] = useState('')

  async function fetchLinks() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/v1/admin/shortlinks', {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setLinks(data as ShortLink[])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  async function createShortLink() {
    if (!tenantId || !orgId || !newTargetUrl) return
    setCreating(true)
    setError(null)
    try {
      const body: { target_url: string; ttl?: string } = { target_url: newTargetUrl }
      if (newTtl) body.ttl = newTtl
      const r = await fetch('/api/v1/admin/shortlinks', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
        body: JSON.stringify(body),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setLinks([data, ...links])
      setNewTargetUrl('')
      setNewTtl('')
    } catch (e) {
      setError(String(e))
    } finally {
      setCreating(false)
    }
  }

  async function revokeLink(code: string) {
    if (!tenantId || !orgId) return
    if (!confirm(`Revoke short link ${code}?`)) return
    try {
      const r = await fetch(`/api/v1/admin/shortlinks/${code}`, {
        method: 'DELETE',
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      await fetchLinks()
    } catch (e) {
      setError(String(e))
    }
  }

  function formatDate(dateStr: string | null) {
    if (!dateStr) return '—'
    return new Date(dateStr).toLocaleString()
  }

  function getBaseUrl() {
    return window.location.origin
  }

  function getShortUrl(code: string) {
    return `${getBaseUrl()}/s/${code}`
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Short Link Manager</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchLinks} disabled={!tenantId || !orgId || loading}>Load Links</button>
      </div>

      <div style={{ marginBottom: 20, padding: 16, border: '1px solid #ccc', borderRadius: 4 }}>
        <h3>Create Short Link</h3>
        <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', alignItems: 'flex-end' }}>
          <div style={{ flex: 1, minWidth: 300 }}>
            <label style={{ display: 'block', marginBottom: 4 }}>Target URL</label>
            <input
              placeholder="https://example.com/long/path"
              value={newTargetUrl}
              onChange={e => setNewTargetUrl(e.target.value)}
              style={{ width: '100%', padding: 8 }}
            />
          </div>
          <div>
            <label style={{ display: 'block', marginBottom: 4 }}>TTL (e.g., 24h, 7d, 30d)</label>
            <input
              placeholder="Optional"
              value={newTtl}
              onChange={e => setNewTtl(e.target.value)}
              style={{ width: 150, padding: 8 }}
            />
          </div>
          <button onClick={createShortLink} disabled={!newTargetUrl || creating || !tenantId || !orgId}>
            {creating ? 'Creating...' : 'Create'}
          </button>
        </div>
      </div>

      {loading && <p>Loading...</p>}
      {error && <p style={{ color: 'red' }}>Error: {error}</p>}

      {links.length > 0 && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Code</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Short URL</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Target URL</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Created</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Expires</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Revoked</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Scans</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {links.map(link => (
              <tr key={link.code}>
                <td style={{ border: '1px solid #ccc', padding: 8, fontFamily: 'monospace' }}>
                  <code>{link.code}</code>
                </td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  <a href={getShortUrl(link.code)} target="_blank" rel="noopener noreferrer">
                    {getShortUrl(link.code)}
                  </a>
                </td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  <a href={link.target_url} target="_blank" rel="noopener noreferrer" style={{ maxWidth: 300, textOverflow: 'ellipsis', overflow: 'hidden', whiteSpace: 'nowrap', display: 'block' }}>
                    {link.target_url}
                  </a>
                </td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{link.created_at ? new Date(link.created_at).toLocaleString() : '—'}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{formatDate(link.expires_at)}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{link.revoked_at ? '🔴 Revoked' : '🟢 Active'}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>{link.scan_count}</td>
                <td style={{ border: '1px solid #ccc', padding: 8 }}>
                  {!link.revoked_at && (
                    <button
                      onClick={() => revokeLink(link.code)}
                      style={{ background: '#dc3545', color: 'white', border: 'none', padding: '4 8', borderRadius: 3, cursor: 'pointer' }}
                    >
                      Revoke
                    </button>
                  )}
                  {link.revoked_at && <span style={{ color: '#999' }}>Revoked</span>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {links.length === 0 && !loading && <p>No short links found. Create one above.</p>}
    </div>
  )
}