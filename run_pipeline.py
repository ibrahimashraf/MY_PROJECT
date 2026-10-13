import os
import sys
import json
import subprocess
import pdfplumber
import pymupdf4llm
from langchain_text_splitters import MarkdownHeaderTextSplitter
from fastembed import TextEmbedding
from qdrant_client import QdrantClient
from qdrant_client.models import PointStruct, VectorParams, Distance

def main():
    if len(sys.argv) < 2:
        print("Usage: python run_pipeline.py <path_to_pdf>")
        sys.exit(1)

    pdf_path = sys.argv[1]
    if not os.path.exists(pdf_path):
        print(f"Error: File '{pdf_path}' not found.")
        sys.exit(1)

    work_dir = "pipeline_output"
    os.makedirs(work_dir, exist_ok=True)

    # 1. Document Classifier (digital vs scanned)
    print(f"[*] Classifying: {pdf_path}")
    with pdfplumber.open(pdf_path) as pdf:
        sample_chars = sum(len(p.extract_text() or "") for p in pdf.pages[:3])
    is_scanned = sample_chars < 150
    print(f"[+] Document type: {'SCANNED (image-only)' if is_scanned else 'BORN-DIGITAL'}")

    # Configure Tesseract PATH & tessdata_best model
    tess_bin = r"C:\Program Files\Tesseract-OCR"
    if os.path.exists(tess_bin) and tess_bin not in os.environ.get("PATH", ""):
        os.environ["PATH"] = tess_bin + ";" + os.environ["PATH"]
    tessdata_best_dir = os.path.abspath("tessdata_best")
    if os.path.exists(tessdata_best_dir):
        os.environ["TESSDATA_PREFIX"] = tessdata_best_dir

    # 2. OCR if scanned (High-accuracy LSTM via tessdata_best + auto-deskew)
    target_pdf = pdf_path
    if is_scanned:
        ocr_pdf = os.path.join(work_dir, "ocr_output.pdf")
        print(f"[*] Running OCRmyPDF on CPU (tessdata_best LSTM + auto-deskew + clean)...")
        subprocess.run(
            ["ocrmypdf", "--skip-text", "--deskew", "--clean", "--tesseract-oem", "1", "--jobs", "4", pdf_path, ocr_pdf],
            check=True,
            env=os.environ
        )
        target_pdf = ocr_pdf

    # 3. Layout Parser (Markdown without writing disk images)
    print(f"[*] Extracting layout and markdown with PyMuPDF4LLM...")
    md_text = pymupdf4llm.to_markdown(target_pdf, write_images=False)

    # 3b. Zero-Storage Virtual Figure Indexer (Discard black masks/shadows)
    import pymupdf, re
    doc_mupdf = pymupdf.open(target_pdf)
    all_figures = []
    seen_xrefs = set()
    for p_num, page in enumerate(doc_mupdf, 1):
        for img_info in page.get_images(full=True):
            xref = img_info[0]
            if xref in seen_xrefs:
                continue
            seen_xrefs.add(xref)
            base_img = doc_mupdf.extract_image(xref)
            w, h = base_img["width"], base_img["height"]
            if w >= 150 and h >= 100:
                # Filter out pure black shadow masks / border frames
                pix = pymupdf.Pixmap(doc_mupdf, xref)
                step = 4 if pix.alpha else 3
                s = pix.samples
                avg_luma = (sum(s[0::step]) + sum(s[1::step]) + sum(s[2::step])) / (len(s) * 3 / step)
                if avg_luma < 5.0:
                    continue  # Skip pure black backdrop masks

                all_figures.append({
                    "page": p_num,
                    "xref": xref,
                    "width": w,
                    "height": h,
                    "format": base_img["ext"]
                })
    print(f"[+] Indexed {len(all_figures)} real inspection photos (discarded black masks).")

    # 3c. Authentic Arabic Text Preservation (Fix typos only; preserve ة, أ, إ, ى verbatim)
    arabic_re = re.compile(r'[\u0600-\u06FF\u0750-\u077F\u08A0-\u08FF]')
    pristine_arabic = []
    for page in doc_mupdf:
        for b in page.get_text("blocks"):
            t = b[4].strip()
            if arabic_re.search(t):
                pristine_arabic.append(t)

    lines = md_text.splitlines()
    reconciled_lines = []
    p_idx = 0
    for line in lines:
        if arabic_re.search(line):
            if p_idx < len(pristine_arabic):
                if not (reconciled_lines and arabic_re.search(reconciled_lines[-1])):
                    clean_ar = pristine_arabic[p_idx].replace("\n \n", "\n").replace("\n", " ")
                    # Fix source typist error while preserving authentic ة, أ, إ, ى
                    clean_ar = clean_ar.replace("المقاوالت", "المقاولات").replace("مقاوالت", "مقاولات")
                    reconciled_lines.append(f"**{clean_ar}**")
                    p_idx += 1
            continue
        reconciled_lines.append(line)

    # 3d. Inline Base64 Photo Embedding (Renders images in MD with 0 disk image files)
    import base64
    reconciled_lines.append("\n## Embedded Inspection Photos & Diagrams (Inline Base64)\n")
    for fig in all_figures:
        base_img = doc_mupdf.extract_image(fig["xref"])
        b64 = base64.b64encode(base_img["image"]).decode("utf-8")
        reconciled_lines.append(f"![Figure Page {fig['page']} (xref {fig['xref']})](data:image/{fig['format']};base64,{b64})\n")

    md_text = "\n".join(reconciled_lines)
    md_file = os.path.join(work_dir, "corpus.md")
    with open(md_file, "w", encoding="utf-8") as f:
        f.write(md_text)
    print(f"[+] Markdown corpus saved with {len(all_figures)} inline Base64 figures to: {md_file}")

    # 4. Table Extraction & Pydantic Schema Validation (Engineering Rules)
    print("[*] Extracting tables & running numeric physics validation...")
    extracted_tables = []
    with pdfplumber.open(target_pdf) as pdf:
        for page in pdf.pages:
            tables = page.extract_tables()
            if tables:
                extracted_tables.extend(tables)

    from schema_validator import extract_domain_schema
    schema_data = extract_domain_schema(extracted_tables, md_text)
    schema_data.figures = all_figures
    json_path = os.path.join(work_dir, "structured_document.json")
    with open(json_path, "w", encoding="utf-8") as f:
        f.write(schema_data.model_dump_json(indent=2))
    print(f"[+] Structured JSON schema saved to: {json_path}")

    # Human Review Queue for low confidence / physics violations
    review_path = os.path.join(work_dir, "review_queue.json")
    if schema_data.review_flags:
        with open(review_path, "w", encoding="utf-8") as f:
            json.dump({"pdf": pdf_path, "flags": schema_data.review_flags}, f, indent=2)
        print(f"[!] WARNING: {len(schema_data.review_flags)} items flagged for human review -> {review_path}")
    else:
        print("[✓] Validation passed: 0 review flags.")

    # 5. Structure-Aware Chunker (Strip Base64 payloads to keep vector embeddings fast and compact)
    print("[*] Sanitizing Base64 payloads & chunking markdown by headers...")
    clean_rag_md = re.sub(r'data:image/[^;]+;base64,[A-Za-z0-9+/=]+', '[Embedded Image Data]', md_text)
    headers = [("#", "Header 1"), ("##", "Header 2"), ("###", "Header 3")]
    splitter = MarkdownHeaderTextSplitter(headers_to_split_on=headers)
    docs = splitter.split_text(clean_rag_md)
    chunk_items = []
    for d in (docs if docs else []):
        bc = " > ".join(str(v) for v in d.metadata.values() if v)
        chunk_items.append({
            "text": d.page_content,
            "breadcrumb": bc if bc else "Document Root"
        })
    if not chunk_items:
        chunk_items = [{"text": clean_rag_md, "breadcrumb": "Document Root"}]

    print(f"[+] Generated {len(chunk_items)} structural chunks with clause breadcrumbs.")

    # 5. Local Vector Store (Qdrant on-disk + FastEmbed ONNX CPU)
    print("[*] Computing CPU ONNX embeddings & indexing into Qdrant...")
    qdrant_path = os.path.join(work_dir, "qdrant_db")
    client = QdrantClient(path=qdrant_path)
    if client.collection_exists(collection_name="pdf_docs"):
        client.delete_collection(collection_name="pdf_docs")
    client.create_collection(
        collection_name="pdf_docs",
        vectors_config=VectorParams(size=384, distance=Distance.COSINE),
    )
    embed_model = TextEmbedding(model_name="BAAI/bge-small-en-v1.5")
    texts_to_embed = [item["text"] for item in chunk_items]
    vectors = list(embed_model.embed(texts_to_embed))

    points = [
        PointStruct(
            id=i,
            vector=v.tolist(),
            payload={
                "chunk_id": i,
                "breadcrumb": item["breadcrumb"],
                "text": item["text"],
                "source": pdf_path
            }
        )
        for i, (v, item) in enumerate(zip(vectors, chunk_items))
    ]
    client.upsert(collection_name="pdf_docs", points=points)
    print(f"[+] Successfully indexed {len(points)} points into {qdrant_path}")
    print("[✓] Pipeline execution finished successfully.")

if __name__ == "__main__":
    main()
