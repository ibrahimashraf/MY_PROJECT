# INTEGIN Identity Resolver Compatibility Evidence

The pilot identity function now returns actor ID, tenant ID, organization ID, Work-Order role, and capabilities. The Go `identity.Membership` contract and `PostgresResolver` query/scan order were updated to match this signature.

The focused identity and OIDC session tests passed, followed by a full `go test ./...` pass. No HTTP Work-Order mutation handler has been added. This compatibility fix eliminates the previously identified runtime scan mismatch but does not by itself prove a live OIDC-to-Work-Order request path.
