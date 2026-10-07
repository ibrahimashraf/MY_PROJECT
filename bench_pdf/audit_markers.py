"""Generic audit: marker preservation + edge removal on ANY .md under given dirs."""
import glob
import os
import re
import sys
from anymd_plus import stats, strip_edges

targets = []
for d in sys.argv[1:]:
    targets += glob.glob(os.path.join(d, "*.md"))
print("files:", len(targets))
bad = 0
for p in sorted(set(targets)):
    try:
        md = open(p, encoding="utf-8").read()
    except Exception as e:
        print("UNREADABLE", p, e)
        continue
    out = strip_edges(md)
    for pat, name in [(r"<!-- page \d+ -->", "page"), ("<!-- OCR text -->", "ocr"),
                      (r"<!-- Stopped at", "cursor"), (r"#{1,6}\s", "heads"),
                      (r"\|", "pipes")]:
        b, a = len(re.findall(pat, md)), len(re.findall(pat, out))
        if name in ("page", "ocr", "cursor", "heads") and b != a:
            bad += 1
            print("LOSS [%s] %s: %d -> %d" % (name, os.path.basename(p), b, a))
print("done, files with structural loss:", bad)
