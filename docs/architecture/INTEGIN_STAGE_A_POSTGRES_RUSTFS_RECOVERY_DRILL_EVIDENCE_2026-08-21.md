# INTEGIN Stage A PostgreSQL and RustFS Recovery Drill Evidence

## Result

The current-schema controlled recovery drill **passed**. The source was the applied pilot PostgreSQL and RustFS environment. The destination was a fresh PostgreSQL 18 container bound only to `127.0.0.1:25432` and a fresh RustFS `1.0.0-rc.1` container bound only to `127.0.0.1:29100`; both used new disposable volumes and were removed after verification.

The retained evidence directory is `C:\MY PROJECT\operations\acceptance\backups\stage-a-recovery-b6c1804e50ac`. It contains the generated-fixture manifest, object backup, current-schema PostgreSQL custom-format dump, and a SHA-256 checksum file for the dump. No secret, bearer token, database credential, or S3 credential is recorded in that directory or this record.

| Verification | Result |
|---|---|
| Current schema backup | Fresh custom-format backup captured after an `it-recovery-*` Work-Order, scope, assignment, inspection, and state-event graph was created. |
| Relational restoration | Restored database contained exactly one row for each generated Work-Order, scope, assignment, canonical inspection, and audit state event. |
| Tenant isolation after restoration | A non-superuser `NOBYPASSRLS` role saw the generated inspection with matching tenant/organization context and zero rows after the organization context changed. |
| Object restoration | A generated ciphertext object was backed up, restored through the S3 abstraction, and verified for key, content type, ciphertext SHA-256, plaintext SHA-256 metadata, tenant, organization, and inspection metadata. |
| Source cleanup | Generated pilot relational fixture and source evidence object were deleted; post-run count was zero. |
| Target cleanup | Disposable restore containers and volumes were absent after the run. |

## Storage metadata contract repair

The drill identified that `S3Store.Get` returned an empty metadata map even though `Put` writes `X-Amz-Meta-*` values. The adapter now normalizes and returns S3 response metadata, with focused unit coverage. This correction enabled the restored object proof to verify metadata rather than only ciphertext bytes.

## Controlled repair history

The first drill attempt used an incorrect in-container PostgreSQL backup connection and stopped before creating a restore target. A second cleanup path exposed a recovery-harness SQL argument-binding defect. Both failures were repaired; the retained source fixture was cleaned through the manifest-scoped command; and the entire drill was then repeated successfully. The dump checksum file was generated immediately after the successful run from the retained dump because the initial shell-side checksum write did not persist. No failed-attempt source fixture, target container, or target volume remained.

## Limits

This is a current-schema generated-fixture recovery proof. It does not establish a production disaster-recovery service-level objective, a raw full-volume RustFS restoration claim, restore of customer data, or a post-restore Field application session. The evidence handler currently stores object metadata at the S3 boundary; it does not independently persist an evidence/inspection/audit relational link, so this drill records relational/audit restoration and object-plus-metadata restoration as distinct proofs.
