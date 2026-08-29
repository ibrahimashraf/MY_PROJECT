import { useState } from 'react'
import './App.css'

type EvidenceRef = { kind: string; id: string; label: string }
type Signal = {
  id: string
  title: string
  zone: 'MONITORING' | 'REGULATION' | 'NDT_DEFECT' | 'LIFTING_DEFECT'
  confidence: number
  rationale: string
  evidenceRefs: EvidenceRef[]
  limitations: string[]
  blocking: false
  model: { provider: string; model: string; version: string }
  reasoningTrace: string[]
}

type Health = {
  baseline: string
  candidate: string
  health: 'healthy' | 'degraded' | 'unknown'
  trafficPercent: number
  rollbackReady: boolean
}

const MOCK_HEALTH: Health = {
  baseline: 'v1.26.5',
  candidate: 'v1.27.0-rc1',
  health: 'healthy',
  trafficPercent: 12,
  rollbackReady: true,
}

const MOCK_SIGNALS: Signal[] = [
  {
    id: 'SIG-001',
    title: 'Calibration due within 14 days — 3 assets',
    zone: 'MONITORING',
    confidence: 0.82,
    rationale: 'Deterministic lens: calibration expiry within 14d for assets linked to active work orders; no primary state mutated.',
    evidenceRefs: [
      { kind: 'asset', id: 'EQ-4401', label: 'asset EQ-4401' },
      { kind: 'work_order', id: 'WO-2026-08-1201', label: 'WO-2026-08-1201' },
    ],
    limitations: ['Advisory only — does not extend calibration or authorize use.', 'Requires human confirmation.'],
    blocking: false,
    model: { provider: 'local-mock', model: 'deterministic-lens', version: 'v1' },
    reasoningTrace: [
      'Lens: calibration-expiry (deterministic)',
      'Inputs: asset calibration records + work-order scope — versioned contract v1',
      'Output normalized through internal/advisory — confidence, rationale, evidenceRefs, limitations, blocking=false',
    ],
  },
  {
    id: 'SIG-002',
    title: 'Regulation update — lifting standard revision',
    zone: 'REGULATION',
    confidence: 0.64,
    rationale: 'Regulation lens: standard revision delta detected; advisory summary with evidence links, no verdict change.',
    evidenceRefs: [{ kind: 'standard', id: 'STD-LIFT-2026-08', label: 'STD-LIFT-2026-08' }],
    limitations: ['Does not approve inspections or issue certificates.', 'Secondary view model — no secret or prompt exposure.'],
    blocking: false,
    model: { provider: 'local-mock', model: 'regulation-lens', version: 'v1' },
    reasoningTrace: [
      'Lens: regulation-delta (deterministic)',
      'Inputs: Standards Vault snapshot — allow-listed fields only',
      'Degrades gracefully on provider timeout/malformed response (explicit advisory error, primary workflow unchanged)',
    ],
  },
]

function AdvisoryBoundary() {
  return (
    <div className="banner" role="note" aria-label="Advisory boundary">
      <strong>Advisory only</strong> — Quiet Signal cannot set verdicts, approve inspections, issue certificates, validate calibration, authorize requests,
      accept sync transactions, or expand public QR projections. Service failure degrades panels gracefully; primary INTEGIN workflows are unchanged.
      <span className="contract">Contract v1 · all INTEGIN inspection types & systems (except AI-free) · zones: any advisory (NDT_DEFECT, LIFTING_DEFECT, MONITORING, …) · AI-free rejected · <code>blocking=false</code> · observations only, no hold/keep/edit</span>
    </div>
  )
}

function HealthPanel({ h }: { h: Health }) {
  return (
    <section className="panel health" aria-label="Health and canary">
      <h2>Health / Canary</h2>
      <dl>
        <dt>Baseline</dt><dd>{h.baseline}</dd>
        <dt>Candidate</dt><dd>{h.candidate}</dd>
        <dt>Health</dt><dd className={`health-${h.health}`}>{h.health}</dd>
        <dt>Traffic</dt><dd>{h.trafficPercent}%</dd>
        <dt>Rollback</dt><dd>{h.rollbackReady ? 'ready' : 'not ready'}</dd>
      </dl>
      <p className="meta">Injectable transport · explicit env config · no embedded credentials · integration tests opt-in or mocked</p>
    </section>
  )
}

