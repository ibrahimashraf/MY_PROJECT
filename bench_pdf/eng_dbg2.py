import re
import shutil
import subprocess
from collections import Counter
from anymd_plus import _garbage_runs, clean, garbage_ratio

with open("report.json", encoding="utf-8") as f:
    import json
    rep = json.load(f)
NPX = shutil.which("npx.cmd") or "npx"
src = rep["handbook"]["src"]
raw = subprocess.run([NPX, "-y", "@sylphx/anymd", src],
                     capture_output=True, timeout=600).stdout.decode("utf-8", "err")
out = clean(raw)

# 1. which numeric tokens differ?
cb, ca = Counter(re.findall(r"\d[\d.,]*", raw)), Counter(re.findall(r"\d[\d.,]*", out))
diff = {t: (cb[t], ca.get(t, 0)) for t in cb if cb[t] != ca.get(t, 0)}
print("numeric token diffs:", len(diff))
for t, (b, a) in sorted(diff.items(), key=lambda kv: -kv[1][0])[:15]:
    print("  %r: %d -> %d" % (t, b, a))

# 2. residual flagged lines after clean
n = 0
for line in out.splitlines():
    if re.match(r"#{1,6}\s", line) or "<!--" in line or not line.strip():
        continue
    if _garbage_runs(line):
        n += 1
        if n <= 6:
            print("RESID:", repr(line[:160]))
print("residual flagged lines:", n)
