#!/usr/bin/env python3
"""Read-only validation for an explicitly selected planning-record root."""

from __future__ import annotations

import argparse
import json
from pathlib import Path
from typing import Any


REQUIRED_FILES = ("task_plan.md", "findings.md", "progress.md")
STRUCTURED_HEADINGS = {
    "task_plan.md": (
        "# Task Plan",
        "## Goal",
        "## Authorized scope and stop conditions",
        "## Phases",
        "## Decisions",
    ),
    "findings.md": ("# Findings",),
    "progress.md": ("# Progress", "## Continuity checkpoint"),
}


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Read-only validation of planning records in an explicitly selected root."
    )
    parser.add_argument("--root", required=True, help="Authorized planning-record directory.")
    parser.add_argument(
        "--profile",
        choices=("basic", "structured"),
        default="basic",
        help="basic checks file presence and non-empty content; structured also checks template headings.",
    )
    parser.add_argument("--format", choices=("text", "json"), default="text")
    return parser.parse_args()


def inspect(root: Path, profile: str) -> dict[str, Any]:
    records: list[dict[str, Any]] = []
    valid = True

    for filename in REQUIRED_FILES:
        path = root / filename
        item: dict[str, Any] = {"file": filename, "exists": path.is_file(), "nonempty": False, "headings": []}
        if not item["exists"]:
            item["status"] = "missing"
            valid = False
            records.append(item)
            continue

        try:
            content = path.read_text(encoding="utf-8")
        except OSError as error:
            item["status"] = "unreadable"
            item["error"] = str(error)
            valid = False
            records.append(item)
            continue

        item["nonempty"] = bool(content.strip())
        if not item["nonempty"]:
            item["status"] = "empty"
            valid = False
            records.append(item)
            continue

        if profile == "structured":
            missing = [heading for heading in STRUCTURED_HEADINGS[filename] if heading not in content]
            item["headings"] = missing
            if missing:
                item["status"] = "missing_headings"
                valid = False
                records.append(item)
                continue

        item["status"] = "ok"
        records.append(item)

    return {"root": str(root), "profile": profile, "valid": valid, "records": records}


def render_text(result: dict[str, Any]) -> str:
    lines = [f"Planning record validation: {'PASS' if result['valid'] else 'FAIL'}", f"Root: {result['root']}", f"Profile: {result['profile']}"]
    for item in result["records"]:
        detail = item["status"]
        if item.get("headings"):
            detail += f" ({', '.join(item['headings'])})"
        lines.append(f"- {item['file']}: {detail}")
    return "\n".join(lines)


def main() -> int:
    args = parse_args()
    root = Path(args.root).expanduser()
    if not root.is_dir():
        result: dict[str, Any] = {"root": str(root), "error": "root is not an accessible directory"}
        print(json.dumps(result, indent=2) if args.format == "json" else f"Planning record validation: INVALID ROOT\nRoot: {root}")
        return 2

    result = inspect(root, args.profile)
    print(json.dumps(result, indent=2) if args.format == "json" else render_text(result))
    return 0 if result["valid"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
