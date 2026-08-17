# INTEGIN AI Advisory Service

This optional FastAPI service exposes the versioned INTEGIN advisory contract. It is intentionally deterministic by default so the service can be deployed and tested without an external model provider. A future provider adapter may enrich the output, but every response remains constrained to the advisory contract and `blocking=false`.

## Run locally

```bash
python -m venv .venv
.venv/Scripts/activate
pip install -r requirements.txt
uvicorn main:app --host 0.0.0.0 --port 8000
```

The health endpoint is `GET /healthz`. Advisory requests use `POST /v1/advisory` and must include a tenant, an allowed zone (`MONITORING` or `REGULATION`), and the `v1` contract version.

## Test

```bash
pytest -q
```

The service rejects verdict, certificate, calibration, authorization, sync-security, public-QR, and inspection-approval zones before producing any response. It does not mutate INTEGIN state and does not receive credentials or hidden prompts.
