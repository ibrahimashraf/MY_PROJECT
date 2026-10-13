"""Evaluate remediation pipelines against the damage classes in the corpus.

Answers which pipeline suits which document, measured rather than asserted:

  clean      native text is readable      -> no remediation
  fragmented text layer is glyph-per-line  -> redact + write repaired text back
  image_only no text layer at all          -> ocrmypdf
  mixed      some pages broken, some not   -> per-page routing, ocrmypdf for the rest

For each sampled document it measures, before and after: fragmented-line ratio,
recoverable numeric limits, vector paths, raster images, byte size, and wall
clock. Pages are capped so a whole-corpus run stays feasible; the cap is
reported, not hidden.

    evaluate_remediation.py --corpus D:\\engineering_library --out <dir> --per-class 3
"""

from __future__ import annotations

import argparse
import json
import logging
import pathlib
import subprocess
import sys
import time

import pymupdf

from text_extract import (
    MIN_PAGE_CHARS,
    _numeric_tokens,
    desmear,
    extract_page,
    fragment_ratio,
    normalise,
    ocr_word_boxes,
)

sys.path.insert(0, str(pathlib.Path(__file__).parent))
from build_truthset import VALUE_UNIT, plausible  # noqa: E402


def classify(doc, cap: int) -> str:
    """Damage class from measured fragmentation, not from source routing.

    Routing labels are unreliable for this: a document whose first pages are
    mostly blank or header-only has no "native" pages and gets filed as
    image_only even when its body text is glyph-fragmented. Classification here
    is decided by what the text layer actually looks like.
    """
    good = fragmented = blank = 0
    for index in range(min(doc.page_count, cap)):
        try:
            text = doc[index].get_text()
        except Exception:
            continue
        if len(text.strip()) < MIN_PAGE_CHARS:
            blank += 1
        elif fragment_ratio(text) >= 0.25:
            fragmented += 1
        else:
            good += 1
    readable = good + fragmented
    if readable == 0:
        return "image_only"
    if fragmented == 0:
        return "clean"
    if good == 0:
        return "fragmented"
    return "mixed"


def measure(doc, page_numbers: list[int]) -> dict:
    """Quality of the pages under test, as they currently stand."""
    frag_lines = frag_total = 0
    chars = 0
    limits: set[str] = set()
    vectors = images = 0
    for number in page_numbers:
        try:
            page = doc[number - 1]
            text = page.get_text()
        except Exception:
            continue
        lines = [ln for ln in text.splitlines() if ln.strip()]
        frag_lines += sum(1 for ln in lines if len(ln.strip()) == 1)
        frag_total += len(lines)
        chars += len(text.strip())
        for match in VALUE_UNIT.finditer(text):
            value = match.group(1).lstrip("-+")
            unit = match.group(2).replace(" ", "").lower()
            if plausible(value, unit):
                limits.add(f"{value}{unit}")
        try:
            vectors += len(page.get_drawings())
            images += len(page.get_images(full=True))
        except Exception:
            pass
    return {
        "fragmented_ratio": round(frag_lines / frag_total, 3) if frag_total else 0.0,
        "text_chars": chars,
        "numeric_limits": len(limits),
        "vector_paths": vectors,
        "raster_images": images,
    }


def repair_writeback(doc, page_numbers: list[int]) -> int:
    """Strip the broken glyph layer, write the geometry-repaired text back."""
    rewritten = 0
    for number in page_numbers:
        try:
            page = doc[number - 1]
        except Exception:
            continue
        text = normalise(desmear(_repair_text(page)))
        if len(text.strip()) < MIN_PAGE_CHARS:
            continue
        page.add_redact_annot(page.rect)
        try:
            page.apply_redactions(
                images=pymupdf.PDF_REDACT_IMAGE_NONE,
                graphics=pymupdf.PDF_REDACT_LINE_ART_NONE,
                text=pymupdf.PDF_REDACT_TEXT_REMOVE,
            )
        except Exception:
            continue
        y = 40.0
        for line in [ln for ln in text.splitlines() if ln.strip()]:
            page.insert_text((40, y), line[:120], fontsize=6, render_mode=3)
            y += 8
            if y > page.rect.y1 - 40:
                break
        rewritten += 1
    return rewritten


def _repair_text(page) -> str:
    from text_extract import repair

    return repair(page)


