"""Page text extraction with geometry repair for standards PDFs.

Many standards PDFs emit one glyph per text-showing operator. get_text() then
returns "1\n4\n0\n°\nF" instead of "140°F", which silently produced the 0.0 and
9.0 rows in extracted_codex.json. The glyph *positions* are correct, so the
text can be rebuilt from geometry for free -- no OCR.

Evidence this design is based on is recorded in the review record for the
standards-extractor enhancement; see audit_extraction.py for the corpus sweep.
"""

from __future__ import annotations

import os
import re
import shutil
import subprocess
import tempfile
import unicodedata
from dataclasses import dataclass, field

# Fraction of single-character lines above which a text layer is treated as
# glyph-fragmented and rebuilt from geometry.
FRAGMENT_LINE_RATIO = 0.25

# A row is a run of glyphs sharing a baseline within this fraction of the
# page's median glyph height.
#
# Scaled rather than fixed because the same page mixes display faces with body
# text, and a fixed tolerance either shatters a title into single letters or
# welds two body lines together. The kreuzberg/xberg PDF extractors use the same
# 0.5x-line-height heuristic for this defect.
#
# Height comes from the glyph bounding box, never from span["size"]: producers
# that emit one glyph per operator also report a bogus size of 0.1pt, which
# would drive this tolerance to 0.05pt and silently break the repair.
BASELINE_TOLERANCE_RATIO = 0.5

# Insert a space when the gap before a glyph exceeds this fraction of that
# glyph's own advance width. A fixed point threshold splits every letter on
# large display fonts ("A N A M E R I C A N"); scaling by glyph width does not.
WORD_GAP_RATIO = 0.32

# Below this, a page is treated as having no usable text layer at all.
MIN_PAGE_CHARS = 60

# Extraction sources, in descending order of trust.
SOURCE_NATIVE = "native"
SOURCE_REPAIRED = "repaired"
SOURCE_OCR = "ocr"
SOURCE_NONE = "none"


@dataclass
class PageText:
    """Extracted text for one page plus how it was obtained."""

    source: str
    text: str
    fragment_ratio: float = 0.0
    # Mean per-word OCR confidence, or 1.0 for text that needs no OCR.
    # Below 1.0 this page must not back a statutory gate on its own.
    confidence: float = 1.0
    # Set when two extractors disagree on this page; such a page must not be
    # promoted to a statutory gate without human adjudication.
    disputed: bool = False
    notes: list[str] = field(default_factory=list)
    # Extracted 2D tabular matrices [ [row1_col1, row1_col2], [row2_col1, ...] ]
    tables: list[list[list[str]]] = field(default_factory=list)

    @property
    def has_text(self) -> bool:
        return len(self.text.strip()) >= MIN_PAGE_CHARS

    @property
    def needs_ocr(self) -> bool:
        return self.source == SOURCE_NONE


def _glyphs(page) -> list[tuple[float, float, float, str, float, float]]:
    """Flatten a page into (y0, x0, x1, char, advance_width, height) tuples."""
    out = []
    for block in page.get_text("rawdict")["blocks"]:
        if block.get("type") != 0:
            continue
        for line in block["lines"]:
            for span in line["spans"]:
                for ch in span["chars"]:
                    if not ch["c"].strip():
                        continue
                    x0, y0, x1, y1 = ch["bbox"]
                    out.append((y0, x0, x1, ch["c"], x1 - x0, y1 - y0))
    return out


def _median(values: list[float], default: float) -> float:
    if not values:
        return default
    ordered = sorted(values)
    mid = len(ordered) // 2
    if len(ordered) % 2:
        return ordered[mid]
    return (ordered[mid - 1] + ordered[mid]) / 2


def repair(page) -> str:
    """Rebuild readable text from glyph geometry. Free; no OCR involved.

    Known imperfection: two adjacent text blocks sharing a baseline merge into
    one row, and a few intra-word gaps survive. Treat the result as a first
    pass to be corroborated by OCR, not as an authoritative source.
    """
    glyphs = _glyphs(page)
    if not glyphs:
        return ""
    tolerance = BASELINE_TOLERANCE_RATIO * _median(
        [g[5] for g in glyphs], default=1.0
    )
    glyphs.sort(key=lambda g: (g[0], g[1]))

    rows: list[list[tuple[float, float, str, float]]] = []
    current: list[tuple[float, float, str, float]] = []
    row_y = None
    for y0, x0, x1, char, width, _height in glyphs:
        if row_y is None or abs(y0 - row_y) <= tolerance:
            current.append((x0, x1, char, width))
            row_y = y0 if row_y is None else row_y * 0.9 + y0 * 0.1
        else:
            rows.append(current)
            current = [(x0, x1, char, width)]
            row_y = y0
    if current:
        rows.append(current)

    lines = []
    for row in rows:
        row.sort(key=lambda item: item[0])
        buf = []
        prev_x1 = None
        for x0, x1, char, width in row:
            if prev_x1 is not None and (x0 - prev_x1) > WORD_GAP_RATIO * width:
                buf.append(" ")
            buf.append(char)
            prev_x1 = x1
        line = "".join(buf).strip()
        if line:
            lines.append(line)
    return "\n".join(lines)


