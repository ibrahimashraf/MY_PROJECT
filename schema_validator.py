import re
import json
from typing import List, Optional, Dict, Any
from pydantic import BaseModel, Field

class LoadChartEntry(BaseModel):
    radius_m: float
    boom_length_m: Optional[float] = None
    capacity_tonnes: float
    proof_load_tonnes: Optional[float] = None
    notes: Optional[str] = ""
    constraint_flags: List[str] = Field(default_factory=list)

class InspectionChecklist(BaseModel):
    breadcrumb: str
    category: str
    item: str
    status: str
    comments: Optional[str] = ""

class CraneCertificateSchema(BaseModel):
    report_number: Optional[str] = None
    equipment_type: Optional[str] = None
    manufacturer: Optional[str] = None
    model: Optional[str] = None
    serial_number: Optional[str] = None
    chassis_number: Optional[str] = None
    plate_number: Optional[str] = None
    date_of_examination: Optional[str] = None
    next_examination_date: Optional[str] = None
    inspector_name: Optional[str] = None
    authenticator_name: Optional[str] = None
    safe_to_operate: Optional[str] = None
    checklist: List[InspectionChecklist] = Field(default_factory=list)
    load_chart: List[LoadChartEntry] = Field(default_factory=list)
    figures: List[Dict[str, Any]] = Field(default_factory=list)
    raw_tables: List[List[List[Optional[str]]]] = Field(default_factory=list)
    review_flags: List[str] = Field(default_factory=list)

def validate_load_chart_monotonicity(entries: List[LoadChartEntry]) -> List[str]:
    """Validates physics rule: for a given boom length, capacity must not increase as radius increases."""
    flags = []
    by_boom: Dict[float, List[LoadChartEntry]] = {}
    for e in entries:
        b = e.boom_length_m or 0.0
        by_boom.setdefault(b, []).append(e)

    for boom, pts in by_boom.items():
        sorted_pts = sorted(pts, key=lambda x: x.radius_m)
        for i in range(len(sorted_pts) - 1):
            curr_pt = sorted_pts[i]
            next_pt = sorted_pts[i + 1]
            if next_pt.radius_m > curr_pt.radius_m and next_pt.capacity_tonnes > curr_pt.capacity_tonnes:
                flags.append(
                    f"PHYSICAL VIOLATION: Boom {boom}m: capacity jumped from "
                    f"{curr_pt.capacity_tonnes}t at {curr_pt.radius_m}m to "
                    f"{next_pt.capacity_tonnes}t at {next_pt.radius_m}m."
                )
    return flags

