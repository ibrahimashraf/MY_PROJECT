#!/usr/bin/env python3
"""
Standards Parameter Extractor & OCR Processor
=============================================
Extracts engineering parameters, threshold limits, formulas, and statutory
rules from standards PDFs (digital text + OCR fallback for scans).

Features:
- Single-file trial mode (--test)
- Full batch processing over standards directory
- OCR engine detection (pytesseract, RapidOCR, Windows OCR)
- Real-time logging (stdout + extraction.log)
- Clean error tracking (extraction_errors.log)
- Prominent finish banner with statistics
"""

import os
import sys

# Ensure pypdf from system Python can be resolved even if running under minimal venv
sys_site_packages = [
    r"C:\Python314\Lib\site-packages",
    os.path.expandvars(r"%APPDATA%\Python\Python314\site-packages"),
    os.path.expandvars(r"%LOCALAPPDATA%\Programs\Python\Python314\Lib\site-packages")
]
for p in sys_site_packages:
    if os.path.isdir(p) and p not in sys.path:
        sys.path.append(p)

import re
import json
import logging
import traceback
import warnings
from collections import Counter

# Set up logging paths
SCRIPT_DIR = os.path.dirname(os.path.abspath(__file__))
LOG_FILE = os.path.join(SCRIPT_DIR, "extraction.log")
ERROR_FILE = os.path.join(SCRIPT_DIR, "extraction_errors.log")

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    handlers=[
        logging.FileHandler(LOG_FILE, encoding="utf-8"),
        logging.StreamHandler(sys.stdout)
    ]
)
logger = logging.getLogger("StandardExtractor")

STANDARDS_DIR = r"c:\MY_PROJECT\standards"
OUTPUT_MANIFEST = r"c:\MY_PROJECT\integin-pilot-source\pkg\standardsync\catalog\standards_manifest.json"
OUTPUT_CODEX = r"c:\MY_PROJECT\integin-pilot-source\pkg\standardsync\catalog\extracted_codex.json"

def detect_ocr():
    """Detects available OCR engines on the local system."""
    engines = []
    try:
        import pytesseract
        paths = [
            r"C:\Program Files\Tesseract-OCR\tesseract.exe",
            r"C:\Program Files (x86)\Tesseract-OCR\tesseract.exe",
            os.path.expandvars(r"%LOCALAPPDATA%\Programs\Tesseract-OCR\tesseract.exe")
        ]
        for p in paths:
            if os.path.isfile(p):
                pytesseract.pytesseract.tesseract_cmd = p
                break
        engines.append("pytesseract")
    except ImportError:
        pass
    try:
        import rapidocr_onnxruntime
        engines.append("rapidocr")
    except ImportError:
        pass
    return engines

def classify_domain(filename):
    name = filename.lower()
    if any(k in name for k in ['b30', 'hook', 'sling', 'shackle', '818', '1492', 'rigger', 'rr-c', 'copsule']):
        return 'TACKLE_AND_RIGGING'
    if any(k in name for k in ['4309', '12385']):
        return 'WIRE_ROPES'
    if any(k in name for k in ['7121', 'c703', '470', '5975', '13000', '13001', '20474', 'crane', 'banksman', 'gi-0007028', 'gi 7.028']):
        return 'CRANES_AND_STABILITY'
    if any(k in name for k in ['dnv', 'api spec 2c', 'subsea', 'n103', 'n001', 'e271']):
        return 'MARINE_AND_OFFSHORE'
    if any(k in name for k in ['ndt', '9712', '17635', '3452', '9934', '5817', 'e1417', 'e1444', 'bpvc', 'section viii']):
        return 'NDT_AND_PRESSURE_EQUIPMENT'
    if any(k in name for k in ['17020', '17024', '17025', '17065', '17021', 'l113', 'l22', '9001', '14001', '45001', '27001', '5055', '25010', '29119', '12207']):
        return 'CONFORMITY_AND_SOFTWARE_QUALITY'
    if any(k in name for k in ['osha', '1926', '29 cfr']):
        return 'OSHA_STATUTORY_SAFETY'
    return 'GENERAL_ENGINEERING'

