#!/usr/bin/env python3
"""Disabled-by-default local bridge for bounded planning context and completion decisions."""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

RECORDS = ("task_plan.md", "findings.md", "progress.md")
SENSITIVE = ("password", "secret", "token", "authorization", "cookie", "private_key", "-----begin")


class BridgeError(Exception):
    pass


def root_of(raw: str) -> Path:
    root = Path(raw).expanduser()
    if not root.is_dir(): raise BridgeError("selected root is not an accessible directory")
    return root.resolve()


def safe(event: str, reason: str) -> dict[str, Any]:
    return {"event": event, "inject": False, "context": {}, "allow": True, "bypassed": False, "reason": reason}


def config(root: Path) -> dict[str, Any]:
    path = root / ".planning" / "automation.json"
    if not path.is_file(): raise BridgeError("automation configuration is absent")
    try: value = json.loads(path.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError) as error: raise BridgeError("automation configuration is unreadable") from error
    if not isinstance(value, dict) or not isinstance(value.get("project_root"), str): raise BridgeError("automation configuration is invalid")
    if root_of(value["project_root"]) != root: raise BridgeError("automation configuration root does not match selected root")
    if not isinstance(value.get("enabled", False), bool): raise BridgeError("automation enabled value must be boolean")
    limit = value.get("max_context_chars", 2000)
    if not isinstance(limit, int) or limit < 128 or limit > 10000: raise BridgeError("max_context_chars is invalid")
    return value


def audit(root: Path, event: str, result: str, bypassed: bool, reason: str) -> None:
    path = root / ".planning" / "automation-events.jsonl"; path.parent.mkdir(parents=True, exist_ok=True)
    entry = {"time": datetime.now(timezone.utc).isoformat().replace("+00:00", "Z"), "event": event, "result": result, "bypassed": bypassed, "reason": reason[:240]}
    with path.open("a", encoding="utf-8") as handle: handle.write(json.dumps(entry, sort_keys=True) + "\n")


def redacted(text: str, limit: int) -> str:
    output = []
    for line in text.splitlines():
        output.append("[redacted sensitive-looking line]" if any(term in line.lower() for term in SENSITIVE) else line)
        if len("\n".join(output)) >= limit: break
    return "\n".join(output)[:limit]


def context(root: Path, limit: int) -> dict[str, str]:
    each = max(128, limit // len(RECORDS)); return {name: redacted((root / name).read_text(encoding="utf-8-sig"), each) if (root / name).is_file() else "[missing]" for name in RECORDS}


def completion(root: Path) -> dict[str, Any]:
    path = root / "task_plan.md"
    if not path.is_file(): return {"all_phases_complete": False, "reason": "task_plan.md is missing"}
    text = path.read_text(encoding="utf-8-sig"); done = len(re.findall(r"(?m)^\s*- \[x\]", text, re.I)); remaining = len(re.findall(r"(?m)^\s*- \[ \]", text)); complete = done > 0 and remaining == 0
    return {"all_phases_complete": complete, "completed_items": done, "remaining_items": remaining, "reason": "complete" if complete else "planning checklist remains incomplete"}


def handle(root: Path, request: dict[str, Any]) -> dict[str, Any]:
    event = request.get("event")
    if event not in {"prepare_context", "check_completion"}: return safe(str(event), "unknown event")
    value = config(root); bypass = (root / ".planning" / "automation-disabled").is_file()
    if not value["enabled"]:
        response = safe(event, "automation is disabled"); audit(root, event, "disabled", False, response["reason"]); return response
    if bypass:
        response = safe(event, "local emergency bypass is active"); response["bypassed"] = True; audit(root, event, "bypass", True, response["reason"]); return response
    if event == "prepare_context":
        response = {"event": event, "inject": True, "context": context(root, value["max_context_chars"]), "allow": True, "bypassed": False, "reason": "bounded local context prepared"}; audit(root, event, "injected", False, response["reason"]); return response
    status = completion(root); response = {"event": event, "inject": False, "context": {}, "allow": bool(status["all_phases_complete"]), "bypassed": False, "reason": status["reason"], "status": status}; audit(root, event, "allow" if response["allow"] else "warn", False, response["reason"]); return response


def main() -> int:
    parser = argparse.ArgumentParser(description="Disabled local planning bridge."); parser.add_argument("--root", required=True); args = parser.parse_args()
    try:
        request = json.loads(sys.stdin.read().lstrip("\ufeff")); response = handle(root_of(args.root), request if isinstance(request, dict) else {})
    except (BridgeError, OSError, json.JSONDecodeError) as error: response = safe("unavailable", str(error))
    print(json.dumps(response, sort_keys=True)); return 0


if __name__ == "__main__":
    raise SystemExit(main())
