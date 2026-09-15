import { useEffect, useState } from 'react'
import './AdvisoryPanel.css'

export type EvidenceRef = { kind: string; id: string; label: string }

export type Signal = {
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

export type Health = {
  baseline: string
  candidate: string
  health: 'healthy' | 'degraded' | 'unknown'
  trafficPercent: number
  rollbackReady: boolean
}

export type ApprovedModel = {
  id: string
  provider: string
  model_name: string
  version: string
  status: string
  max_tokens: number
  allowed_zones: string[]
}

export type ExtractionResult = {
  tenant_id: string
  title: string
  authority: string
  effective_date: string
  clauses: string[]
  checklist_refs: string[]
  blocking: false
  extracted_at: string
}

export type AdvisoryPanelProps = {
  tenantId?: string
  orgId?: string
  token?: string
  apiUrl?: string
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
  {
    id: 'SIG-003',
    title: 'NDT Indication Observation — Magnetic Particle Inspection',
    zone: 'NDT_DEFECT',
    confidence: 0.78,
    rationale: 'Visual indication of linear discontinuity observed along weld toe on Boom Section 2. Requires verification by certified Level II inspector.',
    evidenceRefs: [
      { kind: 'evidence_object', id: 'ev-photo-caliper-01', label: 'MPI UV Photo' },
      { kind: 'asset', id: 'asset-crane-rt100', label: 'Liebherr LTM 1100' },
    ],
    limitations: ['Observation only — cannot determine inspection verdict or issue certificate.', 'Non-blocking human review required.'],
    blocking: false,
    model: { provider: 'ai_service', model: 'defect-observation-model', version: 'v1' },
    reasoningTrace: [
      'Lens: NDT_MPI (optical + magnetic particle pattern analysis)',
      'Inputs: encrypted evidence ciphertext digest verified',
      'AI boundary enforced: blocking=false, zero write capability on inspection_record',
    ],
  },
]

function advisoryHeaders(tenantId: string, orgId: string, token: string): Record<string, string> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    'X-Tenant-ID': tenantId,
    'X-Organization-ID': orgId,
  }
  if (token) headers['Authorization'] = `Bearer ${token}`
  return headers
}

async function fetchApprovedModels(
  base: string,
  tenantId: string,
  orgId: string,
  token: string
): Promise<ApprovedModel[]> {
  const res = await fetch(`${base}/api/v1/advisory/models`, {
    headers: advisoryHeaders(tenantId, orgId, token),
  })
  if (!res.ok) throw new Error(`models request failed (HTTP ${res.status})`)
  const body = (await res.json()) as { models?: ApprovedModel[] }
  return body.models ?? []
}

async function extractDocument(
  base: string,
  tenantId: string,
  orgId: string,
  token: string,
  text: string
): Promise<ExtractionResult> {
  const res = await fetch(`${base}/api/v1/advisory/extract`, {
    method: 'POST',
    headers: advisoryHeaders(tenantId, orgId, token),
    body: JSON.stringify({ text, zone: '', blocking: false }),
  })
  if (!res.ok) throw new Error(`extract request failed (HTTP ${res.status})`)
  return (await res.json()) as ExtractionResult
}

function errorMessage(err: unknown): string {
  return err instanceof Error ? err.message : 'unexpected error'
}

export function AdvisoryBoundary() {
  return (
    <div className="banner" role="note" aria-label="Advisory boundary">
      <strong>Advisory only</strong> — Quiet Signal cannot set verdicts, approve inspections, issue certificates, validate calibration, authorize requests,
      accept sync transactions, or expand public QR projections. Service failure degrades panels gracefully; primary INTEGIN workflows are unchanged.
      <span className="contract">Contract v1 · zones: MONITORING, REGULATION, NDT_DEFECT, LIFTING_DEFECT · AI-free zones rejected · <code>blocking=false</code></span>
    </div>
  )
}

