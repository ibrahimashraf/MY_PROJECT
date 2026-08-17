import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/security/transaction_signer.dart';

void main() {
  test('canonical manifest proof matches the Go device-proof grammar', () {
    final canonical = canonicalManifestProof(
      requestId: 'request-1',
      deviceId: 'device-1',
      authorityId: 'authority-1',
      authorityEpoch: 7,
      inspectionId: 'inspection-1',
      issuedAt: DateTime.utc(2026, 8, 17, 12),
      expiresAt: DateTime.utc(2026, 8, 17, 12, 5),
      keyId: 'device-key-1',
    );
    expect(
      canonical,
      'device-proof/v1|work_package_manifest.read|request-1|device-1|'
      'authority-1|7|inspection-1|2026-08-17T12:00:00Z|'
      '2026-08-17T12:05:00Z|Ed25519|device-key-1',
    );
  });
}
