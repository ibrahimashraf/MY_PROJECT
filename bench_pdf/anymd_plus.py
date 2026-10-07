"""anymd_plus: post-processing layer for anymd markdown (pdfmd-inspired filters).

Pipeline: anymd (extract) -> clean() (transform) -> downstream (RAG/index).
Portable rules: each filter is a pure function, translatable to Rust later.

Filters:
  fix_hyphens      join "hy-\\nphen" line wraps (pdfmd: hyphenation repair)
  strip_edges      drop repeating first/last lines across pages incl. "Page N"
                   (pdfmd: header/footer removal; anymd already drops most)
  merge_orphans    join short fragment lines into previous paragraph
                   (pdfmd: orphan merging, default max 45 chars)
  clean_leaders    delete OCR leader-garbage runs (dotted TOC leaders that
                   tesseract renders as "ccccees..."): substrings >=15 chars
                   with unique/len ratio < 0.35. Real words survive
                   ("internationalization" = 0.60).
Gates:
  stats()          words/headings/tables/lists/pages/tokens/garbage-ratio
  preview()        anymd --pages 1-3 + stats (preflight before full-book OCR)
CLI: clean IN [OUT] | stats FILE | preview PDF
"""
import re
import shutil
import subprocess
import sys

PAGE_RE = re.compile(r"<!-- page \d+ -->")
FOOT_RE = re.compile(r"^\s*(pages?\s+\d+|-\s*-\s*\d+|\d+\s*-\s*-|\(\s*\d+\s*\))\s*$", re.I)


def split_pages(md):
    parts = PAGE_RE.split(md)
    marks = PAGE_RE.findall(md)
    return list(zip([""] + marks, parts))  # (marker, body)


def fix_hyphens(text):
    return re.sub(r"(\w)-\n(\w)", lambda m: m.group(1) + m.group(2), text)


STRUCT_RE = re.compile(r"#{1,6}\s|^\s*[-*+] |^\s*\d+[.)] |\|")


