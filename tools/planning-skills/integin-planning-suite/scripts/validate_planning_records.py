#!/usr/bin/env python3
"""Read-only validation for an explicitly selected planning-record root."""

from __future__ import annotations

import argparse
import json
from pathlib import Path

REQUIRED = ("task_plan.md", "findings.md", "progress.md")
HEADINGS = {"task_plan.md": ("# Task Plan", "## Goal", "## Authorized scope and stop conditions", "## Phases", "## Decisions"), "findings.md": ("# Findings",), "progress.md": ("# Progress", "## Continuity checkpoint")}


def main() -> int:
    parser = argparse.ArgumentParser(description="Read-only planning-record validation.")
    parser.add_argument("--root", required=True); parser.add_argument("--profile", choices=("basic", "structured"), default="basic"); parser.add_argument("--format", choices=("text", "json"), default="text")
    args = parser.parse_args(); root = Path(args.root).expanduser()
    if not root.is_dir():
        result = {"root": str(root), "valid": False, "error": "root is not an accessible directory"}; print(json.dumps(result, indent=2) if args.format == "json" else "Planning record validation: INVALID ROOT"); return 2
    rows = []; valid = True
    for name in REQUIRED:
        path = root / name; row = {"file": name, "exists": path.is_file(), "nonempty": False, "missing_headings": []}
        if not row["exists"]: row["status"] = "missing"; valid = False
        else:
            content = path.read_text(encoding="utf-8-sig"); row["nonempty"] = bool(content.strip())
            if not row["nonempty"]: row["status"] = "empty"; valid = False
            else:
                row["missing_headings"] = [heading for heading in HEADINGS[name] if heading not in content] if args.profile == "structured" else []
                row["status"] = "missing_headings" if row["missing_headings"] else "ok"; valid = valid and row["status"] == "ok"
        rows.append(row)
    result = {"root": str(root.resolve()), "profile": args.profile, "valid": valid, "records": rows}
    if args.format == "json": print(json.dumps(result, indent=2))
    else:
        print("Planning record validation: " + ("PASS" if valid else "FAIL")); [print(f"- {row['file']}: {row['status']}") for row in rows]
    return 0 if valid else 1


if __name__ == "__main__":
    raise SystemExit(main())
