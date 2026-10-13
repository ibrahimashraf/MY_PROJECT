import json
import pathlib
import sys

def main() -> None:
    root = pathlib.Path(__file__).parent
    meta_path = root / "metadata.json"
    if not meta_path.exists():
        print("FAIL: metadata.json missing — run generate.py first", file=sys.stderr)
        sys.exit(1)
    meta = json.loads(meta_path.read_text())
    by_type: dict[str, list[dict]] = {}
    for row in meta:
        by_type.setdefault(row["defect_type"], []).append(row)

    ok = True
    for dt, rows in sorted(by_type.items()):
        total = len(rows)
        train = sum(1 for r in rows if r.get("split") == "train")
        held = sum(1 for r in rows if r.get("split") == "held-out")
        has_inspector = all(r.get("inspector") for r in rows)
        has_standard = all(r.get("standard") for r in rows)
        # also check files exist (sample 3 per type to keep validation fast)
        sample_ok = True
        for r in rows[:3]:
            p = root / r["image"]
            if not p.exists():
                sample_ok = False
                break
        status = "PASS" if (total >= 500 and train == 400 and held == 100 and has_inspector and has_standard and sample_ok) else "FAIL"
        print(f"{status}: {dt} total={total} train={train} held-out={held} inspector={has_inspector} standard={has_standard} sample_exists={sample_ok}")
        if status == "FAIL":
            ok = False

    expected = 12
    if len(by_type) != expected:
        print(f"FAIL: expected {expected} defect types, got {len(by_type)}", file=sys.stderr)
        ok = False

    if ok:
        print(f"PASS: corpus gate — {len(by_type)} types × 500 (400 train + 100 held-out) — synthetic placeholder, no precision/recall claimed")
        sys.exit(0)
    else:
        print("FAIL: corpus gate", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
