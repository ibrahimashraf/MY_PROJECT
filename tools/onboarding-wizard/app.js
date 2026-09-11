/* ============================================================
 * Token grammar (defensive parse — documented expected shape)
 *
 *   PREFIX-DISCIPLINE-YEAR-SEQ
 *   APEX-CRN-2026-00001
 *
 *   - PREFIX     [A-Z0-9]{2,12}   issuer/company slug
 *   - DISCIPLINE [A-Z]{2,12}      e.g. CRN (crane), LEE (lifeline)
 *   - YEAR       \d{4}            certificate issue year
 *   - SEQ        \d{4,6}          0-padded sequence
 *
 * Anything else yields no match -> preview hidden, error shown.
 * Mirrors integin-pilot-source CertificatePolicy.PrefixPattern tokens
 * {PREFIX}-{DISCIPLINE}-{YEAR}-{SEQ:n}.
 * ============================================================ */

var TOKEN_RE = /^([A-Z0-9]{2,12})-([A-Z]{2,12})-(\d{4})-(\d{4,6})$/;

var tokenEl = document.getElementById('token');
var errorEl = document.getElementById('error');
var previewEl = document.getElementById('preview');

function parseToken(raw) {
  if (typeof raw !== 'string') return null;
  var m = TOKEN_RE.exec(raw.trim().toUpperCase());
  if (!m) return null;
  return { prefix: m[1], discipline: m[2], year: m[3], seq: m[4] };
}

function renderPreview(t) {
  document.getElementById('certNumber').textContent =
    t.prefix + '-' + t.discipline + '-' + t.year + '-' + t.seq;
  document.getElementById('certPrefix').textContent = t.prefix;
  document.getElementById('certDiscipline').textContent = t.discipline;
  document.getElementById('certYear').textContent = t.year;
  document.getElementById('certSeq').textContent = t.seq;
  previewEl.hidden = false;
  errorEl.hidden = true;
}

function onInput() {
  var t = parseToken(tokenEl.value);
  if (t) {
    renderPreview(t);
  } else {
    previewEl.hidden = true;
    errorEl.hidden = tokenEl.value.trim() === '';
  }
}

tokenEl.addEventListener('input', onInput);
onInput();