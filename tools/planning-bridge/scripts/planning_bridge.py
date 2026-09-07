#!/usr/bin/env python3
"""Disabled-by-default local event bridge for planning context and completion checks."""

from __future__ import annotations

import argparse
import json
import re
import sys
from datetime import datetime, timezone
from pathlib import Path
from typing import Any


SENSITIVE_TERMS = ("password", "secret", "token", "authorization", "cookie", "private_key", "-----begin")
RECORDS = ("task_plan.md", "findings.md", "progress.md")


class BridgeError(Exception):
    pass


def now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def safe_root(raw: str) -> Path:
    root = Path(raw).expanduser()
    if not root.is_dir():
        raise BridgeError("selected root is not an accessible directory")
    return root.resolve()


def read_config(root: Path) -> dict[str, Any]:
    path = root / ".planning" / "automation.json"
    if not path.is_file():
        raise BridgeError("automation configuration is absent")
    try:
        data = json.loads(path.read_text(encoding="utf-8-sig"))
    except (OSError, json.JSONDecodeError) as error:
        raise BridgeError("automation configuration is unreadable") from error
    if not isinstance(data, dict):
        raise BridgeError("automation configuration must be a JSON object")
    configured = data.get("project_root")
    if not isinstance(configured, str):
        raise BridgeError("automation configuration has no project_root")
    if safe_root(configured) != root:
        raise BridgeError("automation configuration root does not match selected root")
    if not isinstance(data.get("enabled", False), bool):
        raise BridgeError("automation enabled value must be boolean")
    maximum = data.get("max_context_chars", 2000)
    if not isinstance(maximum, int) or maximum < 128 or maximum > 10000:
        raise BridgeError("max_context_chars must be an integer from 128 to 10000")
    return data


def redact(content: str, maximum: int) -> str:
    lines: list[str] = []
    used = 0
    for line in content.splitlines():
        output = "[redacted sensitive-looking line]" if any(term in line.lower() for term in SENSITIVE_TERMS) else line
        if used + len(output) + 1 > maximum:
            break
        lines.append(output)
        used += len(output) + 1
    return "\n".join(lines)


def render_context(root: Path, maximum: int) -> dict[str, str]:
    per_record = max(128, maximum // len(RECORDS))
    rendered: dict[str, str] = {}
    for name in RECORDS:
        path = root / name
        rendered[name] = redact(path.read_text(encoding="utf-8-sig"), per_record) if path.is_file() else "[missing]"
    return rendered


def completion_status(root: Path) -> dict[str, Any]:
    path = root / "task_plan.md"
    if not path.is_file():
        return {"all_phases_complete": False, "completed_items": 0, "remaining_items": 0, "reason": "task_plan.md is missing"}
    content = path.read_text(encoding="utf-8-sig")
    completed = len(re.findall(r"(?m)^\s*- \[x\]", content, flags=re.IGNORECASE))
    remaining = len(re.findall(r"(?m)^\s*- \[ \]", content))
    complete = completed > 0 and remaining == 0
    return {"all_phases_complete": complete, "completed_items": completed, "remaining_items": remaining, "reason": "complete" if complete else "planning checklist remains incomplete"}


def append_audit(root: Path, event: str, result: str, bypassed: bool, reason: str) -> None:
    path = root / ".planning" / "automation-events.jsonl"
    path.parent.mkdir(parents=True, exist_ok=True)
    entry = {"time": now(), "event": event, "result": result, "bypassed": bypassed, "reason": reason[:240]}
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(entry, sort_keys=True) + "\n")


def fail_open(event: str, reason: str) -> dict[str, Any]:
    return {"event": event, "inject": False, "context": {}, "allow": True, "bypassed": False, "reason": reason}


def handle(root: Path, request: dict[str, Any]) -> dict[str, Any]:
    event = request.get("event")
    if event not in {"prepare_context", "check_completion"}:
        return fail_open(str(event), "unknown event")
    config = read_config(root)
    bypassed = (root / ".planning" / "automation-disabled").is_file()
    if not config["enabled"]:
        response = fail_open(event, "automation is disabled")
        append_audit(root, event, "disabled", False, response["reason"])
        return response
    if bypassed:
        response = fail_open(event, "local emergency bypass is active")
        response["bypassed"] = True
        append_audit(root, event, "bypass", True, response["reason"])
        return response
    if event == "prepare_context":
        response = {"event": event, "inject": True, "context": render_context(root, config["max_context_chars"]), "allow": True, "bypassed": False, "reason": "bounded local context prepared"}
        append_audit(root, event, "injected", False, response["reason"])
        return response
    status = completion_status(root)
    response = {"event": event, "inject": False, "context": {}, "allow": bool(status["all_phases_complete"]), "bypassed": False, "reason": status["reason"], "status": status}
    append_audit(root, event, "allow" if response["allow"] else "warn", False, response["reason"])
    return response


def main() -> int:
    parser = argparse.ArgumentParser(description="Disabled-by-default local planning bridge.")
    parser.add_argument("--root", required=True, help="Owner-bound project root.")
    args = parser.parse_args()
    try:
        root = safe_root(args.root)
        request = json.loads(sys.stdin.read().lstrip("\ufeff"))
        if not isinstance(request, dict):
            raise BridgeError("request must be a JSON object")
        response = handle(root, request)
    except (BridgeError, OSError, json.JSONDecodeError) as error:
        response = fail_open("unavailable", str(error))
    print(json.dumps(response, sort_keys=True))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