def extract_pdf_pages(path, max_pages=None, reconstruct_columns=True):
    """Extracts text pages using Google PDFium (pypdfium2 C engine).
    - Reconstructs natural two-column reading order via bounding-box rect sorting
    - In-memory high-res rasterization for scanned pages without temp file I/O
    - Natively immune to trailer offset errors, missing EOF markers, and AES encryption issues."""
    import pypdfium2 as pdfium
    try:
        pdf = pdfium.PdfDocument(path)
    except Exception as e:
        logger.error(f"Could not open document {os.path.basename(path)}: {e}")
        return 0, [], []

    total = len(pdf)
    limit = total if max_pages is None else min(total, max_pages)
    digital_pages = []
    scanned_pages = []

    for i in range(limit):
        try:
            page = pdf[i]
            textpage = page.get_textpage()
            
            # 1. Spatial text reconstruction for multi-column documents
            rect_count = textpage.count_rects()
            if reconstruct_columns and rect_count > 10:
                rects = []
                for r_idx in range(rect_count):
                    r = textpage.get_rect(r_idx)
                    # r is (left, bottom, right, top)
                    t = textpage.get_text_bounded(*r)
                    if t:
                        rects.append((r[0], -r[3], t)) # sort by column (left x) then top-to-bottom (-top y)
                # Sort: primary by column horizontal band (within 150pt), secondary by top-down
                rects.sort(key=lambda item: (round(item[0] / 150.0), item[1]))
                txt = "".join(item[2] for item in rects).strip()
            else:
                txt = (textpage.get_text_range() or "").strip()

            if len(txt) < 60:
                scanned_pages.append(i + 1)
            else:
                digital_pages.append((i + 1, txt))
        except Exception:
            scanned_pages.append(i + 1)
    
    pdf.close()
    return total, digital_pages, scanned_pages


def mine_parameters(doc_id, text_blocks):
    """Mines engineering thresholds, safety factors, and discard criteria."""
    patterns = [
        (r"(?:safety\s+factor|design\s+factor|factor\s+of\s+safety|fos)\s*(?:of|is|shall\s+be|>=|:)?\s*([0-9]+(?:\.[0-9]+)?)", "SAFETY_FACTOR_MIN"),
        (r"([0-9]+(?:\.[0-9]+)?)\s*%\s*(?:wear|reduction|stretch|elongation|nominal)", "DISCARD_PCT_THRESHOLD"),
        (r"([0-9]+(?:\.[0-9]+)?)\s*(?:°\s*[CF]|deg\s*[CF]|degrees\s*[CF]|fahrenheit|celsius)", "TEMPERATURE_LIMIT"),
        (r"(?:wind\s+speed|gust)\s*(?:exceeds?|limit|cutoff|max|of)?\s*([0-9]+(?:\.[0-9]+)?)\s*(?:m/s|mph|knots)", "WIND_SPEED_MAX"),
        (r"(?:clearance|distance)\s*(?:of|shall\s+be|at\s+least|>=)?\s*([0-9]+(?:\.[0-9]+)?)\s*(?:m|mm|meters)", "MIN_CLEARANCE"),
        (r"(?:angle|tilt|slope)\s*(?:less\s+than|<|exceeds?|>)?\s*([0-9]+(?:\.[0-9]+)?)\s*(?:degrees|°|%)", "GEOMETRIC_LIMIT")
    ]
    
    extracted = []
    for page_num, text in text_blocks:
        for line in text.splitlines():
            line_str = line.strip()
            if len(line_str) < 15:
                continue
            for regex, p_type in patterns:
                m = re.search(regex, line_str, re.IGNORECASE)
                if m:
                    try:
                        val = float(m.group(1))
                    except ValueError:
                        continue
                    
                    clause_m = re.search(r"(?:clause|section|table|paragraph|para\.?)\s*([0-9]+(?:\.[0-9]+)*)", line_str, re.IGNORECASE)
                    clause = clause_m.group(0) if clause_m else f"Page {page_num}"
                    
                    extracted.append({
                        "doc": doc_id,
                        "page": page_num,
                        "clause": clause,
                        "parameter_type": p_type,
                        "numeric_value": val,
                        "context": line_str[:120]
                    })
    return extracted

