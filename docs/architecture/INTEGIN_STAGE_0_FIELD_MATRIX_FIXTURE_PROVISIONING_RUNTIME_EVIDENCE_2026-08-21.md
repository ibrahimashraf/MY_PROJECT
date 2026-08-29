# INTEGIN Stage 0 Field Matrix Fixture Provisioning Runtime Evidence

## Result

The controlled Field runtime proof **passed** against the existing loopback-only acceptance server. The Field acceptance test used the server's local provisioning, sync, and evidence routes. It provisioned a unique device, applied the signed inspection synchronization transaction, stored the evidence object, and received the expected duplicate-safe response when it repeated the same evidence submission.

| Assertion | Observed result |
|---|---|
| Device provisioning | Accepted by the loopback-only local provisioning endpoint. |
| Signed sync transaction | Applied by the sync endpoint. |
| First evidence upload | `APPLIED`. |
| Identical evidence replay | `DUPLICATE`. |
| Test result | Flutter test passed. |
| Receipt-driven cleanup | Passed. |

## Safety and cleanup

The test now writes a receipt containing only generated fixture identifiers **before** provisioning. The cleanup harness sources the existing acceptance configuration only into its process, sets the matching tenant and organization RLS context, deletes only receipt-named sync-state and authority records, deletes only the generated evidence object, verifies their absence, and removes the receipt. It does not expose configuration values or delete unscoped acceptance data.

The first cleanup-harness run exposed a parameter-binding defect in its verification query after the Field test itself had passed. The fixture receipt was retained, the defect was repaired, and cleanup completed. The complete Field test and cleanup were then repeated successfully in one run. This document relies on that second complete run.

## Limits

This is a controlled acceptance proof using the local loopback service and its configured local data stores. It does not prove production hosting, live external identity-provider availability, or a real external Keycloak tenant.