def fragment_ratio(text: str) -> float:
    """Share of non-blank lines holding exactly one character."""
    lines = [ln for ln in text.splitlines() if ln.strip()]
    if not lines:
        return 0.0
    single = sum(1 for ln in lines if len(ln.strip()) == 1)
    return single / len(lines)


def is_fragmented(text: str) -> bool:
    return fragment_ratio(text) >= FRAGMENT_LINE_RATIO


def desmear(text: str) -> str:
    """Collapse kerning-split glyph runs left over by repair.

    "1 4 0 ° F" -> "140°F". The PDF's own advance widths caused this, so it
    survives geometry repair. Also joins a detached sign to its digits: pypdf
    emits "- 2 9 °C" with the sign on its own run, and without this the value
    parses as a positive 29 instead of -29.
    """
    text = re.sub(r"(?<=\d)\s+(?=\d)", "", text)
    text = re.sub(r"(?<=[−–—\-])\s+(?=[\d.])", "", text)
    text = re.sub(r"\s+(?=[°%])", "", text)
    text = re.sub(r"(?<=[°%])\s+(?=\w)", "", text)
    return text


def normalise(text: str) -> str:
    """NFKC normalisation: decomposes ligatures and compatibility characters.

    A PDF may map "fi" to U+FB01, which breaks substring search and numeric
    scanning alike. Idempotent, and cheap enough to run on every page.
    """
    return unicodedata.normalize("NFKC", text)


def to_number(token: str) -> float | None:
    """Parse an extracted numeric token, or None when it is not a number.

    Standards text uses U+2212 MINUS SIGN and the dashing family, so float()
    raises on "−29". Split glyph runs are collapsed first, since geometry repair
    leaves "−2 9".

    Deliberately does NOT guess at misread characters: rewriting letter-o to
    zero turns "no digits here" into the number 0.0, inventing a statutory
    value out of prose. A misread digit is caught by confidence scoring and
    routed to human adjudication instead.
    """
    token = re.sub(r"(?<=\d)\s+(?=\d)", "", token.strip())
    token = token.replace("\u2212", "-").replace("\u2013", "-").replace("\u2014", "-")
    match = re.search(r"-?\d+(?:\.\d+)?", token)
    if not match:
        return None
    return float(match.group(0))


def extract_page(page) -> PageText:
    """Native text when it is clean, geometry-repaired text when it is not.

    Pages with neither are reported as SOURCE_NONE so the caller can decide
    whether to spend OCR on them.
    """
    native = page.get_text()
    ratio = fragment_ratio(native)
    if len(native.strip()) >= MIN_PAGE_CHARS and ratio < FRAGMENT_LINE_RATIO:
        return PageText(
            source=SOURCE_NATIVE, text=normalise(native), fragment_ratio=ratio
        )

    repaired = normalise(desmear(repair(page)))
    if len(repaired.strip()) >= MIN_PAGE_CHARS:
        return PageText(
            source=SOURCE_REPAIRED,
            text=repaired,
            fragment_ratio=ratio,
            notes=["rebuilt_from_geometry"],
        )
    return PageText(source=SOURCE_NONE, text=repaired, fragment_ratio=ratio)


def find_tesseract() -> str | None:
    """Locate the tesseract binary.

    It is installed but not on PATH on this machine, so PATH lookup alone
    reports it missing. Checked explicitly rather than assumed.
    """
    found = shutil.which("tesseract")
    if found:
        return found
    for candidate in (
        r"C:\Program Files\Tesseract-OCR\tesseract.exe",
        r"C:\Program Files (x86)\Tesseract-OCR\tesseract.exe",
    ):
        if os.path.isfile(candidate):
            return candidate
    return None


def _numeric_tokens(text: str) -> set[str]:
    """Numeric tokens, with U+2212 MINUS SIGN folded to ASCII hyphen.

    Standards text writes negatives as U+2212, so an ASCII-only pattern
    silently drops every negative limit -- exactly the values a safety gate
    depends on most.
    """
    text = text.replace("\u2212", "-").replace("\u2013", "-").replace("\u2014", "-")
    return {m.group(0) for m in re.finditer(r"-?\d+(?:\.\d+)?", text)}