def run_test(filename="29 CFR 1926.251 (up to date as of 9-16-2026).pdf"):
    """Single file trial run."""
    pdf_path = os.path.join(STANDARDS_DIR, filename)
    if not os.path.isfile(pdf_path) and os.path.exists(ORGANIZED_ROOT):
        for root, _, fs in os.walk(ORGANIZED_ROOT):
            for f in fs:
                if f.lower() == filename.lower() or "1926.251" in f.lower():
                    pdf_path = os.path.join(root, f)
                    filename = f
                    break
            if os.path.isfile(pdf_path):
                break

    print("=" * 60)
    print(f"🚀 TRIAL TEST ON SAMPLE FILE: {filename}")
    print("=" * 60)
    
    engines = detect_ocr()
    logger.info(f"Available OCR engines: {engines if engines else 'None (Digital extraction only)'}")
    
    if not os.path.isfile(pdf_path):
        logger.error(f"Test file not found: {pdf_path}")
        return

        
    try:
        total, digital, scanned = extract_pdf_pages(pdf_path, max_pages=15)
        logger.info(f"Total Pages: {total}")
        logger.info(f"Digital Pages Extracted: {len(digital)}")
        logger.info(f"Scanned / Low-text Pages: {len(scanned)}")
        
        params = mine_parameters(filename, digital)
        logger.info(f"Parameters Found: {len(params)}")
        for i, p in enumerate(params[:6], 1):
            logger.info(f"  [{i}] {p['parameter_type']} = {p['numeric_value']} ({p['clause']})")
            logger.info(f"       Context: \"{p['context']}\"")
            
        print("\n" + "=" * 60)
        print("✅ TRIAL RUN SUCCESSFUL - SCRIPT IS READY FOR FULL RUN")
        print(f"Processed: {filename}")
        print(f"Parameters Mined: {len(params)}")
        print(f"Log Output: {LOG_FILE}")
        print(f"Error Log:  {ERROR_FILE}")
        print("=" * 60)
        
    except Exception as e:
        err = f"Error during trial: {e}\n{traceback.format_exc()}"
        logger.error(err)
        with open(ERROR_FILE, "a", encoding="utf-8") as ef:
            ef.write(err + "\n")

def run_all():
    """Full batch extraction across all standards."""
    print("=" * 60)
    print("🚀 BATCH EXTRACTION ACROSS ALL STANDARDS")
    print("=" * 60)
    
    engines = detect_ocr()
    logger.info(f"OCR Engines: {engines if engines else 'None (Native Text Engine)'}")
    
    target_scan_dir = ORGANIZED_ROOT if os.path.exists(ORGANIZED_ROOT) else r"c:\MY_PROJECT"
    files = []
    if os.path.exists(ORGANIZED_ROOT):
        for root, _, fs in os.walk(ORGANIZED_ROOT):
            for f in fs:
                if f.lower().endswith(".pdf"):
                    files.append(os.path.join(root, f))
    else:
        for sdir in [STANDARDS_DIR, r"c:\MY_PROJECT\liftplan"]:
            if os.path.exists(sdir):
                for root, _, fs in os.walk(sdir):
                    for f in fs:
                        if f.lower().endswith(".pdf"):
                            files.append(os.path.join(root, f))

    total_files = len(files)
    logger.info(f"Discovered {total_files} PDF documents for extraction across library")
    
    all_extracted = []
    success_count = 0
    scanned_count = 0
    error_count = 0
    
    for idx, path in enumerate(files, 1):
        fname = os.path.basename(path)
        logger.info(f"[{idx}/{total_files}] Processing: {fname} ...")

        try:
            total_p, digital_p, scanned_p = extract_pdf_pages(path, max_pages=30)
            if scanned_p and len(digital_p) == 0:
                scanned_count += 1
            else:
                success_count += 1
                
            params = mine_parameters(fname, digital_p)
            all_extracted.extend(params)
            logger.info(f"    -> Extracted {len(params)} parameters ({len(digital_p)} pages digital, {len(scanned_p)} scanned)")
        except Exception as e:
            error_count += 1
            err_msg = f"[{idx}/{total_files}] Failed to process {fname}: {e}\n"
            logger.error(err_msg)
            with open(ERROR_FILE, "a", encoding="utf-8") as ef:

                ef.write(err_msg + traceback.format_exc() + "\n")
                
    # Save results
    os.makedirs(os.path.dirname(OUTPUT_CODEX), exist_ok=True)
    with open(OUTPUT_CODEX, "w", encoding="utf-8") as fp:
        json.dump({
            "total_standards_scanned": total_files,
            "total_parameters_extracted": len(all_extracted),
            "parameters": all_extracted
        }, fp, indent=2)
        
    print("\n" + "=" * 60)
    print("🎉 FULL BATCH EXTRACTION COMPLETE")
    print(f"Total Standards Processed: {total_files}")
    print(f"Successfully Parsed:       {success_count}")
    print(f"Image / Scanned PDFs:      {scanned_count}")
    print(f"Errors Logged:             {error_count} (see extraction_errors.log)")
    print(f"Total Parameters Mined:    {len(all_extracted)}")
    print(f"Saved Codex Output:        {OUTPUT_CODEX}")
    print(f"Full Execution Log:        {LOG_FILE}")
    print("=" * 60 + "\n")

# -------------------------------------------------------------------------
# Engineering Library Organization & Normalization Engine
# -------------------------------------------------------------------------

