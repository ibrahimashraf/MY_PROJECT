import re
import shutil
import subprocess
from anymd_plus import _garbage_runs, clean_leaders
with open("report.json", encoding="utf-8") as f:
    import json
    rep = json.load(f)
NPX = shutil.which("npx.cmd") or "npx"
src = rep["handbook"]["src"]
raw = subprocess.run([NPX, "-y", "@sylphx/anymd", src],
                     capture_output=True, timeout=600).stdout.decode("utf-8", "err")
l = next(x for x in raw.splitlines() if "1.1.1 General" in x and len(x) > 100 and not x.startswith("#"))
print("in dots:", l.count("."), "len:", len(l))
runs = _garbage_runs(l)
print("runs:", [(len(g), g[:12] + ".." + g[-6:]) for g in runs])
out = clean_leaders(l)
print("out dots:", out.count("."), "len:", len(out))
for m in re.finditer(r"\.{10,}", out):
    print("leftover dot-run len", len(m.group(0)), "ctx:", repr(out[max(0, m.start() - 30):m.end() + 8]))
