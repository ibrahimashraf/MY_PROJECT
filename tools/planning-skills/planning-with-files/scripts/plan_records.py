#!/usr/bin/env python3
"""Explicit local planning-record utilities with no network or host-hook behavior."""

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
PLAN_ID_RE = re.compile(r"^[a-z0-9][a-z0-9-]{0,63}$")
EXCLUDED_PARTS = {".git", ".planning", "node_modules", "private", "dist", "build", "coverage", "__pycache__"}
SENSITIVE_TERMS = ("password", "secret", "token", "authorization", "cookie", "private_key", "-----begin")


class PlanningError(Exception):
    pass


def now() -> str:
    return datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")


def sha256_file(path: Path) -> str:
    digest = hashlib.sha256()
    with path.open("rb") as handle:
        for block in iter(lambda: handle.read(131072), b""):
            digest.update(block)
    return digest.hexdigest()


def parse_root(raw: str) -> Path:
    root = Path(raw).expanduser()
    if not root.is_dir():
        raise PlanningError(f"root is not an accessible directory: {root}")
    return root.resolve()


def validate_plan_id(plan_id: str) -> str:
    if not PLAN_ID_RE.fullmatch(plan_id):
        raise PlanningError("plan ID must be lowercase hyphen-case and at most 64 characters")
    return plan_id


def state_dir(root: Path) -> Path:
    return root / ".planning"


def ensure_under_root(root: Path, path: Path) -> Path:
    resolved = path.resolve()
    try:
        resolved.relative_to(root)
    except ValueError as error:
        raise PlanningError("resolved path escapes the selected root") from error
    return resolved