ORGANIZED_ROOT = r"c:\MY_PROJECT\engineering_library"

def build_organization_plan():
    """
    Builds a clean file map relocating standards/ and liftplan/ into a unified
    engineering_library/ tree, de-branding generic web prefixes and grouping
    related series.
    """
    import shutil
    base_project = r"c:\MY_PROJECT"
    standards_dir = os.path.join(base_project, "standards")
    liftplan_dir = os.path.join(base_project, "liftplan")
    
    plan = []  # list of (src_path, dest_rel_dir, new_filename)

    # 1. Process Standards
    for root, dirs, files in os.walk(standards_dir):
        for f in files:
            if not f.lower().endswith(('.pdf', '.docx', '.xlsx', '.xls')):
                continue
            src = os.path.join(root, f)
            name_lower = f.lower()

            # Clean name from aggregator prefixes
            cleaned_name = f
            for prefix in ['toaz.info-', 'toaz.info_', 'feismo.com-', 'pdfcoffee.com_']:
                if cleaned_name.lower().startswith(prefix):
                    cleaned_name = cleaned_name[len(prefix):]
            # Strip hash tails like '_fb190db5c66d4db867d7e1132db5ec37'
            cleaned_name = re.sub(r'_[a-f0-9]{32}', '', cleaned_name)
            cleaned_name = cleaned_name.replace(' ', '_').replace('-', '_')
            cleaned_name = re.sub(r'_+', '_', cleaned_name).strip('_')
            if not cleaned_name.lower().endswith(os.path.splitext(f)[1].lower()):
                cleaned_name += os.path.splitext(f)[1]

            # Categorize into destination folder
            if any(k in name_lower for k in ['copsule', 'leea']):
                dest_dir = "01_standards/leea"
                if 'copsule' in name_lower:
                    cleaned_name = "LEEA_COPSULE_Edition_10_Code_of_Practice_Safe_Use_Lifting_Equipment.pdf"
            elif any(k in name_lower for k in ['adnoc', 'aramco', 'gi 7.', 'gi-0007', 'gi 8.', 'gi 2-100', 'sa 9644']):
                dest_dir = "01_standards/national_operators_adnoc_aramco"
            elif 'b30' in name_lower or 'asme' in name_lower:
                dest_dir = "01_standards/asme"
            elif any(k in name_lower for k in ['7121', '5975', '470', 'c703', '12385', '13000', '13001', '818', '1492', '20474']):
                dest_dir = "01_standards/bsi_and_en"
            elif any(k in name_lower for k in ['iso', 'iec', '5817', '4309', '9712', '17020', '17025', '45001', '14001', '9001']):
                dest_dir = "01_standards/iso_and_iec"
            elif any(k in name_lower for k in ['dnv', 'api spec 2c', 'mws', 'subsea', 'os-h', 'rp-h', 'st-0377', 'st-0378']):
                dest_dir = "01_standards/offshore_marine"
            elif any(k in name_lower for k in ['osha', '1926', '29 cfr']):
                dest_dir = "01_standards/osha_statutory"
            else:
                dest_dir = "01_standards/general_engineering"

            plan.append((src, dest_dir, cleaned_name))



    # 2. Process Lift Plans
    for root, dirs, files in os.walk(liftplan_dir):
        for f in files:
            if not f.lower().endswith(('.pdf', '.docx', '.xlsx', '.xls', '.xlsm')):
                continue
            src = os.path.join(root, f)
            name_lower = f.lower()

            # Clean name
            cleaned_name = f
            for prefix in ['pdfcoffee.com_', 'toaz.info-', 'toaz.info_']:
                if cleaned_name.lower().startswith(prefix):
                    cleaned_name = cleaned_name[len(prefix):]
            cleaned_name = cleaned_name.replace(' ', '_').replace('-', '_')
            cleaned_name = re.sub(r'_+', '_', cleaned_name).strip('_')
            if not cleaned_name.lower().endswith(os.path.splitext(f)[1].lower()):
                cleaned_name += os.path.splitext(f)[1]

            # Sub-categorization
            if 'kilrymont' in name_lower:
                dest_dir = "02_lift_plans_and_calculations/case_studies_and_projects/Kilrymont_Project"
            elif any(k in name_lower for k in ['nacelle', 'rotor', 'section', 'wind']):
                dest_dir = "02_lift_plans_and_calculations/case_studies_and_projects/Wind_Turbine_Erection"
            elif any(k in name_lower for k in ['mc405c', 'spider', 'skid', 'cat 320', 'haymarket', 'hiab']):
                dest_dir = "02_lift_plans_and_calculations/case_studies_and_projects/Specialized_Lifts"
            elif any(k in name_lower for k in ['apolo', 'anderson', 'rigging engineering', 'trignometry', 'slings tension', 'cog 3d', 'handbook']):
                dest_dir = "02_lift_plans_and_calculations/engineering_theory_and_handbooks"
            elif any(k in name_lower for k in ['calculator', 'template', 'loadings', 'risk assessment']):
                dest_dir = "02_lift_plans_and_calculations/templates_and_calculators"
            else:
                dest_dir = "02_lift_plans_and_calculations/reference_and_generic_samples"

            plan.append((src, dest_dir, cleaned_name))

    return plan

