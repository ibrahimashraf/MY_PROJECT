import re
import shutil
import subprocess
from anymd_plus import _garbage_runs, clean_leaders, merge_orphans, strip_edges
with open("report.json", encoding="utf-8") as f:
    import json
    rep = json.load(f)
NPX = shutil.which("npx.cmd") or "npx"
src = rep["handbook"]["src"]
raw = subprocess.run([NPX, "-y", "@sylphx/anymd", src],
                     capture_output=True, timeout=600).stdout.decode("utf-8", "err")
cand = [l for l in raw.splitlines() if "1.1.1 General" in l]
print("raw cands:", len(cand))
for l in cand[:2]:
    print("RAW :", repr(l[:200]))
    s1 = strip_edges(l if False else raw)  # placeholder no-op
    print("runs:", [repr(g) for g in _garbage_runs(l)][:3])
    print("LEAD:", repr(clean_leaders(l)[:200]))
