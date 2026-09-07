from __future__ import annotations

import json
import re
from collections import defaultdict
from pathlib import Path


SKILLS_ROOT = Path("/home/ubuntu/skills")
OUTPUT_DIR = Path("/home/ubuntu/review-guard-skill-design")
PLACEHOLDER_RE = re.compile(r"\[TODO:|\bTODO\b", re.IGNORECASE)
SKILL_PATH_RE = re.compile(r"/home/ubuntu/skills/([a-z0-9-]+)/")
WORD_RE = re.compile(r"[a-zA-Z][a-zA-Z-]{2,}")


def frontmatter_and_body(text: str) -> tuple[dict[str, str], str]:
    if not text.startswith("---\n"):
        return {}, text
    parts = text.split("---\n", 2)
    if len(parts) < 3:
        return {}, text
    metadata: dict[str, str] = {}
    for line in parts[1].splitlines():
        if ":" in line:
            key, value = line.split(":", 1)
            metadata[key.strip()] = value.strip()
    return metadata, parts[2]


def token_set(text: str) -> set[str]:
    stop_words = {
        "this", "that", "with", "from", "when", "where", "which", "will", "into",
        "skill", "use", "for", "and", "the", "your", "are", "not", "all", "can",
        "must", "should", "only", "before", "after", "through", "without",
    }
    return {word.lower() for word in WORD_RE.findall(text) if word.lower() not in stop_words}


def markdown_escape(value: str) -> str:
    return value.replace("|", "\\|").replace("\n", " ")


def main() -> None:
    records: list[dict[str, object]] = []
    duplicate_names: defaultdict[str, list[str]] = defaultdict(list)
    all_paths = sorted(SKILLS_ROOT.glob("*/SKILL.md"))

    for skill_file in all_paths:
        text = skill_file.read_text(encoding="utf-8")
        metadata, body = frontmatter_and_body(text)
        slug = skill_file.parent.name
        description = metadata.get("description", "")
        headings = re.findall(r"^#{1,3}\s+(.+)$", body, flags=re.MULTILINE)
        references = sorted(set(SKILL_PATH_RE.findall(text)))
        missing_references = [ref for ref in references if not (SKILLS_ROOT / ref).exists()]
        example_files = sorted(
            path.relative_to(skill_file.parent).as_posix()
            for path in skill_file.parent.rglob("*")
            if path.is_file() and (path.name.startswith("example") or "TODO" in path.name)
        )
        name = metadata.get("name", "")
        duplicate_names[name].append(slug)
        records.append(
            {
                "slug": slug,
                "path": str(skill_file),
                "name": name,
                "description": description,
                "line_count": len(text.splitlines()),
                "word_count": len(WORD_RE.findall(text)),
                "headings": headings,
                "has_frontmatter": bool(metadata),
                "has_required_name": bool(name),
                "has_required_description": bool(description),
                "contains_placeholder": bool(PLACEHOLDER_RE.search(text)),
                "example_files": example_files,
                "referenced_skills": references,
                "missing_referenced_skills": missing_references,
                "token_set": sorted(token_set(description + " " + " ".join(headings))),
            }
        )

    pairs: list[dict[str, object]] = []
    for index, left in enumerate(records):
        left_tokens = set(left["token_set"])
        for right in records[index + 1 :]:
            right_tokens = set(right["token_set"])
            union = left_tokens | right_tokens
            score = len(left_tokens & right_tokens) / len(union) if union else 0.0
            if score >= 0.22:
                pairs.append({"left": left["slug"], "right": right["slug"], "score": round(score, 3)})

    duplicate_name_values = {name: slugs for name, slugs in duplicate_names.items() if name and len(slugs) > 1}
    summary = {
        "skill_count": len(records),
        "missing_frontmatter": [record["slug"] for record in records if not record["has_frontmatter"]],
        "missing_name": [record["slug"] for record in records if not record["has_required_name"]],
        "missing_description": [record["slug"] for record in records if not record["has_required_description"]],
        "placeholder_skills": [record["slug"] for record in records if record["contains_placeholder"]],
        "long_skills": [record["slug"] for record in records if int(record["line_count"]) > 500],
        "missing_references": {
            record["slug"]: record["missing_referenced_skills"]
            for record in records
            if record["missing_referenced_skills"]
        },
        "duplicate_names": duplicate_name_values,
        "similarity_candidates": sorted(pairs, key=lambda pair: float(pair["score"]), reverse=True),
    }
    OUTPUT_DIR.mkdir(parents=True, exist_ok=True)
    (OUTPUT_DIR / "skill-audit-metadata.json").write_text(
        json.dumps({"summary": summary, "records": records}, indent=2), encoding="utf-8"
    )

    report = [
        "# Installed Skill Static Audit Inventory",
        "",
        "This inventory is a read-only structural scan. It does not prove domain correctness, runtime behavior, or external-source currency.",
        "",
        "## Summary",
        "",
        f"- Installed skills: **{summary['skill_count']}**",
        f"- Missing required metadata: **{len(summary['missing_name']) + len(summary['missing_description'])}**",
        f"- Placeholder-bearing skills: **{len(summary['placeholder_skills'])}**",
        f"- Skills above 500 lines: **{len(summary['long_skills'])}**",
        f"- Missing internal skill references: **{len(summary['missing_references'])}**",
        f"- Duplicate metadata names: **{len(summary['duplicate_names'])}**",
        "",
        "## Per-skill structural inventory",
        "",
        "| Skill | Lines | Description | Placeholders | Example scaffold files | Missing internal skill references |",
        "|---|---:|---|---|---|---|",
    ]
    for record in records:
        report.append(
            "| {slug} | {lines} | {description} | {placeholder} | {examples} | {missing} |".format(
                slug=record["slug"],
                lines=record["line_count"],
                description=markdown_escape(str(record["description"])) or "—",
                placeholder="yes" if record["contains_placeholder"] else "no",
                examples=", ".join(record["example_files"]) or "—",
                missing=", ".join(record["missing_referenced_skills"]) or "—",
            )
        )

    report.extend(["", "## Similarity candidates", ""])
    if pairs:
        report.extend(["| Skill A | Skill B | Metadata/heading token overlap |", "|---|---|---:|"])
        for pair in summary["similarity_candidates"]:
            report.append(f"| {pair['left']} | {pair['right']} | {pair['score']:.3f} |")
    else:
        report.append("No high-overlap metadata/heading candidates were detected by this narrow lexical heuristic.")

    (OUTPUT_DIR / "skill-audit-inventory.md").write_text("\n".join(report) + "\n", encoding="utf-8")


if __name__ == "__main__":
    main()