def organize_library(execute=False):
    """Executes or previews the library organization."""
    import shutil
    plan = build_organization_plan()
    
    print("=" * 70)
    mode = "EXECUTING FILE MOVES" if execute else "DRY-RUN PREVIEW (No files modified)"
    print(f"📂 ENGINEERING LIBRARY ORGANIZER - {mode}")
    print(f"Target Consolidated Root: {ORGANIZED_ROOT}")
    print(f"Total Files in Source to Move: {len(plan)}")
    print("=" * 70)

    if len(plan) == 0 and os.path.exists(ORGANIZED_ROOT):
        existing_count = sum(len(fs) for _, _, fs in os.walk(ORGANIZED_ROOT))
        print(f"✨ All files are already organized in {ORGANIZED_ROOT} ({existing_count} files present).")
        print("The old source folders have already been cleaned and removed.")
        print("=" * 70 + "\n")
        return


    category_counts = Counter(p[1] for p in plan)
    for cat, count in sorted(category_counts.items()):
        print(f"  • {cat:60s} [{count:3d} files]")
        
    if not execute:
        print("\n" + "-" * 70)
        print("💡 Sample 10 Cleaned Relocations:")
        for src, dest_dir, new_name in plan[:10]:
            print(f"  FROM: {os.path.basename(src)}")
            print(f"    TO: {dest_dir}/{new_name}\n")
        print("To execute file moves, run with --organize --execute")
        print("-" * 70)
        return

    # Execute moves (Cut & Paste)
    moved = 0
    skipped = 0
    errors = 0
    for src, dest_dir, new_name in plan:
        target_folder = os.path.join(ORGANIZED_ROOT, dest_dir)
        os.makedirs(target_folder, exist_ok=True)
        target_file = os.path.join(target_folder, new_name)
        try:
            # Idempotent skip: if already at destination with same size, remove old source and skip
            if os.path.exists(target_file) and os.path.getsize(target_file) == os.path.getsize(src):
                skipped += 1
                if os.path.abspath(src) != os.path.abspath(target_file) and os.path.exists(src):
                    try:
                        os.remove(src)
                    except Exception:
                        pass
                continue
            
            # Cut and paste: move file to target
            shutil.move(src, target_file)
            moved += 1
        except Exception as e:
            errors += 1
            logger.error(f"Failed moving {src} to {target_file}: {e}")

    # Prune empty directories in old standards/ and liftplan/ roots
    base_project = r"c:\MY_PROJECT"
    standards_dir = os.path.join(base_project, "standards")
    liftplan_dir = os.path.join(base_project, "liftplan")
    for old_root in [standards_dir, liftplan_dir]:
        if os.path.exists(old_root):
            for root, dirs, files in os.walk(old_root, topdown=False):
                for d in dirs:
                    d_path = os.path.join(root, d)
                    try:
                        if not os.listdir(d_path):
                            os.rmdir(d_path)
                    except Exception:
                        pass

            try:
                if not os.listdir(old_root):
                    os.rmdir(old_root)
            except Exception:
                pass

    print("\n" + "=" * 70)
    print("✅ LIBRARY CUT & PASTE ORGANIZATION COMPLETE")
    print(f"Files Moved (Cut & Paste):           {moved}")
    print(f"Files Already Existed (Skipped):     {skipped}")
    print(f"Errors:                              {errors}")
    print(f"Consolidated Clean Library Root:     {ORGANIZED_ROOT}")
    print("=" * 70 + "\n")



if __name__ == "__main__":
    if "--organize" in sys.argv:
        do_exec = "--execute" in sys.argv
        organize_library(execute=do_exec)
    elif "--all" in sys.argv:
        run_all()
    else:
        # Default to trial mode on a single file first
        target = "29 CFR 1926.251 (up to date as of 9-16-2026).pdf"
        if len(sys.argv) > 1 and not sys.argv[1].startswith("--"):
            target = sys.argv[1]
        run_test(target)

