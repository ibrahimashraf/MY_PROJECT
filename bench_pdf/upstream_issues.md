# Upstream anymd issues (drafts, with repros from our bench)

## 1. OCR tables drop row labels (accuracy)
**Title:** `OCR table reconstruction drops row labels`

**Body:**
On scanned pages, data tables lose their row-label column while values survive.
Repro: weight/CofG table, Rigging Engineering Basics p26 (scan). anymd output keeps
`TOTALS 35355`, moments `225165.50 103082.75 120198.57`, `CofG 6.37 2.92 3.40`
(all exact) but member rows render as bare value pipes (`|17711.96|0.00|`)
without `H1..H10 / C1.. / B1..` labels. A tesseract-direct pass on the same page
preserves labels, so the glyphs are available — the table builder discards them.
Happy to share the page image + both outputs.

## 2. No OCR language selection
**Title:** `Add --ocr-lang for tesseract (e.g. eng+ara)`

**Body:**
`--ocr/--no-ocr` has no language option. Arabic scans OCR as English gibberish.
`--ocr-lang eng+ara` (passed to tesseract, documented next to `tesseract --list-langs`)
would fix mixed EN/AR rigging manuals. Workaround today: none via CLI.

## 3. No per-page OCR quality signal
**Title:** `Expose OCR confidence per page block`

**Body:**
Downstream pipelines need to gate OCR quality (route low-confidence pages to VLM
fallback). Today the only signal is parsing the text. Request: `<!-- page N
ocr_confidence=0.83 -->` or equivalent in `inspect structure` JSON, sourced from
tesseract word confidences. Our use case: dotted-leader TOC pages score high on
char count but are unusable; confidence would quarantine them automatically
(cf. spike #803 — this composes with a future VLM backend as its router).

## 4. 512MB file cap blocks large scans
**Title:** `Make 512MB file limit configurable / stream large PDFs`

**Body:**
`Rigging_Engineering_Basics` scan (555MB, 440pp) is refused:
`file is 555 MB; the limit is 512 MB`. Workaround is caller-side chunking
(split with pypdf, convert, renumber `<!-- page N -->`), which works but every
consumer reimplements it. Request: `--max-bytes` flag or chunked streaming
internally (pages are processed independently already).
