"""Corpus-wide audit of text recovery, before any catalog is rewritten.

Answers two open findings from the enhancement review:

  F-06  Is geometry repair sufficient corpus-wide, or does OCR scope stay large?
  F-07  Neither native nor OCR is trustworthy alone -- how often do they agree?

Writes a per-page route map so the extraction run is decided by evidence rather
than by assumption. Read-only with respect to the corpus and the catalog.

    python audit_extraction.py --corpus D:\\engineering_library --out audit.json
"""

from __future__ import annotations

import argparse
import json
import logging
import pathlib
import random
import re
import sys
from collections import Counter

import pymupdf

from text_extract import (
    SOURCE_NONE,
    SOURCE_REPAIRED,
    extract_page,
    ocr_page,
)

# OCR is slow (measured ~18-24s/page). Cross-check a stratified sample so the
# agreement rate is real without a multi-hour run.
SAMPLE_OCR_PER_SOURCE = 12
SEED = 7121


def collect(corpus: pathlib.Path, page_cap: int) -> list[dict]:
    """Route every page to native / repaired / none without invoking OCR."""
    routes: list[dict] = []
    files = sorted(corpus.rglob("*.pdf"))
    for path in files:
        # Relative path, not just the name: the corpus is nested by standards
        # body, so a bare filename cannot be reopened later.
        rel = path.relative_to(corpus).as_posix()
        try:
            doc = pymupdf.open(str(path))
        except Exception as exc:  # one bad PDF must not end the sweep
            routes.append({"doc": rel, "error": f"open_failed: {exc}"})
            continue

        scanned = min(doc.page_count, page_cap)
        for index in range(scanned):
            try:
                page = doc[index]
                result = extract_page(page)
                routes.append(
                    {
                        "doc": rel,
                        "page": index + 1,
                        "source": result.source,
                        "fragment_ratio": round(result.fragment_ratio, 3),
                        "chars": len(result.text.strip()),
                        "drawings": count_drawings(page),
                        "images": count_images(page),
                    }
                )
            except Exception as exc:
                routes.append(
                    {
                        "doc": rel,
                        "page": index + 1,
                        "source": "error",
                        "error": str(exc),
                    }
                )
        doc.close()
    return routes


def count_drawings(page) -> int:
    """Real geometry only: glyph-sized paths are outlined text, not figures."""
    try:
        drawings = page.get_drawings()
    except Exception:
        return 0
    return sum(
        1 for d in drawings if not (d["rect"].width < 12 and d["rect"].height < 12)
    )


def count_images(page) -> int:
    try:
        return len(page.get_images(full=True))
    except Exception:
        return 0


def sample_for_crosscheck(routes: list[dict], per_source: int) -> list[dict]:
    rng = random.Random(SEED)
    picked: list[dict] = []
    by_source: dict[str, list[dict]] = {}
    for route in routes:
        if route.get("source") in (SOURCE_REPAIRED, SOURCE_NONE):
            by_source.setdefault(route["source"], []).append(route)
    for source, group in sorted(by_source.items()):
        picked.extend(rng.sample(group, min(per_source, len(group))))
    return picked


def _normalise(text: str) -> str:
    """Collapse formatting so agreement measures content, not line breaks."""
    return re.sub(r"\s+", " ", text).strip().lower()


def _numeric_tokens(text: str) -> set[str]:
    return {m.group(0) for m in re.finditer(r"-?\d+(?:\.\d+)?", _normalise(text))}


def crosscheck(corpus: pathlib.Path, routes: list[dict], per_source: int) -> list[dict]:
    """Run OCR beside the chosen route and record agreement.

    Compares normalised text and the numeric token set. A raw string compare
    would report every line-wrap difference as a disagreement.
    """
    try:
        from rapidocr_onnxruntime import RapidOCR
    except ImportError:
        print("rapidocr-onnxruntime not installed; skipping crosscheck", file=sys.stderr)
        return []

    engine = RapidOCR()
    results = []
    for route in sample_for_crosscheck(routes, per_source):
        path = corpus / route["doc"]
        try:
            doc = pymupdf.open(str(path))
            page = doc[route["page"] - 1]
            native = extract_page(page)
            ocr = ocr_page(page, engine)
            doc.close()
        except Exception as exc:
            results.append({**route, "error": str(exc)})
            continue

        native_nums = _numeric_tokens(native.text)
        ocr_nums = _numeric_tokens(ocr.text)
        shared = native_nums | ocr_nums
        results.append(
            {
                **route,
                "ocr_chars": len(ocr.text.strip()),
                "ocr_note": ocr.notes[0] if ocr.notes else "",
                "text_agrees": _normalise(native.text) == _normalise(ocr.text),
                "numeric_recall": (
                    len(native_nums & ocr_nums) / len(native_nums)
                    if native_nums
                    else None
                ),
                "numeric_precision": (
                    len(ocr_nums & native_nums) / len(ocr_nums)
                    if ocr_nums
                    else None
                ),
                "numeric_disputed": sorted(native_nums ^ ocr_nums),
                "shared_numeric_count": len(native_nums & ocr_nums),
            }
        )
        print(f"  {route['doc']} p{route['page']} ({route['source']}) done")
    return results


