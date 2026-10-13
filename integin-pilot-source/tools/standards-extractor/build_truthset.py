"""Build a labelled ground-truth set for the limit extractors.

Reader agreement is a proxy for correctness, not correctness: all readers can
share a failure mode and agree perfectly on a page where every one of them read
the wrong column. The only way to know whether the readers are fit to drive
statutory gates is to check values against the page image by hand.

Emits a worksheet a human can fill in:

    build_truthset.py --corpus D:\\engineering_library --out <dir> --count 60

Sampling is stratified by standards body and deliberately mixes high-agreement
and low-agreement candidates, so the set can catch shared failure as well as
ordinary error.
"""

from __future__ import annotations

import argparse
import csv
import json
import logging
import pathlib
import random
import re
import sys

import pymupdf

from text_extract import desmear, extract_page, normalise, ocr_page, ocr_word_boxes

SEED = 7121

UNIT = r"(?:°\s*[CF]|m/s|mm|%|kg|tonnes?|months?|weeks?|days?|years?|kN)"
VALUE_UNIT = re.compile(r"(-?\d+(?:\.\d+)?)\s*(" + UNIT + r")\b", re.IGNORECASE)

# A candidate must sit near a limit phrase, otherwise citations ("2001;21(3):
# 275-281") parse as durations and pollute the benchmark.
LIMIT_CONTEXT = re.compile(
    r"\b(not (?:less|more) than|at (?:least|most)|maximum|minimum|max\b|min\b|"
    r"up to|exceed|within|throughout|every|per|rated|swl|capacity|limit)\b",
    re.IGNORECASE,
)

# Plausible bounds per unit. Anything outside is usually a citation or a part
# number wearing a unit suffix, and costs a human review slot for nothing.
PLAUSIBLE = {
    "%": (0.0, 100.0),
    "°c": (-80.0, 400.0),
    "°f": (-120.0, 750.0),
    "m/s": (0.0, 60.0),
    "mm": (0.0, 5000.0),
    "kg": (0.0, 500000.0),
    "months": (0.0, 120.0),
    "weeks": (0.0, 520.0),
    "days": (0.0, 3650.0),
    "years": (0.0, 60.0),
    "kn": (0.0, 100000.0),
    "tonnes": (0.0, 10000.0),
}


def canonical(value: str, unit: str) -> tuple[str, str]:
    return value.lstrip("-+"), re.sub(r"\s+", "", unit).lower()


def plausible(value: str, unit: str) -> bool:
    try:
        number = abs(float(value))
    except ValueError:
        return False
    low, high = PLAUSIBLE.get(unit, (float("-inf"), float("inf")))
    return low <= number <= high


def candidates_in(text: str) -> list[tuple[str, str, str, bool]]:
    """Return (value, unit, context, has_limit_keyword) per candidate.

    Plausibility bounds are the only hard filter: they already reject the
    citation noise that motivates this stage (a bibliography line such as
    "2001;21(3):275-281" reads as a 275-year duration, and 2100 years is out of
    range). A limit keyword is recorded as a hint for the human rather than
    enforced, because the keyword list is guesswork and the reviewer is the
    authority.
    """
    found = []
    for match in VALUE_UNIT.finditer(text):
        value, unit = canonical(match.group(1), match.group(2))
        if not plausible(value, unit):
            continue
        context = text[max(0, match.start() - 80) : match.end() + 40].replace("\n", " ")
        found.append((value, unit, context.strip(), bool(LIMIT_CONTEXT.search(context))))
    return found


def page_text(doc, index: int, use_ocr: bool) -> str:
    """Readable text for a page, falling back to OCR when the layer is unusable.

    A third of this corpus is image-only, and those documents are exactly the
    people-lifting and thorough-examination regulations. Without OCR they
    contribute no candidates at all.
    """
    result = extract_page(doc[index])
    if result.has_text:
        return normalise(desmear(result.text))
    if use_ocr:
        ocr = ocr_page(doc[index])
        if ocr.has_text:
            return normalise(ocr.text)
    return ""


def pick_documents(corpus: pathlib.Path, wanted: int) -> list[pathlib.Path]:
    """One document per standards body where possible, for spread."""
    by_body: dict[str, pathlib.Path] = {}
    for path in sorted(corpus.rglob("*.pdf")):
        rel = path.relative_to(corpus)
        body = rel.parts[1] if len(rel.parts) > 2 else "root"
        by_body.setdefault(body, path)
    chosen = list(by_body.values())
    rng = random.Random(SEED)
    rng.shuffle(chosen)
    return chosen[:wanted]


