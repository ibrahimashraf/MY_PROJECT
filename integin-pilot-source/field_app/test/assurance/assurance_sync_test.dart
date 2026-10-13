import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:integin_field_app/assurance/assurance_sync_service.dart';
import 'package:integin_field_app/assurance/canonical_sync_payload.dart';
import 'package:integin_field_app/security/transaction_signer.dart';

void main() {
  group('CanonicalSyncPayload & Ed25519 Signing Parity', () {
    test('serializes canonical JSON matching Go backend expectations', () {
      final payload = CanonicalSyncPayload(
        tenantId: '018f45a0-0000-7000-8000-000000000001',
        assetId: '018f45a0-0000-7000-8000-000000000002',
        state: 'CONDEMNED',
        lamportClock: 42,
        revision: 1,
        effectiveAtSeconds: 1774645200,
        reasonsBytes: Uint8List.fromList(utf8.encode('{"defect":"severe_reduction"}')),
        evidenceChainBytes: Uint8List.fromList(utf8.encode('[]')),
      );

      final jsonMap = payload.toJson();

      expect(jsonMap['tenant_id'], '018f45a0-0000-7000-8000-000000000001');
      expect(jsonMap['asset_id'], '018f45a0-0000-7000-8000-000000000002');
      expect(jsonMap['state'], 'CONDEMNED');
      expect(jsonMap['lamport_clock'], 42);
      expect(jsonMap['revision'], 1);
      expect(jsonMap['effective_at'], 1774645200);
      expect(jsonMap['reasons'], base64Encode(utf8.encode('{"defect":"severe_reduction"}')));
      expect(jsonMap['evidence_chain'], base64Encode(utf8.encode('[]')));
    });

    test('DeviceSigner signs canonical bytes verifiable by standard Ed25519.verify', () async {
      final algorithm = Ed25519();
      final keyPair = await algorithm.newKeyPair();
      final signer = DeviceSigner(keyPair);

      final payload = CanonicalSyncPayload(
        tenantId: '018f45a0-0000-7000-8000-000000000001',
        assetId: '018f45a0-0000-7000-8000-000000000002',
        state: 'NON_COMPLIANT',
        lamportClock: 10,
        revision: 2,
        effectiveAtSeconds: 1774645000,
        reasonsBytes: Uint8List.fromList(utf8.encode('["broken_wires_cluster"]')),
        evidenceChainBytes: Uint8List.fromList(utf8.encode('[]')),
      );

      final canonicalBytes = payload.toCanonicalBytes();
      final signatureBytes = await signer.signRawBytes(canonicalBytes);

      final publicKey = await keyPair.extractPublicKey();

      // Verify cryptographic signature matches canonical bytes
      final verified = await algorithm.verify(
        canonicalBytes,
        signature: Signature(signatureBytes, publicKey: publicKey),
      );

      expect(verified, isTrue);
    });

    test('AssuranceSyncService submits payload and handles backend accepted response', () async {
      final keyPair = await Ed25519().newKeyPair();
      final signer = DeviceSigner(keyPair);

      late Map<String, dynamic> receivedServerPayload;

      final mockClient = MockClient((request) async {
        if (request.url.path == '/api/v1/assurance/sync') {
          receivedServerPayload = jsonDecode(request.body) as Map<String, dynamic>;
          return http.Response(jsonEncode({'status': 'ACCEPTED'}), 200);
        }
        return http.Response('Not Found', 404);
      });

      final service = AssuranceSyncService(
        endpoint: Uri.parse('http://127.0.0.1:8080'),
        client: mockClient,
      );

      final result = await service.submitSync(
        tenantId: '018f45a0-0000-7000-8000-000000000001',
        assetId: '018f45a0-0000-7000-8000-000000000002',
        signerId: 'inspector-42',
        state: 'CONDEMNED',
        lamportClock: 55,
        revision: 3,
        effectiveAt: DateTime.utc(2026, 9, 27, 21, 0, 0),
        reasonsBytes: Uint8List.fromList(utf8.encode('["critical_reduction"]')),
        evidenceChainBytes: Uint8List.fromList(const []),
        signer: signer,
        isOfflineOrigin: true,
      );

      expect(result.isSuccess, isTrue);
      expect(result.status, AssuranceSyncStatus.accepted);
      expect(receivedServerPayload['tenant_id'], '018f45a0-0000-7000-8000-000000000001');
      expect(receivedServerPayload['state'], 'CONDEMNED');
      expect(receivedServerPayload['is_offline_origin'], isTrue);
      expect(receivedServerPayload['signature'], isNotEmpty);
    });
  });
}
