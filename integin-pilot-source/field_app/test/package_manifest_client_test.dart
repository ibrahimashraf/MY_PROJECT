// INTEGIN Field manifest verifier tests: invalid signed payloads never bind to cache data.
import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/workpackages/package_manifest_client.dart';

void main() {
  test('rejects invalid manifest signature before package mapping', () async {
    final keyPair = await Ed25519().newKeyPair();
    final verifier = PackageManifestVerifier(
      authorityPublicKey: await keyPair.extractPublicKey(),
    );
    final now = DateTime.utc(2026, 8, 17, 10);
    final manifest = SignedPackageManifest({
      'manifest_version': 'work-package-manifest/v1',
      'tenant_id': 'tenant-1',
      'organization_id': 'organization-1',
      'inspection_id': 'inspection-1',
      'device_id': 'device-1',
      'package_id': 'package-1',
      'package_version': 1,
      'package_hash': 'sha256:${'a' * 64}',
      'package': <String, dynamic>{},
      'assignment_context': {
        'root_asset_id': 'asset-1',
        'inspection_type': 'thorough-inspection',
        'procedure_version': 'v1',
        'scheduled_at': now.toIso8601String(),
        'field_asset_ids': <String, String>{},
      },
      'schema_version': 1,
      'authority_epoch': 7,
      'issued_at': now.toIso8601String(),
      'expires_at': now.add(const Duration(hours: 1)).toIso8601String(),
      'signature_algorithm': 'Ed25519',
      'key_id': 'key-1',
      'signature': base64Encode(List<int>.filled(64, 0)),
    });

    await expectLater(
      verifier.verifyAndBind(manifest, now: now),
      throwsA(isA<StateError>()),
    );
  });
}
