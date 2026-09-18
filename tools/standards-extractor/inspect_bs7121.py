import os
import re
import pypdf

STANDARDS_DIR = r"c:\MY_PROJECT\standards"

files = [
    ("BS 7121-1", "BS 7121-1-2016.pdf"),
    ("BS 7121-2-1", "BS 7121-2-1-2012.pdf"),
    ("BS 7121-2-3", "BS 7121 - 2-3.pdf"),
    ("BS 7121-2-4", "BS 7121-2-4 2025.pdf"),
    ("BS 7121-2-7", "BS 7121-2-7-2012 A2-2022.pdf"),
    ("BS 7121-2-9", "BS 7121-2-9-2013-09.pdf"),
    ("BS 7121-3", "BS 7121-3-2017+A1-2019.pdf"),
    ("BS 7121-4", "BS 7121- 4.pdf"),
    ("BS 7121-5", "BS 7121-5-2019.pdf"),
    ("BS 7121-13", "BS 7121-13-2009.pdf"),
    ("BS 7121-14", "BS 7121-14-2005.pdf"),
]

for label, fname in files:
    fpath = os.path.join(STANDARDS_DIR, fname)
    if not os.path.exists(fpath):
        print(f"MISSING: {label} -> {fname}")
        continue
    try:
        reader = pypdf.PdfReader(fpath)
        num_pages = len(reader.pages)
        print(f"=== {label}: {fname} ({num_pages} pages) ===")
        text = ""
        for p in range(min(8, num_pages)):
            text += f"\n--- Page {p+1} ---\n" + reader.pages[p].extract_text()
        
        lines = [line.strip() for line in text.splitlines() if line.strip()]
        toc_lines = []
        for line in lines:
            if re.match(r"^\d+(\.\d+)*\s+[A-Z]", line) or "Table " in line or "Clause " in line or "Contents" in line:
                toc_lines.append(line)
        print("TOC sample:")
        for t in toc_lines[:15]:
            print("  ", t)
    except Exception as e:
        print(f"ERR reading {label}: {e}")