def extract_domain_schema(tables: List[List[List[Optional[str]]]], text_corpus: str) -> CraneCertificateSchema:
    schema = CraneCertificateSchema()
    schema.raw_tables = tables

    # 1. Metadata Extraction
    rep_match = re.search(r"Report Number:\s*\**([A-Za-z0-9\-]+)\**", text_corpus, re.IGNORECASE)
    if rep_match:
        schema.report_number = rep_match.group(1).strip()
    else:
        schema.review_flags.append("Missing or unreadable Report Number.")

    date_match = re.search(r"Date of (?:Through )?Examination:\s*([0-9\/\-]+)", text_corpus, re.IGNORECASE)
    if date_match:
        schema.date_of_examination = date_match.group(1).strip()

    next_date_match = re.search(r"(?:next thorough examination[^\n]*\n|Latest date by which next thorough[^\n]*\n)\s*\**([0-9\/\-]+)\**", text_corpus, re.IGNORECASE)
    if next_date_match:
        schema.next_examination_date = next_date_match.group(1).strip()

    mfg_match = re.search(r"Manufacturer:\s*\**([A-Za-z0-9\s]+?)(?:\n|\r|\*\*|<br>|$)", text_corpus, re.IGNORECASE)
    if mfg_match:
        schema.manufacturer = mfg_match.group(1).strip()

    model_match = re.search(r"Model:\s*\**([A-Za-z0-9\s]+?)(?:\n|\r|\*\*|<br>|$)", text_corpus, re.IGNORECASE)
    if model_match:
        schema.model = model_match.group(1).strip()

    sn_match = re.search(r"Serial Number:\s*\**([A-Za-z0-9\-]+)", text_corpus, re.IGNORECASE)
    if sn_match:
        schema.serial_number = sn_match.group(1).strip()

    chassis_match = re.search(r"Chassis Number:\s*\**([A-Za-z0-9\-]+)", text_corpus, re.IGNORECASE)
    if chassis_match:
        schema.chassis_number = chassis_match.group(1).strip()

    plate_match = re.search(r"Plate Number:\s*\**([A-Za-z0-9\s\-]+?)(?:\n|\r|\*\*|<br>|$)", text_corpus, re.IGNORECASE)
    if plate_match:
        schema.plate_number = plate_match.group(1).strip()

    equip_match = re.search(r"(?:Mobile Crane[^\n\r<*]+|Telescoping Boom)", text_corpus, re.IGNORECASE)
    if equip_match:
        schema.equipment_type = equip_match.group(0).strip()

    insp_match = re.search(r"person making this report:\s*\**([A-Za-z\s]+?)(?:\n|\r|\*\*|<br>|$)", text_corpus, re.IGNORECASE)
    if insp_match:
        schema.inspector_name = insp_match.group(1).strip()

    auth_match = re.search(r"Signature:\s*\**([A-Za-z\s]+?)(?:\n|\r|\*\*|<br>|$)", text_corpus, re.IGNORECASE)
    if auth_match:
        schema.authenticator_name = auth_match.group(1).strip()

    safe_match = re.search(r"IS THIS EQUIPMENT SAFE TO OPERATE\?[\s\S]*?(YES|NO)", text_corpus, re.IGNORECASE)
    if safe_match:
        schema.safe_to_operate = safe_match.group(1).strip().upper()

    # Global rigging deduction / configuration notes (e.g. outriggers fully extended)
    global_notes = []
    if re.search(r"fully extended outriggers", text_corpus, re.IGNORECASE):
        global_notes.append("Configuration: fully extended outriggers")
    if re.search(r"@6 Falls", text_corpus, re.IGNORECASE):
        global_notes.append("Reeving: @6 Falls")

    # 2. Table Parsing: Load Charts & Granular Inspection Checklists
    current_category = "General Inspection"
    for table in tables:
        boom_col, radius_col, swl_col, proof_col = None, None, None, None

        for row in table:
            cleaned = [c.replace("\n", " ").strip() if c else "" for c in row]
            if not any(cleaned):
                continue

            # Detect Category Header Rows
            cat_match = re.search(r"^(\d+\s*[–-]\s*[A-Z\s/]+)", cleaned[0] if cleaned[0] else (cleaned[1] if len(cleaned) > 1 else ""))
            if cat_match:
                current_category = cat_match.group(1).strip()

            # Detect Header Row for Load Charts
            row_str = " ".join(cleaned).lower()
            if "boom length" in row_str and ("radius" in row_str or "swl" in row_str):
                for idx, cell in enumerate(cleaned):
                    cell_lower = cell.lower()
                    if "boom length" in cell_lower:
                        boom_col = idx
                    elif "radius" in cell_lower:
                        radius_col = idx
                    elif "swl" in cell_lower or "capacity" in cell_lower:
                        swl_col = idx
                    elif "proof" in cell_lower:
                        proof_col = idx
                continue

            # Extract row data if load chart headers identified
            if radius_col is not None and swl_col is not None:
                try:
                    def parse_num(val):
                        if not val:
                            return None
                        m = re.search(r"(\d+(?:\.\d+)?)", val)
                        return float(m.group(1)) if m else None

                    boom_val = parse_num(cleaned[boom_col]) if boom_col is not None and boom_col < len(cleaned) else None
                    radius_val = parse_num(cleaned[radius_col]) if radius_col < len(cleaned) else None
                    swl_val = parse_num(cleaned[swl_col]) if swl_col < len(cleaned) else None
                    proof_val = parse_num(cleaned[proof_col]) if proof_col is not None and proof_col < len(cleaned) else None

                    # Check for asterisk, footnote, or description constraints
                    desc_cell = cleaned[0] if cleaned else ""
                    constraints = list(global_notes)
                    if "*" in "".join(cleaned):
                        constraints.append("Special constraint marker (*)")
                    if "min.radius" in desc_cell.lower():
                        constraints.append("Condition: Min.radius boundary")
                    elif "max. radius" in desc_cell.lower() or "max.radius" in desc_cell.lower():
                        constraints.append("Condition: Max.radius boundary")
                    elif "intermediate" in desc_cell.lower():
                        constraints.append("Condition: Intermediate radius")

                    if radius_val is not None and swl_val is not None:
                        schema.load_chart.append(
                            LoadChartEntry(
                                radius_m=radius_val,
                                boom_length_m=boom_val,
                                capacity_tonnes=swl_val,
                                proof_load_tonnes=proof_val,
                                notes=desc_cell,
                                constraint_flags=constraints
                            )
                        )
                except (ValueError, IndexError):
                    pass
                continue

            # Granular Checklist Item Detection with Hierarchical Breadcrumbs
            has_tick = any("" in c or c == "OK" for c in cleaned)
            has_no = any(c == "NO" for c in cleaned)
            has_na = any(c == "NA" for c in cleaned)

            if has_tick or has_no or has_na:
                item_title = cleaned[0] if cleaned[0] else (cleaned[1] if len(cleaned) > 1 else "")
                if item_title and not any(h in item_title.lower() for h in ["status", "ok", "comments"]) and not cat_match:
                    comment = ""
                    for cell in reversed(cleaned):
                        if cell and cell not in ["", "OK", "NO", "NA", ""]:
                            if cell != item_title:
                                comment = cell
                                break

                    status = "OK" if has_tick else ("FAIL" if has_no else "N/A")
                    breadcrumb_path = f"{current_category} > {item_title}"
                    schema.checklist.append(
                        InspectionChecklist(
                            breadcrumb=breadcrumb_path,
                            category=current_category,
                            item=item_title,
                            status=status,
                            comments=comment
                        )
                    )

    # 3. Validate Physics Monotonicity
    physics_flags = validate_load_chart_monotonicity(schema.load_chart)
    schema.review_flags.extend(physics_flags)

    return schema
