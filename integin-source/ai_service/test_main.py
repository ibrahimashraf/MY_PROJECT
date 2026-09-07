from fastapi.testclient import TestClient

from main import app, build_advisory


client = TestClient(app)


def test_health_and_contract():
    response = client.get("/healthz")
    assert response.status_code == 200
    assert response.json()["contract_version"] == "v1"

    response = client.post(
        "/v1/advisory",
        json={
            "version": "v1",
            "tenant_id": "tenant-1",
            "zone": "MONITORING",
            "lens": "TREND_ANOMALY",
            "evidence_refs": ["event-1"],
        },
    )
    payload = response.json()
    assert response.status_code == 200
    assert payload["tenant_id"] == "tenant-1"
    assert payload["blocking"] is False
    assert payload["rationale"]
    assert payload["evidence_refs"] == ["event-1"]


def test_ai_free_zone_never_generates_an_advisory():
    response = client.post(
        "/v1/advisory",
        json={"version": "v1", "tenant_id": "tenant-1", "zone": "CERTIFICATE"},
    )
    assert response.status_code == 403


def test_contract_and_input_limits_are_explicit():
    response = client.post(
        "/v1/advisory",
        json={"version": "v0", "tenant_id": "tenant-1", "zone": "MONITORING"},
    )
    assert response.status_code == 400

    oversized = {f"field-{index}": index for index in range(65)}
    response = client.post(
        "/v1/advisory",
        json={
            "version": "v1",
            "tenant_id": "tenant-1",
            "zone": "MONITORING",
            "inputs": oversized,
        },
    )
    assert response.status_code == 413


def test_direct_builder_keeps_primary_boundary_false():
    result = build_advisory(
        type("Request", (), {
            "version": "v1",
            "tenant_id": "tenant-1",
            "zone": "REGULATION",
            "lens": "COMPLIANCE_GAP",
            "inputs": {},
            "evidence_refs": [],
        })()
    )
    assert result.blocking is False
