# INTEGIN Field App

Offline-first Flutter capture client for INTEGIN field work. Not an authoritative inspection or certification service.

## Platforms

| Platform | Status |
|----------|--------|
| Android  | Validated (APK build) |
| Web      | Validated (WASM + drift_worker.js) |
| Windows  | Deferred (disk space) |
| iOS/macOS| Deferred (no Apple devices) |

## Local validation

From a machine with Flutter 3.47.5 or newer installed:

```bash
flutter pub get
flutter analyze
flutter test
flutter build web --release
```

206 tests pass, zero analyzer issues.

## Storage architecture

The app uses **drift** (type-safe SQLite ORM) with **SQLite3MultipleCiphers** for AES-256 encryption at rest.

| Layer | Native | Web |
|-------|--------|-----|
| ORM | drift | drift |
| Engine | sqlite3 + sqlite3_flutter_libs | drift WASM (sqlite3.wasm + drift_worker.js) |
| Encryption | SQLite3MC `PRAGMA key` in drift `setup:` callback | OS-level (browser OPFS/IndexedDB encryption) |
| Key storage | `flutter_secure_storage` | `flutter_secure_storage` |

Encryption key: auto-generated 32-byte hex on first launch, stored at `INTEGIN.database.encryption_key`.

### Files

- `lib/storage/tables.dart` — Drift table definitions
- `lib/storage/app_database.dart` — Drift DB class with conditional import (native vs web)
- `lib/storage/platform_database_native.dart` — Native opener (SQLite3MC + PRAGMA key)
- `lib/storage/platform_database_web.dart` — Web WASM opener
- `lib/storage/drift_outbox_store.dart` — OutboxStore drift implementation
- `web/sqlite3.wasm` — SQLite WASM binary
- `web/drift_worker.js` — Drift web worker

## Current capabilities

Tenant and organization context, trusted-device authority, prepared work packs, local inspection findings, AES-GCM encrypted evidence blobs, separate plaintext/ciphertext digests, versioned v1 mutation envelopes, Ed25519 device signatures, append-only outbox state with drift persistence, local completeness validation, and explicit sync outcomes. The UI exposes authority expiry, connectivity, queue count, local capture, and the server-controlled boundary.

## Security and authority boundary

The field client captures attributable work offline; the server remains authoritative. The app does not approve inspections, issue certificates, validate calibration, alter authorization, or expand public QR projections. Native platforms use `flutter_secure_storage` for the encryption key and `SecureDeviceKeyStore` for a stable Ed25519 identity. The HTTP transport posts the v1 envelope and encrypted evidence with separate plaintext/ciphertext digests.

## Next implementation slice

Complete the native durable outbox/recovery behavior, add cross-language compatibility vectors for all sync outcomes and evidence digests, document local authority/device recovery, and add dependency-free failure/retry tests. After required services are installed, run the production-style end-to-end matrix against PostgreSQL and RustFS.
