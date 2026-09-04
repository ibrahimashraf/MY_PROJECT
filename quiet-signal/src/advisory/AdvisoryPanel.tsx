import { useState } from 'react'
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

export function AdvisoryPanel() {
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

      <footer className="foot">
        View models expose only approved advisory fields — no secrets, hidden prompts, private credentials, or primary decision mutation hooks.
      </footer>
    </div>
  )
}
