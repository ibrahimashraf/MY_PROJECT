import { useState, useEffect, useRef } from 'react'
import {
  saveInspection,
  listPendingSync,
  generateInspectionID,
  type InspectionRecord,
} from './db'
import { syncPendingInspections, registerSyncOnReconnect } from './sync'

type InspectionForm = {
  work_order_id: string
  asset_id: string
  equipment_type: string
  answers: Record<string, string>
}

export function FieldApp() {
  const [mode, setMode] = useState<'list' | 'new' | 'edit'>('list')
  const [inspections, setInspections] = useState<InspectionRecord[]>([])
  const [currentId, setCurrentId] = useState<string | null>(null)
  const [tenantId, setTenantId] = useState('tenant-1')
  const [orgId, setOrgId] = useState('org-1')
  const [form, setForm] = useState<InspectionForm>({
    work_order_id: '',
    asset_id: '',
    equipment_type: '',
    answers: {},
  })
  const [syncing, setSyncing] = useState(false)
  const [syncResult, setSyncResult] = useState<string | null>(null)
  const [online, setOnline] = useState(navigator.onLine)
  const photoInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    const cleanup = registerSyncOnReconnect(tenantId, orgId)
    window.addEventListener('online', () => setOnline(true))
    window.addEventListener('offline', () => setOnline(false))
    loadInspections()
    return () => {
      cleanup()
      window.removeEventListener('online', () => setOnline(true))
      window.removeEventListener('offline', () => setOnline(false))
    }
  }, [tenantId, orgId])

  async function loadInspections() {
    const pending = await listPendingSync()
    setInspections(pending)
  }

  async function handleSave() {
    const id = currentId || generateInspectionID()
    const now = new Date().toISOString()
    const inspection: InspectionRecord = {
      id,
      tenant_id: 'tenant-1',
      organization_id: 'org-1',
      work_order_id: form.work_order_id,
      asset_id: form.asset_id,
      equipment_type: form.equipment_type,
      status: 'COMPLETED',
      synced: 0,
      answers: form.answers,
      photos: [],
      created_at: currentId ? inspections.find(i => i.id === currentId)?.created_at || now : now,
      updated_at: now,
    }
    await saveInspection(inspection)
    setMode('list')
    setCurrentId(null)
    setForm({ work_order_id: '', asset_id: '', equipment_type: '', answers: {} })
    await loadInspections()
  }

  async function handleSync() {
    setSyncing(true)
    setSyncResult(null)
    const result = await syncPendingInspections(tenantId, orgId)
    setSyncResult(`Synced: ${result.synced}, Failed: ${result.failed}`)
    setSyncing(false)
    await loadInspections()
  }

  function handlePhotoCapture() {
    photoInputRef.current?.click()
  }

  function handlePhotoChange(e: React.ChangeEvent<HTMLInputElement>) {
    const files = e.target.files
    if (files) {
      const names = Array.from(files).map(f => f.name)
      setForm({ ...form, answers: { ...form.answers, photos: names.join(',') } })
    }
  }

  if (mode === 'new' || mode === 'edit') {
    return (
      <div style={{ padding: 20, maxWidth: 600 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2>{mode === 'new' ? 'New Inspection' : 'Edit Inspection'}</h2>
          <span style={{ color: online ? 'green' : 'orange', fontWeight: 600 }}>
            {online ? 'ONLINE' : 'OFFLINE'}
          </span>
        </div>

        <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
          <label>
            Work Order ID
            <input
              value={form.work_order_id}
              onChange={e => setForm({ ...form, work_order_id: e.target.value })}
              style={{ display: 'block', width: '100%', padding: 8, marginTop: 4 }}
              placeholder="WO-2026-001"
            />
          </label>
          <label>
            Asset ID
            <input
              value={form.asset_id}
              onChange={e => setForm({ ...form, asset_id: e.target.value })}
              style={{ display: 'block', width: '100%', padding: 8, marginTop: 4 }}
              placeholder="EQ-4401"
            />
          </label>
          <label>
            Equipment Type
            <input
              value={form.equipment_type}
              onChange={e => setForm({ ...form, equipment_type: e.target.value })}
              style={{ display: 'block', width: '100%', padding: 8, marginTop: 4 }}
              placeholder="CHAIN sling, WIRE ROPE, etc."
            />
          </label>

          <h3>Inspection Checklist</h3>
          {['visual_defects', 'wear_measurements', 'markings_labels', 'certification_status', 'overall_condition'].map(q => (
            <label key={q}>
              {q.replace(/_/g, ' ').replace(/\b\w/g, l => l.toUpperCase())}
              <select
                value={form.answers[q] || ''}
                onChange={e => setForm({ ...form, answers: { ...form.answers, [q]: e.target.value } })}
                style={{ display: 'block', width: '100%', padding: 8, marginTop: 4 }}
              >
                <option value="">-- Select --</option>
                <option value="PASS">PASS</option>
                <option value="FAIL">FAIL</option>
                <option value="N/A">N/A</option>
              </select>
            </label>
          ))}

          <div>
            <input
              ref={photoInputRef}
              type="file"
              accept="image/*"
              capture="environment"
              multiple
              style={{ display: 'none' }}
              onChange={handlePhotoChange}
            />
            <button type="button" onClick={handlePhotoCapture} style={{ padding: '8px 16px' }}>
              📷 Capture Photo
            </button>
          </div>

          <div style={{ display: 'flex', gap: 8 }}>
            <button onClick={handleSave} style={{ padding: '8px 16px', background: '#2563eb', color: 'white', border: 'none', borderRadius: 4 }}>
              Save Locally
            </button>
            <button onClick={() => { setMode('list'); setCurrentId(null) }} style={{ padding: '8px 16px' }}>
              Cancel
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div style={{ padding: 20 }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <h2>Field Inspections</h2>
        <div style={{ display: 'flex', gap: 12, alignItems: 'center' }}>
          <input placeholder="Tenant" value={tenantId} onChange={e => setTenantId(e.target.value)} style={{ width: 120, padding: 4 }} />
          <input placeholder="Org" value={orgId} onChange={e => setOrgId(e.target.value)} style={{ width: 120, padding: 4 }} />
          <span style={{ color: online ? 'green' : 'orange', fontWeight: 600 }}>
            {online ? 'ONLINE' : 'OFFLINE'}
          </span>
          <button
            onClick={handleSync}
            disabled={syncing || !online}
            style={{ padding: '6px 12px', background: syncing ? '#ccc' : '#16a34a', color: 'white', border: 'none', borderRadius: 4 }}
          >
            {syncing ? 'Syncing...' : 'Sync Now'}
          </button>
        </div>
      </div>

      {syncResult && <p style={{ color: '#16a34a' }}>{syncResult}</p>}

      <button
        onClick={() => { setMode('new'); setCurrentId(null); setForm({ work_order_id: '', asset_id: '', equipment_type: '', answers: {} }) }}
        style={{ padding: '8px 16px', marginBottom: 16, background: '#2563eb', color: 'white', border: 'none', borderRadius: 4 }}
      >
        + New Inspection
      </button>

      <p style={{ color: '#666' }}>{inspections.length} pending sync</p>

      <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
        {inspections.map(insp => (
          <div key={insp.id} style={{ border: '1px solid #ddd', borderRadius: 8, padding: 12 }}>
            <div style={{ display: 'flex', justifyContent: 'space-between' }}>
              <strong>{insp.asset_id}</strong>
              <span style={{ color: insp.synced ? 'green' : 'orange', fontSize: 12 }}>
                {insp.synced ? 'SYNCED' : 'PENDING'}
              </span>
            </div>
            <div style={{ fontSize: 12, color: '#666' }}>
              {insp.equipment_type} · {insp.work_order_id} · {new Date(insp.updated_at).toLocaleString()}
            </div>
            <div style={{ marginTop: 8, display: 'flex', gap: 4 }}>
              {Object.entries(insp.answers).filter(([k]) => k !== 'photos').map(([k, v]) => (
                <span key={k} style={{
                  padding: '2px 6px', borderRadius: 4, fontSize: 11,
                  background: v === 'PASS' ? '#dcfce7' : v === 'FAIL' ? '#fee2e2' : '#f3f4f6',
                }}>
                  {k.replace(/_/g, ' ')}: {String(v)}
                </span>
              ))}
            </div>
          </div>
        ))}
      </div>
    </div>
  )
}
