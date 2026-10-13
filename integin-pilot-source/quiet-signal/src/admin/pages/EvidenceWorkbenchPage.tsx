import { useState } from 'react'

export type EvidenceItem = {
  evidence_id: string
  object_key: string
  content_type: string
  capture_timestamp: string
  plaintext_sha256: string
  ciphertext_sha256: string
  inspection_id: string
  legal_hold: boolean
  encryption?: {
    key_id: string
    algorithm: string
  }
}

export function EvidenceWorkbenchPage() {
  const [tenantId, setTenantId] = useState('')
  const [orgId, setOrgId] = useState('')
  const [items, setItems] = useState<EvidenceItem[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  async function fetchEvidence() {
    if (!tenantId || !orgId) return
    setLoading(true)
    setError(null)
    try {
      const r = await fetch('/api/v1/evidence', {
        headers: { 'X-Tenant-ID': tenantId, 'X-Organization-ID': orgId },
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const data = await r.json()
      setItems((data.evidence ?? data) as EvidenceItem[])
    } catch (e) {
      setError(String(e))
    } finally {
      setLoading(false)
    }
  }

  function integrityValid(item: EvidenceItem): boolean {
    const dual = !!item.plaintext_sha256 && !!item.ciphertext_sha256 && !item.ciphertext_sha256.includes(item.plaintext_sha256) && !item.plaintext_sha256.includes(item.ciphertext_sha256)
    const crypto = !!item.encryption?.key_id && !!item.encryption?.algorithm
    return dual && crypto
  }

  function formatDate(dateStr: string) {
    return new Date(dateStr).toLocaleString()
  }

  return (
    <div style={{ padding: 20 }}>
      <h2>Evidence Workbench</h2>
      <div style={{ display: 'flex', gap: 12, marginBottom: 20, flexWrap: 'wrap', alignItems: 'flex-end' }}>
        <input placeholder="Tenant ID" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 200 }} />
        <input placeholder="Organization ID" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 200 }} />
        <button onClick={fetchEvidence} disabled={!tenantId || !orgId || loading}>
          {loading ? 'Loading...' : 'Load Evidence'}
        </button>
      </div>

      {error && <p style={{ color: 'red' }}>Error: {error}</p>}
      <p style={{ fontSize: 11, color: '#6b7280' }}>
        Privacy: only digests and metadata are shown. Plaintext evidence bytes are never rendered.
      </p>

      {items.length > 0 && (
        <table style={{ borderCollapse: 'collapse', width: '100%' }}>
          <thead>
            <tr>
              <th style={{ border: '1px solid #ccc', padding: 8, textAlign: 'left' }}>Evidence ID</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Object Key</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Content Type</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Captured At</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Plaintext SHA-256</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Ciphertext SHA-256</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Inspection</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Integrity</th>
              <th style={{ border: '1px solid #ccc', padding: 8 }}>Legal Hold</th>
            </tr>
          </thead>
          <tbody>
            {items.map(item => {
              const valid = integrityValid(item)
              return (
                <tr key={item.evidence_id}>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{item.evidence_id}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 11, fontFamily: 'monospace' }}>{item.object_key}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{item.content_type}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12, whiteSpace: 'nowrap' }}>{formatDate(item.capture_timestamp)}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 10, fontFamily: 'monospace' }}>{item.plaintext_sha256}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 10, fontFamily: 'monospace' }}>{item.ciphertext_sha256}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8, fontSize: 12 }}>{item.inspection_id}</td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>
                    <span
                      style={{
                        display: 'inline-block',
                        padding: '2px 8px',
                        borderRadius: 999,
                        fontSize: 11,
                        fontWeight: 700,
                        background: valid ? '#dcfce7' : '#fee2e2',
                        color: valid ? '#059669' : '#b91c1c',
                      }}
                    >
                      {valid ? 'VALID' : 'INVALID'}
                    </span>
                  </td>
                  <td style={{ border: '1px solid #ccc', padding: 8 }}>
                    {item.legal_hold ? (
                      <span style={{ display: 'inline-block', padding: '2px 8px', borderRadius: 999, fontSize: 11, fontWeight: 700, background: '#fef3c7', color: '#b45309' }}>
                        ON HOLD
                      </span>
                    ) : (
                      <span>—</span>
                    )}
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}

      {items.length === 0 && !loading && <p>No evidence items found.</p>}
    </div>
  )
}