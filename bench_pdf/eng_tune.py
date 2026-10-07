import functools
import json
import re
import shutil
import subprocess
import anymd_plus as P

rep = json.load(open("report.json", encoding="utf-8"))
NPX = shutil.which("npx.cmd") or "npx"
outs = {}
for tag in ["handbook", "catalog-crosby"]:
    src = rep[tag]["src"]
    p = subprocess.run([NPX, "-y", "@sylphx/anymd", src], capture_output=True, timeout=600)
    outs[tag] = p.stdout.decode("utf-8", "err")
    print(tag, "chars:", len(outs[tag]), flush=True)

KEYS = ["shackle", "Crosby", "35355", "Rigging"]
for floor in [15, 12, 10, 8]:
    P._garbage_runs = functools.partial(
        lambda line, min_len, max_ratio: [
            m.group(0) for m in re.finditer(r"[^\s|#*\-]{%d,}" % min_len, line)
            if len(set(m.group(0))) / len(m.group(0)) < max_ratio],
        min_len=floor, max_ratio=0.35)
    # restore real fn signature use: patch module attr with closure
    def _gr(line, min_len=floor, max_ratio=0.35):
        return [m.group(0) for m in re.finditer(r"[^\s|#*\-]{%d,}" % min_len, line)
                if len(set(m.group(0))) / len(m.group(0)) < max_ratio]
    P._garbage_runs = _gr
    # clean_leaders/garbage_ratio look up P._garbage_runs at call time? No:
    # they call the module-global _garbage_runs -> patch anymd_plus module attr works
    import anymd_plus
    anymd_plus._garbage_runs = _gr
    for tag, raw in outs.items():
        out = P.clean(raw)
        b, a = P.stats(raw), P.stats(out)
        nums_b = sorted(re.findall(r"\d[\d.,]*", raw))
        nums_a = sorted(re.findall(r"\d[\d.,]*", out))
        lost = [k for k in KEYS if k in raw and k not in out]
        print("floor=%d %-14s garb=%.4f->%.4f heads=%d/%d tables=%d/%d nums_same=%s keys_lost=%s"
              % (floor, tag, b["garbage_ratio"], a["garbage_ratio"],
                 b["headings"], a["headings"], b["tables"], a["tables"],
                 nums_b == nums_a, lost if lost else "none"))
