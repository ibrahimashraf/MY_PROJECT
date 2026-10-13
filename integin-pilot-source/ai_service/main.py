"""INTEGIN optional AI advisory service.

This service is intentionally secondary: it can describe signals, but it cannot
make or mutate any primary INTEGIN decision. The Go client remains the contract
authority and also re-validates the response.
"""

from datetime import datetime, timezone
from typing import Any

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

CONTRACT_VERSION = "v1"
AI_ALLOWED_ZONES = {"MONITORING", "REGULATION", "NDT_DEFECT", "LIFTING_DEFECT"}
AI_FREE_ZONES = {
    "VERDICT",
    "CERTIFICATE",
    "CALIBRATION",
    "AUTHORIZATION",
    "SYNC_SECURITY",
    "PUBLIC_QR",
    "INSPECTION_APPROVAL",
}


class AdvisoryRequest(BaseModel):
    version: str = CONTRACT_VERSION
    tenant_id: str = Field(min_length=1, max_length=128)
    zone: str = Field(min_length=1, max_length=64)
    lens: str = Field(default="", max_length=64)
    inputs: dict[str, Any] = Field(default_factory=dict)
    evidence_refs: list[str] = Field(default_factory=list, max_length=32)


class AdvisoryResponse(BaseModel):
    version: str
    tenant_id: str
    title: str
    summary: str
    severity: str
    confidence: float = Field(ge=0, le=1)
    rationale: str
    evidence_refs: list[str]
    provider: str
    model: str
    prompt_version: str
    limitations: list[str]
    blocking: bool = False
    created_at: datetime


app = FastAPI(title="INTEGIN AI Advisory Service", version=CONTRACT_VERSION)


def _lens_copy(lens: str) -> tuple[str, str, float]:
    copies = {
        "ASSET_INTEGRITY": (
            "Asset integrity signal",
            "A change in asset evidence distribution is worth human review.",
            0.62,
        ),
        "SAFETY_RISK": (
            "Safety risk signal",
            "Recent observations show a pattern that may deserve a closer look.",
            0.71,
        ),
        "COMPLIANCE_GAP": (
            "Compliance gap signal",
            "A standard or checklist reference may have changed.",
            0.78,
        ),
        "PROCESS_DEVIATION": (
            "Process deviation signal",
            "The observed workflow pattern differs from the recent baseline.",
            0.66,
        ),
        "DOCUMENTATION_QUALITY": (
            "Documentation quality signal",
            "Evidence notes are less complete than the selected comparison set.",
            0.74,
        ),
        "TREND_ANOMALY": (
            "Trend anomaly signal",
            "A recent measurement sits outside the selected rolling pattern.",
            0.81,
        ),
        "DATA_COMPLETENESS": (
            "Data completeness signal",
            "Some optional evidence fields may benefit from a completeness review.",
            0.69,
        ),
        # Generic INTEGIN inspection lenses — any current or future inspection type
        # is advisory via the default below; the 4 named lenses just give a
        # nicer title for common cases. New types need no code change.
        "NDT_MPI": (
            "NDT MPI indication — observation",
            "Magnetic particle indication observed — advisory, requires human confirmation with procedure + additional image.",
            0.73,
        ),
        "NDT_UT": (
            "NDT UT indication — observation",
            "Ultrasonic indication pattern observed — advisory, requires human confirmation.",
            0.71,
        ),
        "LIFTING_WIRE_ROPE": (
            "Lifting wire-rope wear — observation",
            "Surface wear / broken wire pattern observed — advisory, internal break not visible, requires human confirmation.",
            0.68,
        ),
        "LIFTING_SHACKLE": (
            "Lifting shackle wear — observation",
            "Shackle wear / deformation pattern observed — advisory, requires human confirmation.",
            0.66,
        ),
        "PAUT": (
            "PAUT indication — observation",
            "Phased Array Ultrasonic Testing indication observed — advisory, requires human confirmation with procedure.",
            0.72,
        ),
        "EDDY_CURRENT": (
            "Eddy current indication — observation",
            "Eddy current signal pattern observed — advisory, requires human confirmation.",
            0.70,
        ),
        "PULSED_EDDY_CURRENT": (
            "Pulsed eddy current indication — observation",
            "Pulsed eddy current wall-loss pattern observed — advisory, requires human confirmation.",
            0.69,
        ),
        "TOFD": (
            "TOFD indication — observation",
            "Time-of-Flight Diffraction indication observed — advisory, requires human confirmation.",
            0.71,
        ),
        "RT": (
            "Radiographic indication — observation",
            "Radiographic image indication observed — advisory, requires human confirmation.",
            0.70,
        ),
        "MT": (
            "Magnetic particle indication — observation",
            "MT indication observed — advisory, requires human confirmation.",
            0.73,
        ),
        "PT": (
            "Penetrant indication — observation",
            "Dye penetrant indication observed — advisory, requires human confirmation.",
            0.68,
        ),
        "ET": (
            "Electromagnetic testing indication — observation",
            "ET signal pattern observed — advisory, requires human confirmation.",
            0.69,
        ),
        "MFL": (
            "MFL indication — observation",
            "Magnetic Flux Leakage indication observed — advisory, requires human confirmation.",
            0.68,
        ),
        "AE": (
            "Acoustic emission indication — observation",
            "Acoustic emission event pattern observed — advisory, requires human confirmation.",
            0.67,
        ),
    }
    return copies.get(
        lens,
        ("Advisory signal", "A secondary pattern was detected for human review.", 0.55),
    )


def build_advisory(request: AdvisoryRequest) -> AdvisoryResponse:
    if request.version != CONTRACT_VERSION:
        raise HTTPException(status_code=400, detail="unsupported contract version")
    # Generic for all INTEGIN inspection types and all systems except AI-free zones.
    # AI_FREE_ZONES are the only forbidden zones — everything else is advisory-only.
    if request.zone in AI_FREE_ZONES:
        raise HTTPException(status_code=403, detail="AI invocation is forbidden in this zone")
    # Keep AI_ALLOWED_ZONES as the documented advisory zones, but also allow any future zone
    # that is not AI-free via the generic fallback — no code change for new INTEGIN types/systems.
    if len(request.inputs) > 64:
        raise HTTPException(status_code=413, detail="too many input fields")

    title, summary, confidence = _lens_copy(request.lens)
    evidence_count = len(request.evidence_refs)
    rationale = (
        f"The {request.lens or 'selected'} lens produced a secondary signal from "
        f"{evidence_count} evidence reference(s)."
    )
    return AdvisoryResponse(
        version=CONTRACT_VERSION,
        tenant_id=request.tenant_id,
        title=title,
        summary=summary,
        severity="ADVISORY",
        confidence=confidence,
        rationale=rationale,
        evidence_refs=list(request.evidence_refs),
        provider="python-deterministic",
        model="none",
        prompt_version="advisory/v1",
        limitations=[
            "This result is advisory only.",
            "It cannot change a primary INTEGIN workflow or decision.",
        ],
        blocking=False,
        created_at=datetime.now(timezone.utc),
    )


@app.get("/healthz")
def healthz() -> dict[str, str]:
    return {"status": "ok", "contract_version": CONTRACT_VERSION}


@app.post("/v1/advisory", response_model=AdvisoryResponse)
def advisory(request: AdvisoryRequest) -> AdvisoryResponse:
    return build_advisory(request)

