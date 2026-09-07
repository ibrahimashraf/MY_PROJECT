from __future__ import annotations

from pathlib import Path


SKILLS_ROOT = Path("/home/ubuntu/skills")
OUTPUT = Path("/home/ubuntu/review-guard-skill-design/SKILL_FULL_COLLECTION_REGISTRY_2026-08-24.md")


def read_frontmatter_value(text: str, key: str) -> str:
    if not text.startswith("---\n"):
        return ""
    end = text.find("\n---", 4)
    if end == -1:
        return ""
    for line in text[4:end].splitlines():
        if line.startswith(f"{key}:"):
            return line.split(":", 1)[1].strip().strip('"')
    return ""


def classify(slug: str, text: str) -> tuple[str, str, str, str]:
    if slug in {"integin-review-governance", "integin-evidence-guard"}:
        return (
            "Local session-created clean-room package",
            "Created in this session; no external upstream",
            "INTEGIN workflow guidance",
            "Internal revision eligible; validate locally after each edit",
        )
    if slug == "planning-with-files":
        return (
            "Identified external repository: OthmanAdi/planning-with-files",
            "External source identity verified; file-level source coverage pending",
            "Strong multi-host hook and plugin dependency",
            "Blocked pending clean-room source audit and explicit host-compatibility decision",
        )
    if slug.startswith("webdev-") or slug.startswith("manus-") or slug in {
        "automation-and-scheduling",
        "data-backup-restoration",
        "persistent-computing",
        "skill-creator",
    }:
        return (
            "No package-level source record found",
            "Likely platform-managed from local naming/instructions; ownership unverified",
            "Platform-coupled or likely platform-coupled",
            "Blocked pending official source of record and target-host compatibility evidence",
        )
    if slug.startswith("integin-"):
        return (
            "No package-level source record found",
            "INTEGIN-named local package; ownership and upstream status unverified",
            "INTEGIN engineering workflow guidance",
            "Blocked for upstream update; local revision needs a separate requirement/evidence record",
        )
    return (
        "No package-level source record found",
        "Unattributed installed package",
        "Dependency not proven by naming alone",
        "Blocked pending source of record, owner, and compatibility target",
    )


def main() -> None:
    rows: list[str] = []
    for skill in sorted(SKILLS_ROOT.glob("*/SKILL.md")):
        slug = skill.parent.name
        text = skill.read_text(encoding="utf-8")
        name = read_frontmatter_value(text, "name") or "—"
        description = read_frontmatter_value(text, "description").replace("|", "\\|") or "—"
        provenance, ownership, host, eligibility = classify(slug, text)
        rows.append(
            f"| `{slug}` | {name} | {provenance} | {ownership} | {host} | {eligibility} |"
        )

    document = [
        "# Full Installed-Skill Registry",
        "",
        "This register classifies all installed skills from direct local evidence only. A missing source record is not proof that a package has no upstream; it means no reproducible update route was found in the local package.",
        "",
        "| Skill | Declared name | Provenance evidence | Ownership classification | Host dependency | Safe update eligibility |",
        "|---|---|---|---|---|---|",
        *rows,
        "",
        "## Classification rules",
        "",
        "A package is externally update-eligible only when its local version, authoritative source, immutable source revision, and target-host compatibility are known. All other packages remain blocked for external update, even if their instructions contain example URLs or version-like text.",
        "",
        "The two session-created INTEGIN skills are the only packages for which a clean-room local origin is directly established by this task record. That does not establish authority over other INTEGIN-named packages.",
    ]
    OUTPUT.write_text("\n".join(document) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