function PhotoObservationsPanel() {
  const [photos, setPhotos] = useState<File[]>([])
  const [obs, setObs] = useState<Signal[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [msg, setMsg] = useState<string | null>(null)
  const [zone, setZone] = useState('NDT_DEFECT')
  const [lens, setLens] = useState('NDT_MPI')
  const [threshold, setThreshold] = useState(0.6)
  const [audit, setAudit] = useState<Record<string, { suggested: number; attached: number; dismissed: number }>>({})
  const apiUrl = (import.meta.env.VITE_ADVISORY_API_URL as string | undefined) ?? ''

  function bumpAudit(l: string, field: 'suggested' | 'attached' | 'dismissed') {
    setAudit((a) => ({ ...a, [l]: { suggested: a[l]?.suggested ?? 0, attached: a[l]?.attached ?? 0, dismissed: a[l]?.dismissed ?? 0, [field]: (a[l]?.[field] ?? 0) + 1 } }))
  }

  async function getObservations() {
    if (photos.length === 0) { setMsg('Select at least one photo.'); return }
    setLoading(true); setMsg(null); setObs(null)
    const ids = photos.map((p) => p.name || 'img-1')
    const effectiveLens = lens.trim() || (zone === 'NDT_DEFECT' ? 'NDT_MPI' : 'LIFTING_WIRE_ROPE')
    const mockFor = (z: string, l: string): Signal => ({
      id: `OBS-${z}-001`, title: `${l} — observation`, zone: z as Signal['zone'], confidence: 0.70,
      rationale: `${l} pattern observed in ${ids.join(', ')} — advisory, requires human confirmation with procedure. Generic: any current or future INTEGIN inspection type is supported via the default lens.`,
      evidenceRefs: ids.map((id) => ({ kind: 'image', id, label: `image ${id}` })), limitations: ['Advisory only — does not set verdict.', 'Requires human confirmation.'], blocking: false,
      model: { provider: 'python-deterministic', model: l, version: 'v1' },
      reasoningTrace: [`Lens: ${l} (deterministic, generic)`, `Inputs: ${ids.length} image(s) + procedure — versioned contract v1`, 'Output: observations[] blocking=false, no hold/keep/edit — any lens via default'],
    })
    bumpAudit(effectiveLens, 'suggested')
    if (!apiUrl) { setTimeout(() => { setObs([mockFor(zone, effectiveLens)]); setLoading(false) }, 400); return }
    try {
      const r = await fetch(`${apiUrl.replace(/\/$/, '')}/v1/advisory`, {
        method: 'POST', headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ version: 'v1', tenant_id: 'tenant-1', zone, lens: effectiveLens, evidence_refs: ids }),
      })
      if (!r.ok) throw new Error(`HTTP ${r.status}`)
      const j = await r.json()
      const mapped: Signal = {
        id: `OBS-${Date.now()}`, title: j.title, zone: zone as Signal['zone'], confidence: j.confidence, rationale: j.rationale,
        evidenceRefs: (j.evidence_refs as string[]).map((x) => ({ kind: 'image', id: x, label: `image ${x}` })),
        limitations: j.limitations, blocking: false, model: { provider: j.provider, model: j.model, version: j.prompt_version },
        reasoningTrace: [`Lens: ${j.model} (deterministic)`, `Provider: ${j.provider}`, `blocking=false — observations only, generic for any INTEGIN type`],
      }
      setObs([mapped]); setLoading(false)
    } catch (e) { setMsg(`Advisory error — primary workflow unchanged: ${String(e)}`); setLoading(false) }
  }

  const visibleObs = obs ? obs.filter((s) => s.confidence >= threshold) : null
  const activeLens = lens.trim() || (zone === 'NDT_DEFECT' ? 'NDT_MPI' : 'LIFTING_WIRE_ROPE')
  const auditForLens = audit[activeLens]

  return (
    <section className="panel" aria-label="Inspection photo — AI observations (opt-in)">
      <h2>Inspection photo — AI observations (opt-in)</h2>
      <p className="meta">Inspector chooses per photo: <strong>Upload directly</strong> or <strong>Get AI observations</strong> — observations only, no hold/keep/edit, no verdict change. Generic for any current/future INTEGIN type — no code change for new types.</p>
      <input type="file" accept="image/*" multiple onChange={(e) => { setPhotos(e.target.files ? Array.from(e.target.files) : []); setObs(null); setMsg(null) }} />
      {photos.length > 0 && <p className="meta">{photos.length} photo(s) selected: {photos.map((p) => p.name).join(', ')}</p>}
      <div style={{ display: 'flex', gap: 8, marginTop: 10, flexWrap: 'wrap', alignItems: 'center' }}>
        <label>Zone (any INTEGIN system) <input list="zone-list" value={zone} onChange={(e) => setZone(e.target.value)} placeholder="NDT_DEFECT, LIFTING_DEFECT or any system" style={{ width: 260 }} /><datalist id="zone-list"><option value="NDT_DEFECT" /><option value="LIFTING_DEFECT" /><option value="MONITORING" /><option value="REGULATION" /><option value="VISUAL_INSPECTION" /><option value="DIMENSIONAL" /></datalist></label>
        <label>Lens (any INTEGIN type) <input list="lens-list" value={lens} onChange={(e) => setLens(e.target.value)} placeholder="PAUT, EDDY_CURRENT, PULSED_EDDY_CURRENT or any NDT" style={{ width: 280 }} /><datalist id="lens-list"><option value="PAUT" /><option value="EDDY_CURRENT" /><option value="PULSED_EDDY_CURRENT" /><option value="TOFD" /><option value="RT" /><option value="NDT_MPI" /><option value="LIFTING_WIRE_ROPE" /></datalist></label>
      </div>
      <div style={{ display: 'flex', gap: 12, marginTop: 10, flexWrap: 'wrap', alignItems: 'center' }}>
        <label>Confidence ≥ {threshold.toFixed(2)} <input type="range" min={0.5} max={0.9} step={0.05} value={threshold} onChange={(e) => setThreshold(parseFloat(e.target.value))} /></label>
        {auditForLens && <span className="meta" style={{ border: '1px solid var(--border)', borderRadius: 999, padding: '2px 8px' }}>Audit {activeLens}: suggested {auditForLens.suggested} · attached {auditForLens.attached} · dismissed {auditForLens.dismissed}</span>}
        <span className="meta" style={{ border: '1px solid var(--border)', borderRadius: 6, padding: '2px 6px' }}>Model card: python-deterministic · {activeLens} v1 · blocking=false · limitations: advisory only, requires human confirmation</span>
      </div>
      <div style={{ display: 'flex', gap: 8, marginTop: 10, flexWrap: 'wrap' }}>
        <button type="button" className="trace-toggle" onClick={() => setMsg(photos.length ? `Uploaded directly: ${photos.map((p) => p.name).join(', ')} — no AI call, primary evidence preserved.` : 'Select at least one photo.')}>Upload directly</button>
        <button type="button" className="trace-toggle" onClick={() => getObservations()} disabled={loading}>Get AI observations — generic for any INTEGIN inspection type</button>
      </div>
      {loading && <p className="meta">Loading observations…</p>}
      {msg && <p className="meta" role="status">{msg}</p>}
      {visibleObs && visibleObs.length === 0 && obs && obs.length > 0 && <p className="meta">Filtered by confidence ≥ {threshold.toFixed(2)} — no observations at this threshold. Lower the slider.</p>}
      {visibleObs && visibleObs.map((s) => (
        <div key={s.id}>
          <SignalCard s={s} />
          <div style={{ border: '1px dashed var(--border)', borderRadius: 8, padding: 8, marginTop: 6, fontSize: 12 }}>
            <strong>Bbox (read-only)</strong> — mock overlay for {s.evidenceRefs.map((e) => e.id).join(', ')} at {Math.round(s.confidence * 100)}% (threshold {Math.round(threshold * 100)}%) — visual evidence link, not a verdict.
            <div style={{ height: 60, background: '#f9fafb', borderRadius: 6, marginTop: 6, display: 'flex', alignItems: 'center', justifyContent: 'center', color: 'var(--muted)' }}>[bbox heatmap placeholder]</div>
          </div>
        </div>
      ))}
      {visibleObs && visibleObs.length > 0 && <div style={{ display: 'flex', gap: 8, marginTop: 8 }}><button type="button" className="trace-toggle" onClick={() => { bumpAudit(activeLens, 'attached'); setMsg('Attached as supporting note — primary verdict still required.') }}>Attach as supporting note</button><button type="button" className="trace-toggle" onClick={() => { bumpAudit(activeLens, 'dismissed'); setObs(null); setMsg('Dismissed — observation discarded, no keep/edit.') }}>Dismiss</button></div>}
    </section>
  )
}

