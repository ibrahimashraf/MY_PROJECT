import base64
import json
import pathlib

# 1x1 transparent PNG (67 bytes) — placeholder, not real NDT
PNG_B64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+ip1sAAAAASUVORK5CYII="
PNG = base64.b64decode(PNG_B64)

DEFECT_TYPES = [
    ("ndt", "PAUT"),
    ("ndt", "EDDY_CURRENT"),
    ("ndt", "PULSED_EDDY_CURRENT"),
    ("ndt", "TOFD"),
    ("ndt", "RT"),
    ("ndt", "MT"),
    ("ndt", "PT"),
    ("ndt", "ET"),
    ("ndt", "MFL"),
    ("ndt", "AE"),
    ("lifting", "WIRE_ROPE"),
    ("lifting", "SHACKLE"),
]

def main() -> None:
    root = pathlib.Path(__file__).parent
    meta = []
    for cat, dt in DEFECT_TYPES:
        d = root / cat / dt
        d.mkdir(parents=True, exist_ok=True)
        for i in range(500):
            p = d / f"{dt}_{i:04d}.png"
            p.write_bytes(PNG)
            meta.append(
                {
                    "image": (pathlib.Path(cat) / dt / p.name).as_posix(),
                    "defect_type": dt,
                    "category": cat,
                    "inspector": "synthetic-placeholder",
                    "standard": "ISO 17635/ASME V",
                    "label": f"{dt} indication",
                    "split": "train" if i < 400 else "held-out",
                }
            )
    (root / "metadata.json").write_text(json.dumps(meta, indent=2))
    print(f"generated {len(meta)} images — 500 per defect type (400 train + 100 held-out)")

if __name__ == "__main__":
    main()
