from __future__ import annotations

import hashlib
import json
import re
from pathlib import Path


ROOT = Path("/home/ubuntu/review-guard-skill-design")
MANIFEST = ROOT / "planning-v3.11.2-skill-artifacts.tsv"
SCRATCH = ROOT / "disposable-planning-v3.11.2-source"
REGISTER_JSON = ROOT / "planning-v3.11.2-artifact-register.json"
REGISTER_MD = ROOT / "planning-v3.11.2-artifact-register.md"
PREFIX = ".agents/skills/planning-with-files/"
SIGNALS = {
    "install_or_dependency": r"\b(npm|npx|pip|brew|apt|install|package manager)\b",
    "network_or_remote": r"\b(curl|wget|https?://|fetch|api)\b",
    "process_execution": r"\b(subprocess|os\.system|eval|exec|powershell|bash|sh )\b",
    "credential_or_secret": r"\b(token|secret|password|credential|api[_ -]?key)\b",
    "hook_or_automation": r"\b(hook|PreToolUse|PostToolUse|UserPromptSubmit|Stop)\b",
    "destructive_operation": r"\b(rm -rf|delete|truncate|overwrite|reset)\b",
}


def content_hash(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def main() -> None:
    records = []
    for line in MANIFEST.read_text(encoding="utf-8").splitlines():
        source_path, artifact_type, size, blob_sha = line.split("\t")
        if not source_path.startswith(PREFIX) or artifact_type != "blob":
            continue
        rel = source_path.removeprefix(PREFIX)
        target = SCRATCH / rel
        retrieved = target.is_file()
        data = target.read_bytes() if retrieved else b""
        text = data.decode("utf-8", errors="replace") if retrieved else ""
        signals = [name for name, pattern in SIGNALS.items() if re.search(pattern, text, flags=re.IGNORECASE)]
        records.append(
            {
                "source_path": source_path,
                "artifact_type": artifact_type,
                "source_size": int(size),
                "source_blob_sha": blob_sha,
                "retrieved": retrieved,
                "retrieved_sha256": content_hash(target) if retrieved else None,
                "line_count": len(text.splitlines()) if retrieved else 0,
                "static_signals": signals,
                "review_state": "scanned_static" if retrieved else "inaccessible",
                "semantic_limit": "Static scan only; it does not prove behavior, safety, or compatibility.",
            }
        )

    REGISTER_JSON.write_text(json.dumps({"artifact_count": len(records), "records": records}, indent=2) + "\n", encoding="utf-8")
    output = [
        "# Planning-with-Files v3.11.2: Clean-Room Artifact Register",
        "",
        "All records below describe an immutable upstream artifact from `.agents/skills/planning-with-files` at commit `9e94390e5912b1ff296556505cd999ff84838160`. The temporary copies were scanned statically only and were not executed, imported, installed, or retained as dependencies.",
        "",
        "| Source artifact | Bytes | Blob SHA | Retrieved | Static signal categories | Review state |",
        "|---|---:|---|---|---|---|",
    ]
    for record in records:
        output.append(
            "| `{path}` | {size} | `{sha}` | {retrieved} | {signals} | {state} |".format(
                path=record["source_path"],
                size=record["source_size"],
                sha=record["source_blob_sha"],
                retrieved="yes" if record["retrieved"] else "no",
                signals=", ".join(record["static_signals"]) or "none detected",
                state=record["review_state"],
            )
        )
    output.extend(
        [
            "",
            "## Interpretation limit",
            "",
            "A signal means only that a matching term or construct appears in an artifact. It is not evidence that an operation occurs, that credentials are present, that tests pass, or that the source is safe or compatible.",
        ]
    )
    REGISTER_MD.write_text("\n".join(output) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