def strip_edges(md, min_repeat=3):
    """Drop repeating plain-text first/last lines (running headers, footers,
    'Page N'). Position is tracked over real lines only (blanks and <!-- -->
    markers don't count). Never touches headings, lists, table rows, or
    markers: a repeated '### Section' opening two pages is content."""
    pages = split_pages(md)
    if len(pages) <= 1:
        return md
    firsts, lasts = {}, {}
    bodies = []
    for mark, body in pages:
        raw = body.splitlines()
        real = [l for l in raw if l.strip() and "<!--" not in l
                and not STRUCT_RE.search(l)]
        bodies.append((mark, raw, real))
        if real:
            firsts[real[0]] = firsts.get(real[0], 0) + 1
            lasts[real[-1]] = lasts.get(real[-1], 0) + 1
    n = len(bodies)
    need = max(min_repeat, n // 2)
    out = []
    for mark, raw, real in bodies:
        first = real[0] if real else None
        last = real[-1] if real else None
        kept, seen_real = [], 0
        n_real = len(real)
        for l in raw:
            if (l.strip() and "<!--" not in l and not STRUCT_RE.search(l)):
                seen_real += 1
                if ((seen_real == 1 and l == first and firsts.get(l, 0) * 2 >= need)
                        or (seen_real == n_real and l == last
                            and (lasts.get(l, 0) * 2 >= need or FOOT_RE.match(l)))):
                    continue
            kept.append(l)
        out.append(mark + "\n" + "\n".join(kept))
    return "\n".join(out)


def merge_orphans(md, max_len=45):
    out = []
    for line in md.splitlines():
        s = line.strip()
        if (out and s and len(s) <= max_len
                and not re.match(r"#{1,6}\s|^\s*[-*+] |^\s*\d+[.)] |\|", s)
                and not re.match(r"<!--|^\s*$", out[-1].strip() if out else "")
                and out[-1].strip()
                and not re.match(r"#{1,6}\s|^\s*[-*+] |^\s*\d+[.)] |\|", out[-1].strip())
                and "<!--" not in out[-1]):
            out[-1] = out[-1].rstrip() + " " + s
        else:
            out.append(line)
    return "\n".join(out)


LEADER_MIN_LEN = 15
LEADER_MAX_RATIO = 0.35


def _garbage_runs(line, min_len=LEADER_MIN_LEN, max_ratio=LEADER_MAX_RATIO):
    return [m.group(0) for m in re.finditer(r"[^\s|#*\-]{%d,}" % min_len, line)
            if len(set(m.group(0))) / len(m.group(0)) < max_ratio]


def _leader_sub(m):
    g = m.group(0)
    # single-pass: one span -> one space, no overlapping-replace remainders
    return " " if len(set(g)) / len(g) < LEADER_MAX_RATIO else g


_LEADER_PAT = re.compile(r"[^\s|#*\-]{%d,}" % LEADER_MIN_LEN)


def clean_leaders(md):
    out = []
    for line in md.splitlines():
        if re.match(r"#{1,6}\s", line) or "<!--" in line:
            out.append(line)
            continue
        # table rows are safe: the pattern excludes '|' so cell borders survive
        line = _LEADER_PAT.sub(_leader_sub, line)
        out.append(re.sub(r"\s{2,}", " ", line).strip() if line.strip() else line)
    return "\n".join(out)


def garbage_ratio(md):
    n = t = 0
    for line in md.splitlines():
        if re.match(r"#{1,6}\s", line) or "<!--" in line or not line.strip():
            continue
        t += 1
        if _garbage_runs(line):
            n += 1
    return round(n / max(t, 1), 4)


def clean(md):
    md = fix_hyphens(md)
    md = strip_edges(md)
    md = merge_orphans(md)
    md = clean_leaders(md)
    return md


def stats(md):
    lines = md.splitlines()
    words = sum(len(l.split()) for l in lines)
    return {"words": words,
            "headings": sum(1 for l in lines if re.match(r"#{1,6}\s", l)),
            "tables": sum(1 for l in lines if l.count("|") >= 2),
            "lists": sum(1 for l in lines if re.match(r"\s*[-*+] |\s*\d+[.)] ", l)),
            "pages": len(PAGE_RE.findall(md)) or 1,
            "tokens_est": len(md) // 4,
            "garbage_ratio": garbage_ratio(md)}


def preview(pdf, pages="1-3"):
    npx = shutil.which("npx.cmd") or "npx"
    p = subprocess.run([npx, "-y", "@sylphx/anymd", pdf, "--pages", pages],
                       capture_output=True, timeout=300)
    if p.returncode != 0:
        return {"ok": False, "error": p.stderr.decode("utf-8", "err")[:200]}
    md = p.stdout.decode("utf-8", "err")
    s = stats(clean(md))
    s["ok"] = True
    return s


def arabic_ratio(md):
    ar = sum(1 for c in md if "\u0600" <= c <= "\u06ff" or "\ufb50" <= c <= "\ufeff")
    alpha = sum(1 for c in md if c.isalpha())
    return round(ar / max(alpha, 1), 4)


def arabic_reshape(md, threshold=0.15):
    """Fix visual-order RTL text (see Keith/Arabic bench): extractors emit
    Arabic glyphs in visual (display) order, unreadable to LLMs and
    unsearchable. NFKC-normalize (presentation forms -> standard) then
    reverse each maximal Arabic-script run. Latin/digits/positions untouched.
    No-op when arabic_ratio below threshold. Stdlib only.
    NOTE: this restores readable words + keyword search, not full Unicode
    bidi paragraph reordering (mixed numbers/dates may sit LTR)."""
    import unicodedata
    if arabic_ratio(md) < threshold:
        return md
    ar = r"[\u0600-\u06ff\ufb50-\ufdff\ufe70-\ufeff]"
    out = []
    for line in md.splitlines():
        if "<!--" in line or re.match(r"#{1,6}\s", line):
            out.append(line)
            continue
        line = unicodedata.normalize("NFKC", line)
        out.append(re.sub(ar + "+", lambda m: m.group(0)[::-1], line))
    return "\n".join(out)


def _drive_free(path):
    import shutil
    anchor = os.path.splitdrive(os.path.abspath(path))[0] or os.path.splitdrive(os.getcwd())[0]
    return shutil.disk_usage(anchor + os.sep).free


def _scratch_dir(want_bytes, preferred=None):
    """Pick a temp dir with >= want_bytes free. C: is often full; fall back
    to D:\\anymd_tmp. Raise instead of dying mid-book (cf. Keith chunk_03)."""
    import tempfile
    cands = [p for p in [preferred, tempfile.gettempdir(), r"D:\anymd_tmp"] if p]
    for c in cands:
        try:
            os.makedirs(c, exist_ok=True)
            if _drive_free(c) >= want_bytes:
                return c
        except OSError:
            continue
    raise OSError("no scratch space for %.1f MB (checked %s)"
                  % (want_bytes / 1e6, cands))


def chunk_convert(pdf, max_mb=500, workdir=None, ocr=False, _npx=None):
    """Convert PDFs over anymd's size cap: split with pypdf, convert each
    chunk, merge with continuous <!-- page N --> numbering. Returns md."""
    import os
    size_mb = os.path.getsize(pdf) / 1e6
    npx = _npx or shutil.which("npx.cmd") or "npx"
    if size_mb <= max_mb:
        args = [npx, "-y", "@sylphx/anymd", pdf] + (["--ocr"] if ocr else [])
        p = subprocess.run(args, capture_output=True, timeout=3600)
        if p.returncode != 0:
            raise RuntimeError(p.stderr.decode("utf-8", "err")[:300])
        return p.stdout.decode("utf-8", "err")
    from pypdf import PdfReader, PdfWriter
    workdir = _scratch_dir(2 * os.path.getsize(pdf), workdir)
    os.makedirs(workdir, exist_ok=True)
    rdr = PdfReader(pdf)
    n = len(rdr.pages)
    per = max(1, int(n * max_mb / size_mb))
    merged, offset = [], 0
    for i in range(0, n, per):
        w = PdfWriter()
        for p in range(i, min(i + per, n)):
            w.add_page(rdr.pages[p])
        cp = os.path.join(workdir, "chunk_%04d-%04d.pdf" % (i + 1, min(i + per, n)))
        with open(cp, "wb") as f:
            w.write(f)
        args = [npx, "-y", "@sylphx/anymd", cp] + (["--ocr"] if ocr else [])
        p = subprocess.run(args, capture_output=True, timeout=3600)
        if p.returncode != 0:
            raise RuntimeError("chunk %s: %s" % (cp, p.stderr.decode("utf-8", "err")[:200]))
        part = p.stdout.decode("utf-8", "err")
        part = re.sub(r"<!-- page (\d+) -->",
                      lambda m: "<!-- page %d -->" % (int(m.group(1)) + offset), part)
        merged.append(part)
        offset += min(i + per, n) - i
        try:
            os.remove(cp)  # chunk pdfs are bulk; md is the asset
        except OSError:
            pass
    return "\n".join(merged)


if __name__ == "__main__":
    cmd = sys.argv[1] if len(sys.argv) > 1 else "help"
    if cmd == "clean":
        src = open(sys.argv[2], encoding="utf-8").read()
        out = clean(src)
        if len(sys.argv) > 3:
            open(sys.argv[3], "w", encoding="utf-8").write(out)
        else:
            print(out)
    elif cmd == "stats":
        import json
        print(json.dumps(stats(open(sys.argv[2], encoding="utf-8").read()), indent=1))
    elif cmd == "preview":
        import json
        print(json.dumps(preview(sys.argv[2], sys.argv[3] if len(sys.argv) > 3 else "1-3"), indent=1))
    elif cmd == "convert-big":
        # convert-big PDF [OUT] [--ocr] : size-cap-proof full conversion
        ocr = "--ocr" in sys.argv
        args = [a for a in sys.argv[2:] if not a.startswith("--")]
        md = chunk_convert(args[0], ocr=ocr)
        if len(args) > 1:
            open(args[1], "w", encoding="utf-8").write(md)
        else:
            print(md)
    elif cmd == "reshape":
        src = open(sys.argv[2], encoding="utf-8").read()
        out = arabic_reshape(src)
        if len(sys.argv) > 3:
            open(sys.argv[3], "w", encoding="utf-8").write(out)
        else:
            print(out)
    elif cmd == "batch":
        # batch DIR [--out DIR] [--ocr] : convert+clean every pdf/md, JSONL stats
        import glob
        import json
        if len(sys.argv) < 3 or not os.path.isdir(sys.argv[2]):
            print("usage: batch DIR [--out DIR] [--ocr]")
            sys.exit(2)
        d = sys.argv[2]
        ocr = "--ocr" in sys.argv
        outdir = sys.argv[sys.argv.index("--out") + 1] if "--out" in sys.argv else os.path.join(d, "cleaned")
        os.makedirs(outdir, exist_ok=True)
        files = sorted(glob.glob(os.path.join(d, "*.pdf")) + glob.glob(os.path.join(d, "*.md")))
        for f in files:
            rec = {"file": os.path.basename(f)}
            try:
                md = chunk_convert(f, ocr=ocr) if f.lower().endswith(".pdf") else open(f, encoding="utf-8").read()
                md = arabic_reshape(clean(md))
                dest = os.path.join(outdir, os.path.splitext(os.path.basename(f))[0] + ".clean.md")
                open(dest, "w", encoding="utf-8").write(md)
                rec.update(stats(md))
                rec["ok"] = True
            except Exception as e:
                rec.update(ok=False, error=str(e)[:200])
            print(json.dumps(rec, ensure_ascii=False), flush=True)
    else:
        print("usage: clean IN [OUT] | stats FILE | preview PDF [PAGES] | convert-big PDF [OUT] [--ocr] | reshape IN [OUT]")
