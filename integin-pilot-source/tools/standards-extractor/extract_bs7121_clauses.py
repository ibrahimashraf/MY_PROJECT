import os
import re
import json
import pypdf

STANDARDS_DIR = r"c:\MY_PROJECT\standards"

targets = [
    {
        "id": "BS 7121-1",
        "file": "BS 7121-1-2016.pdf",
        "keywords": ["categorization of lifts", "basic lift", "intermediate lift", "complex lift", "wind speed", "multiple lifting", "power line", "clearance"]
    },
    {
        "id": "BS 7121-2-1",
        "file": "BS 7121-2-1-2012.pdf",
        "keywords": ["thorough examination", "6 months", "12 months", "pre-use", "major review", "10 years", "supplementary test"]
    },
    {
        "id": "BS 7121-2-3",
        "file": "BS 7121 - 2-3.pdf",
        "keywords": ["slew ring", "backlash", "wear pad", "outrigger", "overload test", "four years", "4-yearly", "NDT"]
    },
    {
        "id": "BS 7121-2-4",
        "file": "BS 7121-2-4 2025.pdf",
        "keywords": ["stabilizer", "interlock", "overload protection", "hose burst", "12 months", "6 months", "slew limit"]
    },
    {
        "id": "BS 7121-2-7",
        "file": "BS 7121-2-7-2012 A2-2022.pdf",
        "keywords": ["buffer", "brake test", "rail", "span", "overload limiter", "hook throat", "125%"]
    },
    {
        "id": "BS 7121-2-9",
        "file": "BS 7121-2-9-2013-09.pdf",
        "keywords": ["spreader", "twistlock", "rail clamp", "storm", "wind", "cycle", "fatigue"]
    },
    {
        "id": "BS 7121-3",
        "file": "BS 7121-3-2017+A1-2019.pdf",
        "keywords": ["outrigger", "ground bearing", "mat", "pick-and-carry", "wind", "derating", "slew"]
    },
    {
        "id": "BS 7121-4",
        "file": "BS 7121- 4.pdf",
        "keywords": ["stabilizer", "tilt", "chassis", "bolt torque", "remote control", "level"]
    },
    {
        "id": "BS 7121-5",
        "file": "BS 7121-5-2019.pdf",
        "keywords": ["weather-vaning", "free slew", "anemometer", "climbing", "jumping", "tie-in", "anti-collision", "clearance"]
    },
    {
        "id": "BS 7121-13",
        "file": "BS 7121-13-2009.pdf",
        "keywords": ["differential height", "deflection", "slope", "locking pin", "hydraulic", "crosshead", "settlement"]
    },
    {
        "id": "BS 7121-14",
        "file": "BS 7121-14-2005.pdf",
        "keywords": ["tipping", "counterweight", "slope", "rated capacity", "reeving", "lowering-in"]
    },
]

results = {}

for target in targets:
    fpath = os.path.join(STANDARDS_DIR, target["file"])
    if not os.path.exists(fpath):
        continue
    reader = pypdf.PdfReader(fpath)
    print(f"\nProcessing {target['id']} ({len(reader.pages)} pages)...")
    matches = []
    for idx, page in enumerate(reader.pages):
        text = page.extract_text() or ""
        for kw in target["keywords"]:
            if re.search(r'\b' + re.escape(kw) + r'\b', text, re.IGNORECASE):
                # find context snippet
                for line in text.splitlines():
                    if re.search(r'\b' + re.escape(kw) + r'\b', line, re.IGNORECASE):
                        matches.append({
                            "page": idx + 1,
                            "keyword": kw,
                            "line": line.strip()[:150]
                        })
                        if len(matches) > 30:
                            break
            if len(matches) > 30:
                break
        if len(matches) > 30:
            break
    results[target["id"]] = matches
    print(f"  Found {len(matches)} occurrences")

with open(r"c:\MY_PROJECT\integin-pilot-source\tools\standards-extractor\bs7121_snippets.json", "w", encoding="utf-8") as f:
    json.dump(results, f, indent=2)
print("Saved bs7121_snippets.json")
