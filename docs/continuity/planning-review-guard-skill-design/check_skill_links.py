from __future__ import annotations

import json
import re
from pathlib import Path


ROOT = Path("/home/ubuntu/skills")
OUT = Path("/home/ubuntu/review-guard-skill-design/skill-link-audit.json")
LINK_RE = re.compile(r"\[[^\]]*\]\(([^)]+)\)")


def main() -> None:
    records = []
    for skill_file in sorted(ROOT.glob("*/SKILL.md")):
        text = skill_file.read_text(encoding="utf-8")
        for line_no, line in enumerate(text.splitlines(), start=1):
            for target in LINK_RE.findall(line):
                target = target.strip().strip("<>")
                if target.startswith(("http://", "https://", "mailto:", "#")):
                    continue
                path_part = target.split("#", 1)[0]
                if not path_part:
                    continue
                resolved = (skill_file.parent / path_part).resolve()
                records.append(
                    {
                        "skill": skill_file.parent.name,
                        "line": line_no,
                        "target": target,
                        "resolved": str(resolved),
                        "exists": resolved.exists(),
                    }
                )
    OUT.write_text(json.dumps(records, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({
        "checked_relative_links": len(records),
        "missing_relative_links": [record for record in records if not record["exists"]],
    }, indent=2))


if __name__ == "__main__":
    main()
