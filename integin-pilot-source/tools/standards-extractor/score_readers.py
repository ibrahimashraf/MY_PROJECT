"""Score the limit extractors against human-labelled ground truth.

Reader agreement is not correctness: readers can share a failure mode and agree
perfectly on a page where each of them read the wrong column. This measures each
reader against labels a human checked against the page image.

    score_readers.py --corpus D:\\engineering_library --worksheet <worksheet.csv>

Only rows with a verdict are scored. verdict is one of:
    ok           value and unit are right
    wrong_value  unit right, number wrong
    wrong_unit   number right, unit wrong
    not_a_limit  the match is a citation, part number or page reference
"""

from __future__ import annotations

import argparse
import csv
import logging
import pathlib
import re
import sys
from collections import defaultdict

import pymupdf

from text_extract import (
    _pypdf_page_text,
    desmear,
    extract_page,
    normalise,
    ocr_page,
)

VERDICTS = {"ok", "wrong_value", "wrong_unit", "not_a_limit"}


def key(value: str, unit: str) -> tuple[str, str]:
    return value.strip().lstrip("-+"), unit.strip().lower()


def readers_for(page) -> dict[str, str]:
    """Every reader's view of one page, as normalised text."""
    views: dict[str, str] = {}
    try:
        views["repair"] = normalise(desmear(extract_page(page).text))
    except Exception:
        views["repair"] = ""
    try:
        views["ocr"] = normalise(ocr_page(page).text)
    except Exception:
        views["ocr"] = ""
    try:
        views["pypdf"] = normalise(desmear(_pypdf_page_text(page)))
    except Exception:
        views["pypdf"] = ""
    return views


def contains_pair(text: str, value: str, unit: str) -> bool:
    """True when the page states this value with this unit, near enough.

    Value must match a digit run in the text (200 matches "2000" so a boundary
    check is required) and the unit must follow within a short window.
    """
    value = value.strip().lstrip("-+")
    unit_re = re.escape(unit.strip()) if unit.strip() else r"\S+"
    pattern = rf"(?<!\d){re.escape(value)}(?!\d)\s*{unit_re}"
    return re.search(pattern, text, re.IGNORECASE) is not None


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--corpus", required=True, type=pathlib.Path)
    parser.add_argument("--worksheet", required=True, type=pathlib.Path)
    args = parser.parse_args()

    logging.disable(logging.CRITICAL)
    if not args.worksheet.is_file():
        print(f"worksheet not found: {args.worksheet}", file=sys.stderr)
        return 1

    # utf-8-sig: this worksheet is filled in by hand, and Excel writes a BOM
    # that would otherwise turn the first column name into "ï»¿id".
    rows = list(csv.DictReader(args.worksheet.open(encoding="utf-8-sig")))
    scored = [r for r in rows if r.get("verdict", "").strip()]
    unlabelled = len(rows) - len(scored)
    if not scored:
        print("no rows labelled yet — fill verdict (and label_value where wrong)")
        print(f"worksheet has {len(rows)} rows, {unlabelled} unlabelled")
        return 1

    unknown = {r["verdict"].strip() for r in scored} - VERDICTS
    if unknown:
        print(f"unrecognised verdict(s): {sorted(unknown)}", file=sys.stderr)
        return 1

    # A verdict of ok asserts the extracted value is correct; anything else is
    # either a correction (label_value) or a rejection (not_a_limit).
    truth: dict[str, tuple[str, str]] = {}
    rejected: set[str] = set()
    for row in scored:
        verdict = row["verdict"].strip()
        if verdict == "not_a_limit":
            rejected.add(row["id"])
            continue
        value = row.get("label_value", "").strip() or row["value"].strip()
        unit = row.get("label_unit", "").strip() or row["unit"].strip()
        truth[row["id"]] = (value, unit)

    pages: dict[tuple[str, int], object] = {}
    hits = defaultdict(lambda: {"found": 0, "total": 0})
    for row in scored:
        oid = row["id"]
        if oid in rejected:
            continue
        expected = truth.get(oid)
        if expected is None:
            continue
        cache_key = (row["doc"], int(row["page"]))
        if cache_key not in pages:
            try:
                doc = pymupdf.open(str(args.corpus / row["doc"]))
                pages[cache_key] = (doc, doc[int(row["page"]) - 1])
            except Exception:
                pages[cache_key] = None
        entry = pages[cache_key]
        if entry is None:
            continue
        _doc, page = entry
        views = readers_for(page)
        for name, text in views.items():
            hits[name]["total"] += 1
            if contains_pair(text, expected[0], expected[1]):
                hits[name]["found"] += 1

    print(f"labelled rows: {len(scored)}  (rejected as not_a_limit: {len(rejected)})")
    print(f"checked against {len(truth)} confirmed values\n")
    print(f"{'reader':<10} {'recall':>8} {'missed':>8}")
    for name in sorted(hits):
        stats = hits[name]
        rate = stats["found"] / stats["total"] if stats["total"] else 0.0
        print(f"{name:<10} {rate:>7.0%} {stats['total'] - stats['found']:>8}")

    print("\nA reader below ~95% recall is not fit to drive a statutory gate")
    print("without a human in the loop.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())