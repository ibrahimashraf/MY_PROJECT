# INTEGIN Matrix Fixture Provisioning Runtime Status

**Date:** 2026-08-20

## Status

The Matrix Fixture Provisioning runtime gate is **not proven complete**. The protected fixture infrastructure exists, and the local pilot environment was prepared, but the Flutter acceptance runner exited with code `1` without producing test output. The live provisioning, signed sync, duplicate-safe evidence, and runtime receipt sequence therefore remains unproven.

## What was verified

The protected Windows-debug fixture directory existed and was used only outside the source repository. A temporary Ed25519 fixture descriptor and private key were generated, bound through the existing key-sync command, and used to generate local pilot SQL. The fixture SQL was applied only to the loopback pilot PostgreSQL target. The local pilot server was built from the active source and reached HTTP health and readiness `200` on `127.0.0.1:18080` with `/local/provision`, `/sync`, and `/evidence` configured for the isolated test.

The server was configured with the dedicated pilot database and RustFS settings from the protected environment file. The pilot data inserted by the fixture generator was removed after the unsuccessful Flutter run. The local pilot server was stopped. Temporary fixture descriptors, private key material, generated SQL, authority secret, logs, and runtime files were removed from the private fixture directory.

## What was not proven

The Flutter test `test/live_provisioning_acceptance_test.dart` was invoked with the three loopback endpoint definitions. The command exited with code `1` and produced an empty captured log. A direct Windows command invocation of `flutter.bat --version` also produced no usable output. Consequently, there is no runtime receipt proving that Flutter provisioned a session, applied signed sync, uploaded evidence, or received `DUPLICATE` on the second evidence submission.

This result must not be confused with the authorized eight-case manifest receipt matrix. That matrix remains **passed** for field binding, valid proof, replay rejection, invalid signature, invalid package hash, expiry, authority mismatch, and unknown key. The fixture runtime gate is a separate evidence requirement.

## Safety outcome

No production database, production endpoint, certificate authority, signing authority, external authority, or manifest acceptance result was changed. No fixture private key or authority secret was committed to source control. The temporary local pilot data was cleaned up and verified absent.

## Next action

Repair or replace the unavailable Flutter runner/toolchain on the connected Windows workstation, then repeat the isolated endpoint-backed test with fresh temporary fixture material. Do not mark this runtime gate passed until provisioning, signed sync, duplicate-safe evidence, receipts, and cleanup are all captured.