def run_ocrmypdf(source: pathlib.Path, target: pathlib.Path, jobs: int) -> tuple[bool, str]:
    env_path = r"C:\Program Files\Tesseract-OCR"
    # --skip-text, not the default: ocrmypdf aborts the whole file when page 1
    # carries any text at all, and image-only scans here routinely carry a
    # running header or a watermark that our 60-character threshold ignores.
    command = [
        sys.executable, "-m", "ocrmypdf", "--output-type", "pdf",
        "--skip-text", "-j", str(jobs), "--quiet", str(source), str(target),
    ]
    try:
        proc = subprocess.run(
            command, capture_output=True, text=True, check=False,
            env={**__import__("os").environ, "PATH": env_path + ";" + __import__("os").environ.get("PATH", "")},
            timeout=1800,
        )
    except subprocess.TimeoutExpired:
        return False, "timeout"
    if proc.returncode != 0:
        return False, (proc.stderr or "").strip().splitlines()[-1:] or ["failed"]
    return True, ""


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", required=True, type=pathlib.Path)
    parser.add_argument("--out", required=True, type=pathlib.Path)
    parser.add_argument("--per-class", type=int, default=3)
    parser.add_argument("--page-cap", type=int, default=12)
    parser.add_argument("--jobs", type=int, default=4)
    args = parser.parse_args()

    logging.disable(logging.CRITICAL)
    args.out.mkdir(parents=True, exist_ok=True)
    work = args.out / "work"
    work.mkdir(exist_ok=True)

    # Classify the corpus from the audit already produced, so no OCR cost here.
    audit = args.out / "audit.json"
    if not audit.is_file():
        print(f"expected {audit} from audit_extraction.py", file=sys.stderr)
        return 1

    # Candidate pool from the audit (already known to contain damage), but the
    # class itself is measured here rather than inherited from routing labels.
    data = json.loads(audit.read_text(encoding="utf-8"))
    pool = sorted({r["doc"] for r in data["routes"] if "error" not in r})
    buckets: dict[str, list[str]] = {"clean": [], "fragmented": [], "mixed": [], "image_only": []}
    for rel in pool:
        try:
            doc = pymupdf.open(str(args.corpus / rel))
        except Exception:
            continue
        kind = classify(doc, args.page_cap)
        doc.close()
        buckets[kind].append(rel)

    print(f"measured damage classes (pages capped at {args.page_cap}):")
    for kind, docs in sorted(buckets.items()):
        print(f"  {kind:<11} {len(docs):>4} documents")

    results = []
    for kind in ("fragmented", "mixed", "image_only"):
        pool = buckets.get(kind, [])
        if not pool:
            continue
        for rel in pool[: args.per_class]:
            source = args.corpus / rel
            try:
                doc = pymupdf.open(str(source))
            except Exception as exc:
                print(f"  skip {rel}: {exc}", file=sys.stderr)
                continue
            pages = list(range(1, min(doc.page_count, args.page_cap) + 1))
            before = measure(doc, pages)
            doc.close()

            row = {"doc": rel, "class": kind, "pages_tested": len(pages), "before": before}

            if kind in ("fragmented", "mixed"):
                target = work / (rel.replace("/", "__").rsplit(".", 1)[0] + ".repaired.pdf")
                try:
                    doc = pymupdf.open(str(source))
                    start = time.time()
                    rewritten = repair_writeback(doc, pages)
                    doc.save(str(target))
                    doc.close()
                    elapsed = time.time() - start
                    check = pymupdf.open(str(target))
                    row["repair_writeback"] = {
                        **measure(check, pages), "seconds": round(elapsed, 1),
                        "size_mb": round(target.stat().st_size / 1e6, 2),
                        "pages_rewritten": rewritten,
                    }
                    check.close()
                except Exception as exc:
                    row["repair_writeback"] = {"error": str(exc)}

            if kind in ("image_only", "mixed"):
                target = work / (rel.replace("/", "__").rsplit(".", 1)[0] + ".ocrmypdf.pdf")
                start = time.time()
                ok, err = run_ocrmypdf(source, target, args.jobs)
                elapsed = time.time() - start
                if ok:
                    check = pymupdf.open(str(target))
                    row["ocrmypdf"] = {
                        **measure(check, pages), "seconds": round(elapsed, 1),
                        "size_mb": round(target.stat().st_size / 1e6, 2),
                    }
                    check.close()
                else:
                    row["ocrmypdf"] = {"error": err}

            results.append(row)
            print(f"  {kind:<11} {rel.rsplit(chr(92),1)[-1][:52]}")

    out_path = args.out / "evaluation.json"
    out_path.write_text(json.dumps(results, indent=1), encoding="utf-8")
    print(f"\nwrote {out_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())