function SignalCard({ s }: { s: Signal }) {
  const [open, setOpen] = useState(false)
  return (
    <article className="signal" aria-label={`Signal ${s.id}`}>
      <header>
        <span className="zone">{s.zone}</span>
        <h3>{s.title}</h3>
        <span className="confidence" title="confidence">{Math.round(s.confidence * 100)}%</span>
      </header>
      <p className="rationale">{s.rationale}</p>
      <ul className="evidence" aria-label="Evidence references">
        {s.evidenceRefs.map((e) => (
          <li key={e.id}><span className="kind">{e.kind}</span> {e.label} <code>{e.id}</code></li>
        ))}
      </ul>
      <ul className="limitations">
        {s.limitations.map((l) => <li key={l}>{l}</li>)}
      </ul>
      <div className="model">provider {s.model.provider} · {s.model.model} {s.model.version} · blocking={String(s.blocking)}</div>
      <button type="button" className="trace-toggle" onClick={() => setOpen((v) => !v)} aria-expanded={open}>
        {open ? 'Hide' : 'Show'} reasoning trace
      </button>
      {open && (
        <ol className="trace">
          {s.reasoningTrace.map((t) => <li key={t}>{t}</li>)}
        </ol>
      )}
    </article>
  )
}

export default function App() {
  const apiUrl = (import.meta.env.VITE_ADVISORY_API_URL as string | undefined) ?? ''
  return (
    <div className="app">
      <header className="topbar">
        <h1>Quiet Signal — Secondary Advisor</h1>
        <span className="env">API: {apiUrl || 'mock (no live integration)'} · transport injectable</span>
      </header>

      <AdvisoryBoundary />

      <main className="grid">
        <HealthPanel h={MOCK_HEALTH} />
        <section className="panel signals" aria-label="Signals">
          <h2>Evidence-linked signals</h2>
          {MOCK_SIGNALS.map((s) => <SignalCard key={s.id} s={s} />)}
        </section>
      </main>

      <div style={{ marginTop: 16 }}>
        <PhotoObservationsPanel />
      </div>

      <footer className="foot">
        View models expose only approved advisory fields — no secrets, hidden prompts, private credentials, or primary decision mutation hooks.
      </footer>
    </div>
  )
}
