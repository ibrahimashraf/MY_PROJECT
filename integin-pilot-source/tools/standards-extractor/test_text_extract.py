"""Checks for the text-recovery paths, using failures measured on real PDFs.

Run: python test_text_extract.py
"""

from __future__ import annotations

import sys

import text_extract
from text_extract import desmear, fragment_ratio, is_fragmented, normalise, to_number


def test_fragment_detection() -> None:
    fragmented = "S\nE\nC\nT\nI\nO\nN\n7\n-\n3\n.\n2\n:\nP\nR\nA\nC\nT\nI\nC\nE\nS"
    clean = "SECTION 7-3.2: OPERATING PRACTICES\nLoader cranes have traditionally been"
    assert is_fragmented(fragmented), "one glyph per line must read as fragmented"
    assert not is_fragmented(clean), "prose must not read as fragmented"
    assert 0.0 == fragment_ratio(""), "empty text has no lines to judge"


def test_desmear_recovers_split_glyph_numbers() -> None:
    # The exact contexts from extracted_codex.json that produced 0.0 and 9.0.
    assert "140" in desmear("temperatures above 1 4 0 ° F (6 0 ° C )")
    assert "29" in desmear("(−2 9 °C ) , the winch manufacturer")
    assert "180" in desmear("in excess of 1 8 0 °F (8 2 °C )")


def test_desmear_reattaches_a_detached_sign() -> None:
    """pypdf emits the sign as its own run; without this -29 parses as +29."""
    assert to_number(desmear("− 2 9 °C")) == -29.0
    assert to_number(desmear("- 2 0 °F")) == -20.0
    assert to_number(desmear("above 1 4 0 ° F")) == 140.0


def test_normalise_decomposes_ligatures() -> None:
    """A PDF may map 'fi' to U+FB01, which breaks substring search."""
    assert normalise("ﬁnal report") == "final report"
    assert normalise("ﬂow") == "flow"
    assert normalise("plain ascii") == "plain ascii"
    assert normalise("140 °F") == "140 °F", "must not disturb numeric text"


def test_to_number_handles_unicode_minus() -> None:
    """float('−29') raises; standards text is full of U+2212."""
    assert to_number("−29") == -29.0
    assert to_number("−2 9") == -29.0
    assert to_number("140") == 140.0
    assert to_number("") is None


def test_to_number_refuses_to_invent_values_from_prose() -> None:
    """Regression: letter-o to zero turned "no digits" into 0.0."""
    assert to_number("no digits here") is None
    assert to_number("one two three") is None
    assert to_number("NOTE 1") == 1.0


def test_desmear_leaves_clean_text_alone() -> None:
    clean = "wind speed 14 m/s and clearance 9.5 m"
    assert clean == desmear(clean)


def test_split_digits_are_detectable_after_repair() -> None:
    """Repair leaves kerning gaps; desmear must clear them.

    Geometry repair recovered '1 4 0 ° F' on the ASME B30.7 temperature clause
    but not the joined number, so both stages are required.
    """
    repaired = "temperatures above 1 4 0 ° F (6 0 ° C ) or below − 2 9 ° C"
    joined = desmear(repaired)
    assert "140" in joined and "29" in joined


def test_repair_does_not_merge_genuinely_separate_lines() -> None:
    """False-positive guard.

    Content that is legitimately one character per line -- vertical column
    labels, formula subscripts, poetry -- must not be collapsed into one row by
    the repair, or unrelated glyphs get welded into a fabricated string.
    """

    class _Char(dict):
        def __init__(self, c, bbox):
            super().__init__(c=c, bbox=bbox)

    class _Span(dict):
        def __init__(self, chars):
            super().__init__(chars=chars)

    class _Line(dict):
        def __init__(self, chars):
            super().__init__(spans=[_Span(chars)])

    class _Block(dict):
        def __init__(self, lines):
            super().__init__(lines=lines, type=0)

    class _Page:
        def __init__(self, blocks):
            self._blocks = {"blocks": blocks}

        def get_text(self, mode=None):
            assert mode == "rawdict"
            return self._blocks

    # Five labels 20pt apart on the same x -- genuine separate lines.
    lines = [
        _Line([_Char(ch, (72.0, 700.0 - 20.0 * i, 78.0, 710.0 - 20.0 * i))])
        for i, ch in enumerate("ABCDE")
    ]
    out = text_extract.repair(_Page([_Block(lines)]))
    assert out.count("\n") == 4, f"5 separate lines must stay 5 lines, got {out!r}"
    for ch in "ABCDE":
        assert ch in out, f"lost glyph {ch}: {out!r}"


def main() -> int:
    checks = [v for k, v in sorted(globals().items()) if k.startswith("test_")]
    failures = 0
    for check in checks:
        try:
            check()
            print(f"PASS {check.__name__}")
        except AssertionError as exc:
            failures += 1
            print(f"FAIL {check.__name__}: {exc}")
    print(f"\n{len(checks) - failures}/{len(checks)} passed")
    return 1 if failures else 0


if __name__ == "__main__":
    raise SystemExit(main())
