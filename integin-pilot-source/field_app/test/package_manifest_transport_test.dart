import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/workpackages/package_manifest_transport.dart';

void main() {
  test('transport signs deterministic manifest-read proof JSON', () async {
    http.Request? captured;
    final client = MockClient((request) async {
      captured = request;
      return http.Response('{}', 200);
    });
    final signer = DeviceSigner(
      await Ed25519().newKeyPair(),
      keyId: 'device-key-1',
    );
    final transport = PackageManifestTransport(
      client: client,
      requestIdGenerator: () => 'request-1',
      // Pilot loopback fixture requires the explicit opt-in.
      allowLoopbackHttp: true,
    );

    await transport.fetch(
      endpoint: Uri.parse('http://127.0.0.1:18080/work-package-manifest'),
      signer: signer,
      deviceId: 'device-1',
      authorityId: 'authority-1',
      authorityEpoch: 7,
      inspectionId: 'inspection-1',
      now: DateTime.utc(2026, 8, 17, 12),
    );

    final request = captured!;
    expect(request.headers['content-type'], 'application/json');
    final body = jsonDecode(request.body) as Map<String, dynamic>;
    final proof = body['proof'] as Map<String, dynamic>;
    expect(proof['purpose'], manifestReadProofPurpose);
    expect(proof['request_id'], 'request-1');
    expect(proof['key_id'], 'device-key-1');
    expect(proof['issued_at'], '2026-08-17T12:00:00Z');
    expect(proof['expires_at'], '2026-08-17T12:05:00Z');
    expect(proof['signature_algorithm'], manifestReadProofSignatureAlgorithm);
  });

  test('transport fails before request when signer key ID is absent', () async {
    final client = MockClient((_) async => http.Response('{}', 200));
    final signer = DeviceSigner(await Ed25519().newKeyPair());
    final transport = PackageManifestTransport(
      client: client,
      allowLoopbackHttp: true,
    );

    expect(
      () => transport.fetch(
        endpoint: Uri.parse('http://127.0.0.1:18080/work-package-manifest'),
        signer: signer,
        deviceId: 'device-1',
        authorityId: 'authority-1',
        authorityEpoch: 7,
        inspectionId: 'inspection-1',
        now: DateTime.utc(2026, 8, 17, 12),
      ),
      throwsA(
        isA<PackageManifestTransportException>().having(
          (error) => error.failure,
          'failure',
          PackageManifestTransportFailure.configuration,
        ),
      ),
    );
  });
}