export function HealthPanel({ h }: { h: Health }) {
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

export function SignalCard({ s }: { s: Signal }) {
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

export function AdvisoryPanel({ tenantId, orgId, token, apiUrl }: AdvisoryPanelProps) {
  const envUrl = (import.meta.env.VITE_ADVISORY_API_URL as string | undefined) ?? ''
  const envToken = (import.meta.env.VITE_ADVISORY_TOKEN as string | undefined) ?? ''
  const base = apiUrl ?? envUrl
  const auth = token ?? envToken
  const live = Boolean(base && tenantId && orgId)

  const [models, setModels] = useState<ApprovedModel[] | null>(null)
  const [modelsFailed, setModelsFailed] = useState(false)
  const [text, setText] = useState('')
  const [extracting, setExtracting] = useState(false)
  const [extractResult, setExtractResult] = useState<ExtractionResult | null>(null)
  const [extractError, setExtractError] = useState('')

  useEffect(() => {
    if (!base || !tenantId || !orgId) return
    let cancelled = false
    fetchApprovedModels(base, tenantId, orgId, auth)
      .then((loaded) => {
        if (cancelled) return
        setModels(loaded)
        setModelsFailed(false)
      })
      .catch(() => {
        if (!cancelled) setModelsFailed(true)
      })
    return () => {
      cancelled = true
    }
  }, [base, tenantId, orgId, auth])

  async function runExtract() {
    if (!tenantId || !orgId || extracting) return
    setExtracting(true)
    setExtractError('')
    setExtractResult(null)
    try {
      setExtractResult(await extractDocument(base, tenantId, orgId, auth, text))
    } catch (err) {
      setExtractError(errorMessage(err))
    } finally {
      setExtracting(false)
    }
  }

  return (
    <div className="app">
      <header className="topbar">
        <h1>Quiet Signal — Secondary Advisor</h1>
        <span className="env">
          API: {live ? base : 'mock signals (no tenant/org credentials)'} · transport injectable
          {modelsFailed ? ' · backend unreachable — mock signals shown' : ''}
        </span>
      </header>

      <AdvisoryBoundary />

      <main className="grid">
        <aside className="side">
          <HealthPanel h={MOCK_HEALTH} />
          <section className="panel models" aria-label="Approved models">
            <h2>Approved models</h2>
            {!live ? (
              <p className="meta">Mock mode — supply tenantId/orgId and an API URL to load approved models.</p>
            ) : modelsFailed ? (
              <p className="meta">Could not reach advisory backend — showing mock signals instead.</p>
            ) : models === null ? (
              <p className="meta">Loading approved models…</p>
            ) : models.length === 0 ? (
              <p className="meta">No approved models returned.</p>
            ) : (
              <ul>
                {models.map((m) => (
                  <li key={m.id}>
                    <strong>{m.model_name}</strong> {m.provider} {m.version}
                    <small>
                      {m.status} · max {m.max_tokens} tokens · zones:{' '}
                      {m.allowed_zones.length > 0 ? m.allowed_zones.join(', ') : 'any'}
                    </small>
                  </li>
                ))}
              </ul>
            )}
          </section>
        </aside>
        <section className="panel signals" aria-label="Signals">
          <h2>Evidence-linked signals</h2>
          {MOCK_SIGNALS.map((s) => <SignalCard key={s.id} s={s} />)}
        </section>
      </main>

      <section className="panel extract" aria-label="Document extraction test">
        <h2>Document extraction test</h2>
        {live ? (
          <>
            <label className="sr-only" htmlFor="advisory-extract-text">Regulatory text</label>
            <textarea
              id="advisory-extract-text"
              value={text}
              onChange={(e) => setText(e.target.value)}
              placeholder="Paste regulatory text to extract clauses and checklist references (read-only advisory test, blocking=false)."
            />
            <button
              type="button"
              onClick={() => void runExtract()}
              disabled={extracting || text.trim() === ''}
            >
              {extracting ? 'Extracting…' : 'Extract — blocking=false'}
            </button>
            {extractError && <p className="err">{extractError}</p>}
            {extractResult && (
              <div className="out" aria-live="polite">
                <h3>{extractResult.title} <code>{extractResult.authority}</code></h3>
                <dl>
                  <dt>Effective</dt><dd>{extractResult.effective_date}</dd>
                  <dt>Tenant</dt><dd>{extractResult.tenant_id}</dd>
                </dl>
                <h4>Clauses</h4>
                {extractResult.clauses.length > 0 ? (
                  <ul>{extractResult.clauses.map((c) => <li key={c}>{c}</li>)}</ul>
                ) : (
                  <p className="meta">None extracted.</p>
                )}
                <h4>Checklist references</h4>
                {extractResult.checklist_refs.length > 0 ? (
                  <ul>{extractResult.checklist_refs.map((c) => <li key={c}>{c}</li>)}</ul>
                ) : (
                  <p className="meta">None referenced.</p>
                )}
                <p className="meta">blocking={String(extractResult.blocking)} · no primary state mutated · result not persisted</p>
              </div>
            )}
          </>
        ) : (
          <p className="meta">Live extraction requires tenantId and orgId (plus an API URL). Nothing is sent or stored in mock mode.</p>
        )}
      </section>

      <footer className="foot">
        View models expose only approved advisory fields — no secrets, hidden prompts, private credentials, or primary decision mutation hooks.
      </footer>
    </div>
  )
}