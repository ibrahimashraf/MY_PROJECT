# Certificate Public-Binding Implementation Proof — 2026-08-22

## Implemented scope

Migration `0014_certificate_public_bindings.candidate.sql` is applied to the pilot database after a fresh pre-apply custom-format backup (`integin-pilot-pre-0014-20260822-094737.dump`). The migration establishes forced-RLS canonical asset registry, revision-bound public inspection scope, ordered public-scope items, policy binding allow-list, and `certificate_snapshot.public_binding_snapshot`.

Certificate issuance now locks and materializes policy-approved public facts within the existing RLS-scoped issuance transaction. It snapshots admitted values, rejects an unavailable required asset or inspection binding, and includes the public-binding snapshot in the issuance snapshot digest. The existing public verifier response is deliberately unchanged and remains limited to its prior five fields.

## Controlled review and proof matrix

| Control | Evidence | Result |
|---|---|---|
| Migration up/down safety | Disposable PostgreSQL copy restored from the pilot baseline; up candidate created four relations with RLS enabled and forced; down candidate removed four relations and snapshot column | Passed; disposable container and dump removed |
| Pilot schema migration | Fresh backup then 0014 applied to pilot | Passed |
| Canonical source selection | Integration fixture created active canonical asset, exact revision public scope, ordered scope item, and required policy bindings | Passed |
| Issuance snapshot | Issued certificate persisted serial number, description, type, inspection type, and public scope into `public_binding_snapshot` | Passed |
| Mutable-source resistance | Fixture mutated the live asset registry after issuance; saved snapshot retained `SERIAL-A` and did not contain `MUTATED-SERIAL` | Passed |
| Existing certificate lifecycle | Independent review, senior self-issue, revocation, replacement/supersession, expiry, digest-backed token, and narrow verifier tests remain in the same integration test | Passed |
| Tenant/org containment and cleanup | Existing cross-organization draft denial ran; post-run query across certificate, asset registry, public scope/items, and policy bindings returned `0|0|0|0|0` for namespaced fixtures | Passed |
| Regression | `go test -count=1 ./...` and `go vet ./...` | Passed |

## Honest release boundary

This slice makes the requested serial, description, type, and scoped-inspection facts **available as immutable, policy-approved certificate snapshot data**. It does not yet put them into the public verifier HTTP response, a QR code, or a rendered certificate PDF. Those next steps require a separate allow-list transport extension and renderer/QR implementation, with shared deployment-grade rate and abuse controls before external availability.

No external authority connection, external recognition, public deployment, document download endpoint, or shared rate-limiting system was added.