def write_text_atomic(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    with tempfile.NamedTemporaryFile("w", encoding="utf-8", delete=False, dir=path.parent) as handle:
        handle.write(content)
        temporary = Path(handle.name)
    os.replace(temporary, path)


def write_json_atomic(path: Path, data: Any) -> None:
    write_text_atomic(path, json.dumps(data, indent=2, sort_keys=True) + "\n")


def read_json(path: Path) -> dict[str, Any]:
    try:
        value = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as error:
        raise PlanningError(f"cannot read JSON state: {path}") from error
    if not isinstance(value, dict):
        raise PlanningError(f"JSON state is not an object: {path}")
    return value


def template_path(filename: str) -> Path:
    return Path(__file__).resolve().parents[1] / "templates" / filename


def select_plan(root: Path, plan_id: str | None, use_active: bool) -> tuple[Path, str]:
    if plan_id:
        identifier = validate_plan_id(plan_id)
        target = ensure_under_root(root, state_dir(root) / "plans" / identifier)
        if not target.is_dir():
            raise PlanningError(f"selected plan does not exist: {identifier}")
        return target, identifier
    if use_active:
        pointer = state_dir(root) / "active-plan.json"
        if not pointer.is_file():
            raise PlanningError("no active-plan pointer exists; pass --plan-id or omit --use-active")
        identifier = validate_plan_id(str(read_json(pointer).get("plan_id", "")))
        target = ensure_under_root(root, state_dir(root) / "plans" / identifier)
        if not target.is_dir():
            raise PlanningError("active-plan pointer references a missing plan")
        return target, identifier
    return root, "root"


def record_paths(plan_dir: Path) -> dict[str, Path]:
    return {name: plan_dir / name for name in RECORDS}


def inspect_records(plan_dir: Path) -> dict[str, Any]:
    result: dict[str, Any] = {"plan_dir": str(plan_dir), "records": {}}
    for name, path in record_paths(plan_dir).items():
        exists = path.is_file()
        result["records"][name] = {
            "exists": exists,
            "nonempty": bool(path.read_text(encoding="utf-8").strip()) if exists else False,
        }
    return result


def emit(data: Any, output: str) -> None:
    if output == "json":
        print(json.dumps(data, indent=2, sort_keys=True))
        return
    if isinstance(data, dict):
        for key, value in data.items():
            print(f"{key}: {value}")
    else:
        print(data)


def command_init(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    if args.plan_id:
        identifier = validate_plan_id(args.plan_id)
        target = state_dir(root) / "plans" / identifier
        target.mkdir(parents=True, exist_ok=True)
    else:
        target, identifier = root, "root"
    target = ensure_under_root(root, target)
    existing = [name for name, path in record_paths(target).items() if path.is_file() and path.read_text(encoding="utf-8").strip()]
    if existing and not args.force:
        raise PlanningError("refusing to overwrite nonempty planning records: " + ", ".join(existing))
    for name in RECORDS:
        source = template_path(name)
        if not source.is_file():
            raise PlanningError(f"missing local template: {name}")
        write_text_atomic(target / name, source.read_text(encoding="utf-8"))
    emit({"initialized": identifier, "plan_dir": str(target), "records": list(RECORDS)}, args.format)
    return 0


def command_resolve(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    plan_dir, identifier = select_plan(root, args.plan_id, args.use_active)
    emit({"root": str(root), "plan_id": identifier, "plan_dir": str(plan_dir)}, args.format)
    return 0


def command_set_active(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    identifier = validate_plan_id(args.plan_id)
    plan_dir, _ = select_plan(root, identifier, False)
    records = inspect_records(plan_dir)
    missing = [name for name, item in records["records"].items() if not item["exists"]]
    if missing:
        raise PlanningError("refusing to activate incomplete plan: " + ", ".join(missing))
    write_json_atomic(state_dir(root) / "active-plan.json", {"plan_id": identifier, "updated_at": now()})
    emit({"active_plan": identifier, "plan_dir": str(plan_dir)}, args.format)
    return 0


def task_status(path: Path) -> dict[str, int]:
    if not path.is_file():
        return {"complete": 0, "remaining": 0}
    content = path.read_text(encoding="utf-8")
    return {"complete": len(re.findall(r"(?m)^\s*- \[x\]", content, flags=re.IGNORECASE)), "remaining": len(re.findall(r"(?m)^\s*- \[ \]", content))}


def command_status(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    plan_dir, identifier = select_plan(root, args.plan_id, args.use_active)
    result = inspect_records(plan_dir)
    result.update({"plan_id": identifier, "task_status": task_status(plan_dir / "task_plan.md")})
    emit(result, args.format)
    return 0


def command_check_complete(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    plan_dir, identifier = select_plan(root, args.plan_id, args.use_active)
    status = task_status(plan_dir / "task_plan.md")
    complete = status["complete"] > 0 and status["remaining"] == 0
    emit(
        {
            "plan_id": identifier,
            "all_phases_complete": complete,
            "completed_items": status["complete"],
            "remaining_items": status["remaining"],
        },
        args.format,
    )
    return 0 if complete else 1


def command_ledger_append(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    _, identifier = select_plan(root, args.plan_id, args.use_active)
    entry = {"time": now(), "plan_id": identifier, "kind": args.kind, "message": args.message}
    path = state_dir(root) / "ledger.jsonl"
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("a", encoding="utf-8") as handle:
        handle.write(json.dumps(entry, sort_keys=True) + "\n")
    emit(entry, args.format)
    return 0


def command_ledger_summary(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    path = state_dir(root) / "ledger.jsonl"
    entries: list[dict[str, Any]] = []
    malformed = 0
    if path.is_file():
        for line in path.read_text(encoding="utf-8").splitlines():
            try:
                value = json.loads(line)
                if isinstance(value, dict):
                    entries.append(value)
                else:
                    malformed += 1
            except json.JSONDecodeError:
                malformed += 1
    kinds = Counter(str(entry.get("kind", "unknown")) for entry in entries)
    emit({"entries": len(entries), "kinds": dict(sorted(kinds.items())), "malformed": malformed}, args.format)
    return 0 if malformed == 0 else 1


def attest_path(root: Path, identifier: str) -> Path:
    return state_dir(root) / "attestations" / f"{identifier}.json"


def command_attest(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    plan_dir, identifier = select_plan(root, args.plan_id, args.use_active)
    plan = plan_dir / "task_plan.md"
    if not plan.is_file():
        raise PlanningError("cannot attest a missing task_plan.md")
    record = {"plan_id": identifier, "task_plan": str(plan.relative_to(root)), "sha256": sha256_file(plan), "attested_at": now()}
    write_json_atomic(attest_path(root, identifier), record)
    emit(record, args.format)
    return 0


def command_verify_attestation(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    plan_dir, identifier = select_plan(root, args.plan_id, args.use_active)
    path = attest_path(root, identifier)
    if not path.is_file():
        raise PlanningError("no attestation exists for the selected plan")
    record = read_json(path)
    plan = plan_dir / "task_plan.md"
    matches = plan.is_file() and record.get("sha256") == sha256_file(plan)
    emit({"plan_id": identifier, "matches": matches, "attested_at": record.get("attested_at")}, args.format)
    return 0 if matches else 1


def relative_manifest(root: Path) -> dict[str, dict[str, Any]]:
    files: dict[str, dict[str, Any]] = {}
    for path in root.rglob("*"):
        if not path.is_file():
            continue
        relative = path.relative_to(root)
        if any(part in EXCLUDED_PARTS or part.startswith(".env") for part in relative.parts):
            continue
        if path.stat().st_size > 2_000_000:
            continue
        files[str(relative)] = {"size": path.stat().st_size, "sha256": sha256_file(path)}
    return files


def command_snapshot(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    identifier = validate_plan_id(args.snapshot_id) if args.snapshot_id else datetime.now(timezone.utc).strftime("snapshot-%Y%m%d-%H%M%S")
    path = state_dir(root) / "snapshots" / f"{identifier}.json"
    if path.exists() and not args.force:
        raise PlanningError("snapshot already exists; choose another ID or pass --force")
    record = {"snapshot_id": identifier, "created_at": now(), "files": relative_manifest(root)}
    write_json_atomic(path, record)
    emit({"snapshot_id": identifier, "files": len(record["files"])}, args.format)
    return 0


def command_reconcile(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    identifier = validate_plan_id(args.snapshot_id)
    path = state_dir(root) / "snapshots" / f"{identifier}.json"
    if not path.is_file():
        raise PlanningError("selected snapshot does not exist")
    prior = read_json(path).get("files", {})
    if not isinstance(prior, dict):
        raise PlanningError("snapshot has an invalid file manifest")
    current = relative_manifest(root)
    added = sorted(set(current) - set(prior))
    removed = sorted(set(prior) - set(current))
    changed = sorted(name for name in set(current) & set(prior) if current[name] != prior[name])
    result = {"snapshot_id": identifier, "added": added, "removed": removed, "changed": changed}
    emit(result, args.format)
    return 0 if not (added or removed or changed) else 1


def redact(text: str, limit: int) -> str:
    lines: list[str] = []
    for line in text.splitlines():
        lines.append("[redacted sensitive-looking line]" if any(term in line.lower() for term in SENSITIVE_TERMS) else line)
        if len("\n".join(lines)) >= limit:
            break
    return "\n".join(lines)[:limit]


def command_render_context(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    plan_dir, identifier = select_plan(root, args.plan_id, args.use_active)
    sections: dict[str, str] = {}
    for name, path in record_paths(plan_dir).items():
        sections[name] = redact(path.read_text(encoding="utf-8"), args.max_chars) if path.is_file() else "[missing]"
    emit({"plan_id": identifier, "plan_dir": str(plan_dir), "sections": sections}, args.format)
    return 0


def command_doctor(args: argparse.Namespace) -> int:
    root = parse_root(args.root)
    result: dict[str, Any] = {"root": str(root), "root_records": inspect_records(root), "active_pointer": "absent", "problems": []}
    pointer = state_dir(root) / "active-plan.json"
    if pointer.exists():
        try:
            data = read_json(pointer)
            identifier = validate_plan_id(str(data.get("plan_id", "")))
            target = state_dir(root) / "plans" / identifier
            result["active_pointer"] = {"plan_id": identifier, "exists": target.is_dir()}
            if not target.is_dir():
                result["problems"].append("active pointer references a missing plan")
        except PlanningError as error:
            result["active_pointer"] = "invalid"
            result["problems"].append(str(error))
    emit(result, args.format)
    return 0 if not result["problems"] else 1


def add_selection(parser: argparse.ArgumentParser) -> None:
    parser.add_argument("--plan-id")
    parser.add_argument("--use-active", action="store_true")


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(description="Explicit local utilities for planning records.")
    parser.add_argument("--format", choices=("text", "json"), default="text")
    commands = parser.add_subparsers(dest="command", required=True)
    for name in ("init", "resolve", "set-active", "status", "check-complete", "ledger-append", "ledger-summary", "attest", "verify-attestation", "snapshot", "reconcile", "render-context", "doctor"):
        child = commands.add_parser(name)
        child.add_argument("--root", required=True)
        child.add_argument("--format", choices=("text", "json"), default=argparse.SUPPRESS)
    commands.choices["init"].add_argument("--plan-id")
    commands.choices["init"].add_argument("--force", action="store_true")
    for name in ("resolve", "status", "check-complete", "ledger-append", "attest", "verify-attestation", "render-context"):
        add_selection(commands.choices[name])
    commands.choices["set-active"].add_argument("--plan-id", required=True)
    commands.choices["ledger-append"].add_argument("--kind", required=True)
    commands.choices["ledger-append"].add_argument("--message", required=True)
    commands.choices["snapshot"].add_argument("--snapshot-id")
    commands.choices["snapshot"].add_argument("--force", action="store_true")
    commands.choices["reconcile"].add_argument("--snapshot-id", required=True)
    commands.choices["render-context"].add_argument("--max-chars", type=int, default=2000)
    return parser


def main() -> int:
    args = build_parser().parse_args()
    try:
        actions = {
            "init": command_init,
            "resolve": command_resolve,
            "set-active": command_set_active,
            "status": command_status,
            "check-complete": command_check_complete,
            "ledger-append": command_ledger_append,
            "ledger-summary": command_ledger_summary,
            "attest": command_attest,
            "verify-attestation": command_verify_attestation,
            "snapshot": command_snapshot,
            "reconcile": command_reconcile,
            "render-context": command_render_context,
            "doctor": command_doctor,
        }
        return actions[args.command](args)
    except PlanningError as error:
        print(f"planning utility error: {error}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