def third_opinion(page) -> set[str]:
    """An independent read of the numbers on a page, for cross-checking.

    pypdf coalesces glyphs by its own geometry, so it splits letters ("1 4 0 ° F")
    where the repair does not. Letter splitting does not touch numbers, so
    after de-smearing its numeric tokens stand as a genuinely independent vote
    on every value -- useful because agreement between two readers that share a
    strategy proves much less than agreement between two that do not.
    """
    try:
        import pypdf
    except ImportError:
        return set()
    return set(_numeric_tokens(desmear(_pypdf_page_text(page))))


def _pypdf_page_text(page) -> str:
    """Run pypdf over the same file pymupdf has open."""
    import pypdf

    try:
        path = page.parent.name
        index = page.number
    except AttributeError:
        return ""
    if not path:
        return ""
    try:
        reader = pypdf.PdfReader(path)
        if index >= len(reader.pages):
            return ""
        return reader.pages[index].extract_text() or ""
    except Exception:
        return ""


def ocr_word_boxes(page, dpi: int = 150, psm: int = 6) -> list[tuple[str, tuple[float, float, float, float]]]:
    """OCR a page and return (word, bbox) pairs, for cropping a region.

    Needed because a page whose text came from OCR has no searchable text
    layer: page.search_for finds nothing, so without the boxes a crop cannot be
    placed and the value is unreviewable.
    """
    executable = find_tesseract()
    if not executable:
        return []
    with tempfile.TemporaryDirectory() as tmp:
        image = os.path.join(tmp, "page.png")
        page.get_pixmap(dpi=dpi).save(image)
        proc = subprocess.run(
            [executable, image, "stdout", "-l", "eng", "--psm", str(psm), "tsv"],
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            check=False,
        )
    if proc.returncode != 0:
        return []
    boxes = []
    for row in proc.stdout.splitlines()[1:]:
        cells = row.split("\t")
        if len(cells) < 12 or not cells[11].strip():
            continue
        try:
            left, top = float(cells[6]), float(cells[7])
            width, height = float(cells[8]), float(cells[9])
        except ValueError:
            continue
        # tesseract reports pixels; convert back to PDF points.
        scale = 72.0 / dpi
        boxes.append((cells[11], (left * scale, top * scale, (left + width) * scale, (top + height) * scale)))
    return boxes


def ocr_page(page, dpi: int = 150, psm: int = 6, lang: str = "eng") -> PageText:
    """OCR a page render via tesseract.

    Measured on this corpus at ~2.3s/page against ~28s/page for an ONNX OCR
    engine, with equal accuracy on the hardest page tested, so tesseract is the
    default. Emits TSV rather than plain text so per-word confidence survives;
    without it an OCR page cannot be distinguished from a trustworthy one.

    Returns SOURCE_NONE when the binary is absent or fails rather than raising,
    so a sweep can still report coverage.
    """
    executable = find_tesseract()
    if not executable:
        return PageText(
            source=SOURCE_NONE, text="", notes=["tesseract_not_installed"]
        )

    with tempfile.TemporaryDirectory() as tmp:
        image = os.path.join(tmp, "page.png")
        page.get_pixmap(dpi=dpi).save(image)
        proc = subprocess.run(
            [executable, image, "stdout", "-l", lang, "--psm", str(psm), "tsv"],
            capture_output=True,
            text=True,
            encoding="utf-8",
            errors="replace",
            check=False,
        )
    if proc.returncode != 0:
        return PageText(source=SOURCE_NONE, text="", notes=["ocr_failed"])

    words, scores = _parse_tsv(proc.stdout)
    if not words:
        return PageText(source=SOURCE_NONE, text="", notes=["ocr_empty"])

    confidence = sum(scores) / len(scores) if scores else 1.0
    return PageText(
        source=SOURCE_OCR,
        text=normalise(desmear(" ".join(words))),
        fragment_ratio=0.0,
        confidence=confidence / 100.0,
        notes=[
            f"ocr_engine=tesseract dpi={dpi} psm={psm}",
            f"min_word_confidence={min(scores) / 100.0:.3f}",
        ],
    )


def _parse_tsv(tsv: str) -> tuple[list[str], list[float]]:
    """Pull word text and confidence out of tesseract TSV output."""
    words: list[str] = []
    scores: list[float] = []
    for row in tsv.splitlines()[1:]:
        cells = row.split("\t")
        if len(cells) < 12:
            continue
        text = cells[11].strip()
        if not text:
            continue
        words.append(text)
        try:
            score = float(cells[10])
        except ValueError:
            continue
        # tesseract emits -1 for structural rows and 0 for unreadable glyphs;
        # neither is a confidence and averaging them in would understate the page.
        if score >= 0:
            scores.append(score)
    return words, scores
