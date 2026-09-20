# INTEGIN Certificate Runtime Composition Decision

## Route activation

The public verifier handler is safe to mount only when a database-backed certificate repository exists. It is public and does not require OIDC, but it stays absent when `INTEGIN_DB_URL` is absent. The handler itself supplies no-store behavior and a process-local limiter; deployment-grade shared rate limiting remains a runtime prerequisite for public release.

Certificate mutation routes are mounted only inside the existing enabled OIDC branch, after the server has successfully built the configured validator and local PostgreSQL identity resolver. The handler receives those dependencies directly and never falls back to request-derived tenant, organization, actor, capability, or policy values. If OIDC is disabled, mutation routes remain absent rather than becoming anonymously accessible.

## Mux contract

`server.Dependencies` receives optional authenticated and public certificate handlers. `NewMux` mounts exact prefixes only when each handler is non-nil: `/certificates/` for authenticated lifecycle operations and `/verify/certificates/` for public lookup. Existing sync, readiness, manifest, evidence, work-order, and local-provisioning routes remain unchanged.

## Runtime acceptance

The composition test must prove absent optional handlers return the existing not-found response; mounted public handler reaches only the verifier prefix; mounted authenticated handler reaches only the certificate prefix; OIDC-disabled composition does not mount mutations; and the existing protected handler wiring continues to pass. The runtime proof must separately cover valid public lookup, invalid/missing token response, public limiter rejection, unauthenticated mutation rejection, and server-derived membership authority.
