# INTEGIN Open-Source Building-Block Research — 2026-08-18

**Status:** Reference review only. No external repository, model, package, or service was cloned, executed, or adopted.

| Capability | Candidate | License / fit | Recommendation |
| --- | --- | --- | --- |
| Basic image and PDF OCR | [Tesseract](https://github.com/tesseract-ocr/tesseract) | Apache-2.0; established OCR engine with text, hOCR, PDF, TSV, ALTO, and PAGE output. | Use as a benchmark or lightweight deterministic fallback. Validate Arabic and industrial-nameplate accuracy on tenant-approved test material. |
| Advanced OCR and document layout | [PaddleOCR](https://github.com/PaddlePaddle/PaddleOCR) | Apache-2.0 codebase; supports structured PDF/image extraction and multilingual OCR. | Strong candidate for later self-hosted OCR service. Review each chosen model’s license, compute profile, and Arabic/nameplate accuracy before adoption. |
| Complex-document parsing | [Docling](https://github.com/docling-project/docling) | MIT codebase; local/air-gapped execution and structured document representation. | Consider for PDF/certificate/report parsing, not as the authoritative record system. Check model licenses independently. |
| Charts and dashboards | [Apache ECharts](https://github.com/apache/echarts) | Apache-2.0 JavaScript charting library. | Good UI building block for deterministic, tenant-scoped KPI dashboards; retain server-side metric calculation. |
| QR and barcode capture | [ZXing](https://github.com/zxing/zxing) | Mature multi-format reader; official Java project is maintenance-mode. | Use as a reference only. At implementation time, select an actively maintained, license-reviewed Flutter/native binding. |
| Image pre-processing | [OpenCV](https://github.com/opencv/opencv) | Apache-2.0 computer-vision library. | Optional preprocessing for deskew, crop, contrast, and plate-region workflows; it must not make safety/compliance decisions. |
| Local advisory inference | [LocalAI](https://github.com/mudler/localai) | MIT; self-hosted multi-modal serving surface. | Candidate only after the deterministic ingestion/risk model exists. Enforce a network boundary, model-license review, logging/provenance, and `blocking=false`. |
| Lightweight local LLM serving | [Ollama](https://github.com/ollama/ollama) | MIT; local model REST API. | Suitable for development/evaluation, but do not make a model server part of the authority path. Review model license and data-handling policy separately. |

## Repository adoption rules

1. Do **not** copy a whole inspection, ERP, EHS, or certificate platform. Its data model, tenant model, license, and authority semantics will not match INTEGIN.
2. Adopt only a bounded library or service behind an INTEGIN adapter. Preserve tenant scope, immutable source evidence, human confirmation, audit history, and server authority.
3. Pin an exact release/commit, preserve license and notices, maintain an SBOM/dependency record, scan for vulnerabilities, and create contract tests before production use.
4. Treat model weights and datasets as separate licensing/security decisions from the source repository.
5. Never execute repository-provided setup scripts or Docker commands without a reviewed INTEGIN deployment plan.

## Research sources

- PaddleOCR repository: structured image/PDF extraction and Apache-2.0 statement.
- Docling repository: local execution, document parsing scope, MIT statement, and model-license caveat.
- Tesseract repository: OCR scope/output formats and Apache-2.0 statement.
- Apache ECharts repository: browser visualization scope and Apache-2.0 statement.
- ZXing repository: multi-format capture scope and maintenance-mode status.
- OpenCV repository: Apache-2.0 computer-vision building block.
- LocalAI and Ollama repositories: self-hosted advisory inference options.
