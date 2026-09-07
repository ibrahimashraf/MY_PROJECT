#!/usr/bin/env python3
"""Explicit project-local planning utilities with no hook, network, or shell behavior."""

from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import sys
import tempfile
from collections import Counter
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

RECORDS = ("task_plan.md", "findings.md", "progress.md")
PLAN_ID = re.compile(r"^[a-z0-9][a-z0-9-]{0,63}$")
EXCLUDED = {".git", ".planning", "node_modules", "private", "dist", "build", "coverage", "__pycache__"}
SENSITIVE = ("password", "secret", "token", "authorization", "cookie", "private_key", "-----begin")


class PlanError(Exception):
    pass


def utc() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def root_of(raw: str) -> Path:
    path = Path(raw).expanduser()
    if not path.is_dir():
        raise PlanError("root is not an accessible directory")
    return path.resolve()


def plan_id(value: str) -> str:
    if not PLAN_ID.fullmatch(value):
        raise PlanError("plan ID must be lowercase hyphen-case and at most 64 characters")
    return value


def read_text(path: Path) -> str:
    return path.read_text(encoding="utf-8-sig")


def atomic(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", delete=False, dir=path.parent) as handle:
        handle.write(content)
        temporary = Path(handle.name)
    os.replace(temporary, path)


def read_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(read_text(path))
    except (OSError, json.JSONDecodeError) as error:
        raise PlanError("cannot read JSON state") from error
    if not isinstance(value, dict):
        raise PlanError("JSON state is not an object")
    return value


def write_json(path: Path, value: Any) -> None:
    atomic(path, json.dumps(value, indent=2, sort_keys=True) + "\n")


def planning(root: Path) -> Path:
    return root / ".planning"


def ensure_child(root: Path, path: Path) -> Path:
    resolved = path.resolve()
    try:
        resolved.relative_to(root)
    except ValueError as error:
        raise PlanError("path escapes the selected root") from error
    return resolved


def selected(root: Path, value: str | None, use_active: bool) -> tuple[Path, str]:
    if value:
        identifier = plan_id(value)
        target = ensure_child(root, planning(root) / "plans" / identifier)
        if not target.is_dir():
            raise PlanError("selected plan does not exist")
        return target, identifier
    if use_active:
        pointer = planning(root) / "active-plan.json"
        if not pointer.is_file():
            raise PlanError("no active plan exists")
        identifier = plan_id(str(read_json(pointer).get("plan_id", "")))
        return selected(root, identifier, False)
    return root, "root"


def records(path: Path) -> dict[str, Path]:
    return {name: path / name for name in RECORDS}


def inspect(path: Path) -> dict[str, Any]:
    return {name: {"exists": item.is_file(), "nonempty": bool(read_text(item).strip()) if item.is_file() else False} for name, item in records(path).items()}


def status(path: Path) -> dict[str, int]:
    task = path / "task_plan.md"
    content = read_text(task) if task.is_file() else ""
    return {"complete": len(re.findall(r"(?m)^\s*- \[x\]", content, re.I)), "remaining": len(re.findall(r"(?m)^\s*- \[ \]", content))}


def output(value: Any, style: str) -> None:
    if style == "json":
        print(json.dumps(value, indent=2, sort_keys=True))
    elif isinstance(value, dict):
        for key, item in value.items():
            print(f"{key}: {item}")
    else:
        print(value)


def sha(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(131072), b""):
            digest.update(block)
    return digest.hexdigest()


def template(name: str) -> Path:
    return Path(__file__).resolve().parents[1] / "templates" / name


def do_init(args: argparse.Namespace) -> int:
    root = root_of(args.root)
    target = ensure_child(root, planning(root) / "plans" / plan_id(args.plan_id)) if args.plan_id else root
    target.mkdir(parents=True, exist_ok=True)
    existing = [name for name, path in records(target).items() if path.is_file() and read_text(path).strip()]
    if existing and not args.force:
        raise PlanError("refusing to overwrite nonempty records: " + ", ".join(existing))
    for name in RECORDS:
        source = template(name)
        if not source.is_file():
            raise PlanError("required local template is missing")
        atomic(target / name, read_text(source))
    output({"initialized": args.plan_id or "root", "plan_dir": str(target)}, args.format)
    return 0


def do_resolve(args: argparse.Namespace) -> int:
    root = root_of(args.root); target, identifier = selected(root, args.plan_id, args.use_active)
    output({"root": str(root), "plan_id": identifier, "plan_dir": str(target)}, args.format); return 0


def do_set_active(args: argparse.Namespace) -> int:
    root = root_of(args.root); target, identifier = selected(root, args.plan_id, False)
    missing = [name for name, state in inspect(target).items() if not state["exists"]]
    if missing:
        raise PlanError("refusing to activate incomplete plan: " + ", ".join(missing))
    write_json(planning(root) / "active-plan.json", {"plan_id": identifier, "updated_at": utc()})
    output({"active_plan": identifier, "plan_dir": str(target)}, args.format); return 0


def do_status(args: argparse.Namespace) -> int:
    root = root_of(args.root); target, identifier = selected(root, args.plan_id, args.use_active)
    output({"plan_id": identifier, "records": inspect(target), "task_status": status(target)}, args.format); return 0


def do_complete(args: argparse.Namespace) -> int:
    root = root_of(args.root); target, identifier = selected(root, args.plan_id, args.use_active); counts = status(target)
    complete = counts["complete"] > 0 and counts["remaining"] == 0
    output({"plan_id": identifier, "all_phases_complete": complete, "completed_items": counts["complete"], "remaining_items": counts["remaining"]}, args.format)
    return 0 if complete else 1


def do_ledger_append(args: argparse.Namespace) -> int:
    root = root_of(args.root); _, identifier = selected(root, args.plan_id, args.use_active)
    entry = {"time": utc(), "plan_id": identifier, "kind": args.kind, "message": args.message}
    path = planning(root) / "ledger.jsonl"; path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as handle: handle.write(json.dumps(entry, sort_keys=True) + "\n")
    output(entry, args.format); return 0


def do_ledger_summary(args: argparse.Namespace) -> int:
    root = root_of(args.root); path = planning(root) / "ledger.jsonl"; entries: list[dict[str, Any]] = []; malformed = 0
    if path.is_file():
        for line in read_text(path).splitlines():
            try:
                item = json.loads(line)
                if isinstance(item, dict): entries.append(item)
                else: malformed += 1
            except json.JSONDecodeError: malformed += 1
    output({"entries": len(entries), "kinds": dict(sorted(Counter(str(x.get("kind", "unknown")) for x in entries).items())), "malformed": malformed}, args.format)
    return 0 if malformed == 0 else 1


def attest_path(root: Path, identifier: str) -> Path:
    return planning(root) / "attestations" / f"{identifier}.json"


def do_attest(args: argparse.Namespace) -> int:
    root = root_of(args.root); target, identifier = selected(root, args.plan_id, args.use_active); task = target / "task_plan.md"
    if not task.is_file(): raise PlanError("cannot attest a missing task plan")
    value = {"plan_id": identifier, "task_plan": str(task.relative_to(root)), "sha256": sha(task), "attested_at": utc()}
    write_json(attest_path(root, identifier), value); output(value, args.format); return 0


def do_verify(args: argparse.Namespace) -> int:
    root = root_of(args.root); target, identifier = selected(root, args.plan_id, args.use_active); path = attest_path(root, identifier)
    if not path.is_file(): raise PlanError("no attestation exists")
    task = target / "task_plan.md"; value = read_json(path); matches = task.is_file() and value.get("sha256") == sha(task)
    output({"plan_id": identifier, "matches": matches, "attested_at": value.get("attested_at")}, args.format); return 0 if matches else 1


def manifest(root: Path) -> dict[str, Any]:
    values: dict[str, Any] = {}
    for path in root.rglob("*"):
        if not path.is_file(): continue
        relative = path.relative_to(root)
        if any(part in EXCLUDED or part.startswith(".env") for part in relative.parts) or path.stat().st_size > 2_000_000: continue
        values[str(relative)] = {"size": path.stat().st_size, "sha256": sha(path)}
    return values


def do_snapshot(args: argparse.Namespace) -> int:
    root = root_of(args.root); identifier = plan_id(args.snapshot_id) if args.snapshot_id else datetime.now(timezone.utc).strftime("snapshot-%Y%m%d-%H%M%S")
    path = planning(root) / "snapshots" / f"{identifier}.json"
    if path.exists() and not args.force: raise PlanError("snapshot exists; choose another ID or pass --force")
    value = {"snapshot_id": identifier, "created_at": utc(), "files": manifest(root)}; write_json(path, value); output({"snapshot_id": identifier, "files": len(value["files"])}, args.format); return 0


def do_reconcile(args: argparse.Namespace) -> int:
    root = root_of(args.root); identifier = plan_id(args.snapshot_id); path = planning(root) / "snapshots" / f"{identifier}.json"
    if not path.is_file(): raise PlanError("selected snapshot does not exist")
    prior = read_json(path).get("files", {}); current = manifest(root)
    if not isinstance(prior, dict): raise PlanError("snapshot manifest is invalid")
    result = {"snapshot_id": identifier, "added": sorted(set(current) - set(prior)), "removed": sorted(set(prior) - set(current)), "changed": sorted(name for name in set(current) & set(prior) if current[name] != prior[name])}
    output(result, args.format); return 0 if not any(result[key] for key in ("added", "removed", "changed")) else 1


def redact(text: str, maximum: int) -> str:
    lines = ["[redacted sensitive-looking line]" if any(word in line.lower() for word in SENSITIVE) else line for line in text.splitlines()]
    return "\n".join(lines)[:maximum]


def do_context(args: argparse.Namespace) -> int:
    root = root_of(args.root); target, identifier = selected(root, args.plan_id, args.use_active)
    sections = {name: redact(read_text(path), args.max_chars) if path.is_file() else "[missing]" for name, path in records(target).items()}
    output({"plan_id": identifier, "plan_dir": str(target), "sections": sections}, args.format); return 0


def do_doctor(args: argparse.Namespace) -> int:
    root = root_of(args.root); pointer = planning(root) / "active-plan.json"; result: dict[str, Any] = {"root": str(root), "root_records": inspect(root), "active_pointer": "absent", "problems": []}
    if pointer.exists():
        try:
            identifier = plan_id(str(read_json(pointer).get("plan_id", ""))); target = planning(root) / "plans" / identifier
            result["active_pointer"] = {"plan_id": identifier, "exists": target.is_dir()}
            if not target.is_dir(): result["problems"].append("active pointer references a missing plan")
        except PlanError as error:
            result["active_pointer"] = "invalid"; result["problems"].append(str(error))
    output(result, args.format); return 0 if not result["problems"] else 1


def parser() -> argparse.ArgumentParser:
    base = argparse.ArgumentParser(description="Explicit local planning utilities."); base.add_argument("--format", choices=("text", "json"), default="text"); commands = base.add_subparsers(dest="command", required=True)
    names = ("init", "resolve", "set-active", "status", "check-complete", "ledger-append", "ledger-summary", "attest", "verify-attestation", "snapshot", "reconcile", "render-context", "doctor")
    for name in names:
        item = commands.add_parser(name); item.add_argument("--root", required=True); item.add_argument("--format", choices=("text", "json"), default=argparse.SUPPRESS)
    commands.choices["init"].add_argument("--plan-id"); commands.choices["init"].add_argument("--force", action="store_true")
    for name in ("resolve", "status", "check-complete", "ledger-append", "attest", "verify-attestation", "render-context"):
        commands.choices[name].add_argument("--plan-id"); commands.choices[name].add_argument("--use-active", action="store_true")
    commands.choices["set-active"].add_argument("--plan-id", required=True); commands.choices["ledger-append"].add_argument("--kind", required=True); commands.choices["ledger-append"].add_argument("--message", required=True)
    commands.choices["snapshot"].add_argument("--snapshot-id"); commands.choices["snapshot"].add_argument("--force", action="store_true"); commands.choices["reconcile"].add_argument("--snapshot-id", required=True); commands.choices["render-context"].add_argument("--max-chars", type=int, default=2000)
    return base


def main() -> int:
    args = parser().parse_args(); actions = {"init": do_init, "resolve": do_resolve, "set-active": do_set_active, "status": do_status, "check-complete": do_complete, "ledger-append": do_ledger_append, "ledger-summary": do_ledger_summary, "attest": do_attest, "verify-attestation": do_verify, "snapshot": do_snapshot, "reconcile": do_reconcile, "render-context": do_context, "doctor": do_doctor}
    try: return actions[args.command](args)
    except PlanError as error: print(f"planning utility error: {error}", file=sys.stderr); return 2


if __name__ == "__main__":
    raise SystemExit(main())
