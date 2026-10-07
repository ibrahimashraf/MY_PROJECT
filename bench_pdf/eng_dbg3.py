import re
import shutil
import subprocess
from anymd_plus import _garbage_runs
with open("report.json", encoding="utf-8") as f:
    import json
    rep = json.load(f)
NPX = shutil.which("npx.cmd") or "npx"
src = rep["handbook"]["src"]
raw = subprocess.run([NPX, "-y", "@sylphx/anymd", src],
                     capture_output=True, timeout=600).stdout.decode("utf-8", "err")
i = raw.find("2.12.02")
print("CTX:", repr(raw[max(0, i - 120):i + 80]))
line = "1.1 RIGGING PROCEDURES 1-3 1.1.1 General ....................1-3 1.1.2 Corporate -3"
print("runs:", [repr(g) for g in _garbage_runs(line)])
