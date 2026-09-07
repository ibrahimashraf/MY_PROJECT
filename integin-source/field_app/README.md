# INTEGIN Field App

This directory contains the first Flutter/Flutter Web implementation slice for INTEGIN field work. It is an offline-first capture client, not an authoritative inspection or certification service.

## Local validation

From a machine with Flutter 3.47.0 or newer installed:

```bash
flutter pub get
flutter analyze
flutter test
flutter build web --release
```

The current source has been validated in the sandbox with no analyzer issues, nineteen passing tests, and a successful Web release build. The connected Windows project does not currently expose Flutter on PATH; install Flutter locally or add its `bin` directory before running these commands there.

## Current capabilities

The app models tenant and organization context, trusted-device authority, prepared work packs, local inspection findings, AES-GCM encrypted evidence blobs, separate plaintext/ciphertext digests, versioned v1 mutation envelopes, Ed25519 device signatures, append-only outbox state, local completeness validation, persisted JSON outbox storage through `shared_preferences`, and explicit sync outcomes. The initial UI makes authority expiry, connectivity, queue count, local capture, and the server-controlled boundary visible.

## Security and authority boundary

The field client may capture attributable work while offline, but the server remains authoritative. The app does not approve inspections, issue certificates, validate calibration, alter authorization, or expand public QR projections. Browser persistence is durable through `shared_preferences`; native Flutter platforms select a `SecureKeyValueStore` backed by `flutter_secure_storage`, and `SecureDeviceKeyStore` provisions a stable Ed25519 identity. The HTTP transport posts the v1 envelope and encrypted evidence with separate plaintext/ciphertext digests. The server route remains authoritative; the legacy development HMAC path exists only for migration tests and must not be used for production enrollment.

## Next implementation slice

While PostgreSQL and RustFS installation is pending, the next portable slice is to strengthen the field-app and protocol contract without claiming live infrastructure: complete the native durable outbox/recovery behavior, add cross-language compatibility vectors for all sync outcomes and evidence digests, document local authority/device recovery, and add dependency-free failure/retry tests. After the required services are installed, run the production-style end-to-end matrix against PostgreSQL and RustFS.