def build_figure_manifest(routes: list[dict]) -> list[dict]:
    """Locate pages whose engineering content is graphical.

    Vector paths and embedded images are invisible to get_text(), so a rigging
    arrangement or load chart contributes nothing to the text catalog. This
    records that the figure exists and where, so it is locatable rather than
    silently dropped. Deliberately a manifest only: rendering or digitising the
    figures has no consumer yet.
    """
    manifest = []
    for route in routes:
        if "error" in route:
            continue
        drawings = route.get("drawings", 0)
        images = route.get("images", 0)
        chars = route.get("chars", 0)
        if drawings < 30 and images == 0:
            continue
        # A page carrying both geometry and little text is a figure. Most
        # image-only hits are publisher logos on prose pages, so they are kept
        # for traceability but marked low-signal.
        high_signal = (drawings >= 30 and chars < 300) or (images > 0 and chars < 300)
        manifest.append(
            {
                "figure_id": f"{route['doc']}#p{route['page']}",
                "doc": route["doc"],
                "page": route["page"],
                "vector_paths": drawings,
                "raster_images": images,
                "text_chars": chars,
                "high_signal": high_signal,
                "graphic_only": chars < 60,
            }
        )
    return manifest


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", required=True, type=pathlib.Path)
    parser.add_argument("--out", required=True, type=pathlib.Path)
    parser.add_argument(
        "--page-cap", type=int, default=30, help="max pages per document"
    )
    parser.add_argument(
        "--ocr-sample", type=int, default=SAMPLE_OCR_PER_SOURCE
    )
    parser.add_argument("--skip-crosscheck", action="store_true")
    args = parser.parse_args()

    logging.disable(logging.CRITICAL)

    if not args.corpus.is_dir():
        print(f"corpus not found: {args.corpus}", file=sys.stderr)
        return 1

    routes = collect(args.corpus, args.page_cap)
    errors = [r for r in routes if "error" in r]
    counted = [r for r in routes if "error" not in r]

    by_source = Counter(r["source"] for r in counted)
    total = len(counted)
    figures = build_figure_manifest(counted)
    high_signal = [f for f in figures if f["high_signal"]]
    graphic_only = [f for f in figures if f["graphic_only"]]
    figure_docs = {f["doc"] for f in high_signal}

    print(f"pages routed: {total}   errors: {len(errors)}")
    for source, count in by_source.most_common():
        print(f"  {source:<9} {count:>6}  ({count / max(1, total):.0%})")
    print(
        f"figures: {len(figures)} total, {len(high_signal)} high-signal"
        f"   fully graphic: {len(graphic_only)}"
    )
    print(f"documents containing high-signal figures: {len(figure_docs)}")

    agreement = []
    if not args.skip_crosscheck and args.ocr_sample > 0:
        print(f"\ncrosschecking {args.ocr_sample} pages per source with OCR...")
        agreement = crosscheck(args.corpus, routes, args.ocr_sample)
        ok = [c for c in agreement if "error" not in c]
        failed = len(agreement) - len(ok)
        if failed:
            print(f"crosscheck FAILED on {failed}/{len(agreement)} pages:")
            for c in agreement:
                if "error" in c:
                    print(f"   {c['doc']} p{c.get('page')}: {c['error']}")
        if not ok:
            print("no crosscheck page succeeded; agreement rate is unknown")
        else:
            recalls = [c["numeric_recall"] for c in ok if c["numeric_recall"] is not None]
            precisions = [
                c["numeric_precision"] for c in ok if c["numeric_precision"] is not None
            ]
            agree = sum(1 for c in ok if c["text_agrees"])
            mean_recall = sum(recalls) / len(recalls) if recalls else 0.0
            mean_precision = sum(precisions) / len(precisions) if precisions else 0.0
            print(f"crosschecked: {len(ok)} ok, {failed} failed")
            print(f"  normalised text agrees on {agree}/{len(ok)}")
            print(f"  numeric recall (route values found by OCR): {mean_recall:.0%}")
            print(f"  numeric precision (OCR values found in route): {mean_precision:.0%}")

    report = {
        "corpus": str(args.corpus),
        "page_cap": args.page_cap,
        "pages_routed": total,
        "errors": len(errors),
        "by_source": dict(by_source),
        "figure_pages": len(figures),
        "high_signal_figures": len(high_signal),
        "fully_graphic_pages": len(graphic_only),
        "documents_with_figures": len(figure_docs),
        "routes": routes,
        "crosscheck": agreement,
    }
    args.out.write_text(json.dumps(report, indent=1), encoding="utf-8")
    print(f"\nwrote {args.out}")

    manifest_path = args.out.with_name("figure_manifest.json")
    manifest_path.write_text(json.dumps(figures, indent=1), encoding="utf-8")
    print(f"wrote {manifest_path} ({len(figures)} figures)")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
