# INTEGIN AI Advisory Deployment

## Service Topology

The optional AI service is a separate Python process. The Go INTEGIN application remains the authority for inspections, certificates, authorization, calibration, offline synchronization, QR projection, and all primary state transitions. The Go client sends only approved advisory requests to the versioned `/v1/advisory` endpoint and receives normalized advisory output.

```text
Go INTEGIN application
  ├── deterministic domain and platform engines
  ├── internal/advisory          advisory-only contract and AI-free zones
  ├── internal/aiintegration     versioned HTTP client and health checks
  └── internal/advisorview       secondary advisor-panel view models
              │
              └── optional HTTP → Python AI advisory service
                                  ├── monitoring lenses
                                  ├── regulation analysis
                                  └── model/provider adapters
```

The Python service is optional. If it is unavailable, times out, returns a non-success status, or produces a malformed response, the Go application records an explicit advisory error and continues without changing primary workflow state.

## Configuration

The integration is disabled by default. Credentials must be supplied through the deployment environment or a secret manager rather than committed to source control.

| Variable | Required | Meaning |
|---|---:|---|
| `INTEGIN_AI_ENABLED` | No | Boolean switch; defaults to `false`. |
| `INTEGIN_AI_ENDPOINT` | When enabled | Base URL for the Python service, for example `http://ai-service:8000`. |
| `INTEGIN_AI_API_KEY` | Optional | Bearer credential supplied to the service. |
| `INTEGIN_AI_TIMEOUT` | No | Request timeout in seconds; defaults to five seconds. |

The Go configuration loader rejects enabled integration without an endpoint, invalid boolean values, and non-positive timeouts. Default unit tests never require these variables or a running AI service.

## Contract and Security Rules

Requests use contract version `v1` and include tenant ID, advisory zone, monitoring lens, input keys/data, and evidence references. The client rejects all AI-free zones before making an HTTP request. Responses are validated for contract version and tenant identity, and are normalized through `internal/advisory`, which forces `blocking=false` and requires rationale and a confidence value between zero and one.

The AI service must not receive secrets, credentials, hidden prompts, or primary-decision authority. Secondary advisor views expose only approved advisory fields, evidence references, limitations, and model metadata. Public QR responses do not use the AI integration.

## Rollout and Rollback

Deploy a candidate AI-service version alongside a known-good baseline. Mark the candidate healthy only after the `/healthz` check succeeds and mock or opt-in integration tests pass. Shift traffic gradually using the canary metadata contract. Keep rollback readiness true while candidate traffic is non-zero. If health degrades, return traffic to the baseline and disable `INTEGIN_AI_ENABLED`; the Go application continues operating deterministically without advisory enrichment.

## Validation

From the project root:

```bash
gofmt -l $(find . -name '*.go' -type f)
go test ./...
go vet ./...
```

The default suite uses local mock HTTP servers and does not require PostgreSQL, MinIO, SMTP, or the Python AI service. Live integration testing must be explicitly enabled in a deployment environment and must never be used to gate primary business operations.
