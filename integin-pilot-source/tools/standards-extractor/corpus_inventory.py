"""Inventory what the corpus actually is, before trying to parse it.

Every quality number produced so far inherited an error from routing on a
single reader. pymupdf returns empty text for documents that pypdf reads
happily, so "image_only" was largely a pymupdf artefact, and the corpus turned
out to contain Chinese-language documents filed under ASME filenames. Both
misled every downstream measurement.

This costs no OCR and no network. For each document it records what each reader
can actually see, so routing can be decided on evidence:

    corpus_inventory.py --corpus D:\\engineering_library --out inventory.json

Language is detected here rather than assumed, because the filename lies.
"""

from __future__ import annotations

import argparse
import json
import logging
import pathlib
import re
import sys
from collections import Counter

import pymupdf
import pypdf

SAMPLE_PAGES = 12

CJK = re.compile(r"[\u3040-\u30ff\u3400-\u4dbf\u4e00-\u9fff\uac00-\ud7af]")
LATIN = re.compile(r"[A-Za-z]")


def language_of(text: str) -> str:
    """Coarse script detection. Deliberately not a language model."""
    if not text:
        return "unknown"
    cjk = len(CJK.findall(text))
    latin = len(LATIN.findall(text))
    if cjk > latin:
        return "cjk"
    if latin > 0:
        return "latin"
    return "non_script"


def read_with_pymupdf(doc, pages: list[int]) -> str:
    out = []
    for number in pages:
        try:
            out.append(doc[number - 1].get_text())
        except Exception:
            continue
    return "\n".join(out)


def read_with_pypdf(path: pathlib.Path, pages: list[int]) -> str:
    """Second opinion.

    Essential, not optional: pymupdf reports zero text for files that carry a
    perfectly readable text layer, and routing on that alone sends perfectly
    good documents to OCR.
    """
    try:
        reader = pypdf.PdfReader(str(path))
    except Exception:
        return ""
    out = []
    for number in pages:
        if number - 1 >= len(reader.pages):
            continue
        try:
            out.append(reader.pages[number - 1].extract_text() or "")
        except Exception:
            continue
    return "\n".join(out)


def single_char_ratio(text: str) -> float:
    lines = [ln for ln in text.splitlines() if ln.strip()]
    if not lines:
        return 0.0
    return sum(1 for ln in lines if len(ln.strip()) == 1) / len(lines)


def classify(mu_text: str, pp_text: str) -> str:
    """Damage class, using both readers.

    'none' means neither reader finds text on the sampled pages, which is the
    only evidence that OCR is genuinely required.
    """
    mu_useful = len(mu_text.strip()) >= 60
    pp_useful = len(pp_text.strip()) >= 60
    if not mu_useful and not pp_useful:
        return "image_only"
    # Prefer whichever reader produced something a human could read.
    reference = mu_text if mu_useful else pp_text
    if single_char_ratio(reference) >= 0.25:
        return "fragmented"
    return "clean"


def inspect(path: pathlib.Path, root: pathlib.Path) -> dict:
    try:
        doc = pymupdf.open(str(path))
    except Exception as exc:
        return {"doc": path.relative_to(root).as_posix(), "error": f"open: {exc}"}

    pages = list(range(1, min(doc.page_count, SAMPLE_PAGES) + 1))
    mu_text = read_with_pymupdf(doc, pages)
    vectors = images = tables = 0
    for number in pages:
        try:
            page = doc[number - 1]
            vectors += len(page.get_drawings())
            images += len(page.get_images(full=True))
            tables += len(page.find_tables().tables)
        except Exception:
            continue
    page_count = doc.page_count
    doc.close()

    pp_text = read_with_pypdf(path, pages)
    kind = classify(mu_text, pp_text)

    combined = mu_text if len(mu_text.strip()) >= 60 else pp_text
    return {
        "doc": path.relative_to(root).as_posix(),
        "pages": page_count,
        "class": kind,
        "language": language_of(combined),
        "pymupdf_chars": len(mu_text.strip()),
        "pypdf_chars": len(pp_text.strip()),
        # A reader disagreement is the signal that flagged 34 real documents.
        "readers_disagree": (
            len(mu_text.strip()) < 60 <= len(pp_text.strip())
            or len(pp_text.strip()) < 60 <= len(mu_text.strip())
        ),
        "vector_paths_sampled": vectors,
        "raster_images_sampled": images,
        "tables_sampled": tables,
    }


def route_for(row: dict) -> str:
    """The cheapest reader that can be trusted for this document."""
    if row.get("error"):
        return "manual_review"
    if row["class"] == "clean":
        return "pymupdf" if row["pymupdf_chars"] >= 60 else "pypdf"
    if row["class"] == "fragmented":
        return "geometry_repair"
    # image_only by both readers
    return "rapidocr" if row["language"] == "cjk" else "tesseract"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", required=True, type=pathlib.Path)
    parser.add_argument("--out", required=True, type=pathlib.Path)
    args = parser.parse_args()

    logging.disable(logging.CRITICAL)
    if not args.corpus.is_dir():
        print(f"corpus not found: {args.corpus}", file=sys.stderr)
        return 1

    rows = []
    for path in sorted(args.corpus.rglob("*.pdf")):
        rows.append(inspect(path, args.corpus))
        if len(rows) % 25 == 0:
            print(f"  {len(rows)} documents", file=sys.stderr)

    for row in rows:
        if "error" not in row:
            row["route"] = route_for(row)

    args.out.write_text(json.dumps(rows, indent=1), encoding="utf-8")

    ok = [r for r in rows if "error" not in r]
    print(f"\ndocuments inventoried: {len(ok)}  (errors: {len(rows) - len(ok)})")
    print(f"sampled {SAMPLE_PAGES} pages each\n")

    print("damage class:")
    for key, count in Counter(r["class"] for r in ok).most_common():
        print(f"  {key:<12} {count:>4}")
    print("\nlanguage:")
    for key, count in Counter(r["language"] for r in ok).most_common():
        print(f"  {key:<12} {count:>4}")
    print("\nreader disagreement (pymupdf blind, pypdf reads, or vice versa):")
    disagree = [r for r in ok if r["readers_disagree"]]
    print(f"  {len(disagree)} documents")
    print("\nrouted to:")
    for key, count in Counter(r["route"] for r in ok).most_common():
        print(f"  {key:<18} {count:>4}")

    tables = [r for r in ok if r["tables_sampled"] > 0]
    print(f"\ndocuments with detectable tables: {len(tables)}")
    print(f"\nwrote {args.out}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())