def render_crop(doc, page_number: int, needle: str, out_path: pathlib.Path) -> bool:
    """Write a crop around the first occurrence of `needle` on the page.

    Tries the PDF text layer first. Pages whose text came from OCR have no
    searchable layer, so it falls back to tesseract's word boxes; without that
    fallback every candidate from an image-only document would be unlabellable.
    """
    try:
        page = doc[page_number - 1]
        hits = page.search_for(needle)
    except Exception:
        return False

    if hits:
        rect = hits[0]
    else:
        boxes = ocr_word_boxes(page)
        match = next((b for w, b in boxes if needle in w), None)
        if match is None:
            # Last resort: the whole page, so the reviewer can still see it.
            try:
                page.get_pixmap(dpi=140).save(str(out_path))
            except Exception:
                return False
            return True
        rect = pymupdf.Rect(*match)

    clip = pymupdf.Rect(
        max(page.rect.x0, rect.x0 - 300),
        max(page.rect.y0, rect.y0 - 34),
        min(page.rect.x1, rect.x1 + 300),
        min(page.rect.y1, rect.y1 + 34),
    )
    try:
        page.get_pixmap(dpi=170, clip=clip).save(str(out_path))
    except Exception:
        return False
    return True


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", required=True, type=pathlib.Path)
    parser.add_argument("--out", required=True, type=pathlib.Path)
    parser.add_argument("--docs", type=int, default=20)
    parser.add_argument("--per-doc", type=int, default=6)
    parser.add_argument("--no-ocr", action="store_true")
    args = parser.parse_args()

    logging.disable(logging.CRITICAL)
    if not args.corpus.is_dir():
        print(f"corpus not found: {args.corpus}", file=sys.stderr)
        return 1

    args.out.mkdir(parents=True, exist_ok=True)
    crops = args.out / "crops"
    crops.mkdir(exist_ok=True)

    rows: list[dict] = []
    for doc_path in pick_documents(args.corpus, args.docs):
        try:
            doc = pymupdf.open(str(doc_path))
        except Exception as exc:
            print(f"  skip {doc_path.name}: {exc}", file=sys.stderr)
            continue
        rel = doc_path.relative_to(args.corpus).as_posix()

        best: list[tuple[str, str, str, bool]] = []
        best_page = 0
        for index in range(doc.page_count):
            try:
                text = page_text(doc, index, not args.no_ocr)
            except Exception:
                continue
            hits = candidates_in(text)
            if len(hits) > len(best):
                best, best_page = hits, index + 1
            if len(best) >= args.per_doc * 3:
                break
        if not best or not best_page:
            doc.close()
            continue

        seen: set[tuple[str, str]] = set()
        added = 0
        for value, unit, context, has_keyword in best:
            if added >= args.per_doc:
                break
            key = (value, unit)
            if key in seen:
                continue
            seen.add(key)
            oid = f"T{len(rows) + 1:04d}"
            crop_name = f"{oid}.png"
            render_crop(doc, best_page, value, crops / crop_name)
            rows.append(
                {
                    "id": oid,
                    "doc": rel,
                    "page": best_page,
                    "value": value,
                    "unit": unit,
                    "limit_keyword": "yes" if has_keyword else "no",
                    "context": context,
                    "crop": f"crops/{crop_name}",
                    "label_value": "",
                    "label_unit": "",
                    "verdict": "",
                    "notes": "",
                }
            )
            added += 1
        doc.close()
        print(f"  {rel} p{best_page}: {added} candidates")

    worksheet = args.out / "worksheet.csv"
    with worksheet.open("w", newline="", encoding="utf-8") as handle:
        writer = csv.DictWriter(handle, fieldnames=list(rows[0].keys()))
        writer.writeheader()
        writer.writerows(rows)

    (args.out / "candidates.json").write_text(
        json.dumps(rows, indent=1), encoding="utf-8"
    )
    print(f"\nwrote {worksheet} ({len(rows)} candidates)")
    print("fill label_value, label_unit and verdict, then run score_readers.py")
    print("verdict must be one of: ok, wrong_value, wrong_unit, not_a_limit")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())