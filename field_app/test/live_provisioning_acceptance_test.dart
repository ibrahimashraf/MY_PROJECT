import 'dart:convert';

import 'package:crypto/crypto.dart' as crypto;
import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:integin_field_app/application/field_app_controller.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/provisioning/local_provisioning_client.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/sync/http_sync_transport.dart';
import 'package:integin_field_app/sync/sync_client.dart';

const _provisionEndpoint = String.fromEnvironment('INTEGIN_LIVE_PROVISIONING_ENDPOINT');
const _syncEndpoint = String.fromEnvironment('INTEGIN_LIVE_SYNC_ENDPOINT');
const _evidenceEndpoint = String.fromEnvironment('INTEGIN_LIVE_EVIDENCE_ENDPOINT');

InspectionWorkPack _workPack(String inspectionID) => InspectionWorkPack(
      inspectionId: inspectionID,
      rootAssetId: 'live-asset',
      inspectionType: 'live-acceptance',
      procedureVersion: 'integin-local-provision-v1',
      scheduledDate: DateTime.now().toUtc(),
      items: const [
        ChecklistItem(
          id: 'live-item',
          sectionId: 'live-section',
          prompt: 'Live acceptance inspection response',
          assetId: 'live-asset',
        ),
      ],
    );

void main() {
  final shouldRun = _provisionEndpoint.isNotEmpty &&
      _syncEndpoint.isNotEmpty &&
      _evidenceEndpoint.isNotEmpty;

  test(
    'provisioned Flutter client applies signed sync and duplicate-safe evidence',
    () async {
      final algorithm = Ed25519();
      final keyPair = await algorithm.newKeyPair();
      final publicKey = await keyPair.extractPublicKey();
      final signer = DeviceSigner(
        keyPair,
        keyId: crypto.sha256.convert(publicKey.bytes).toString(),
      );
      final session = await LocalProvisioningClient(
        endpoint: Uri.parse(_provisionEndpoint),
      ).provision(signer);
      final store = InMemoryOutboxStore();
      final controller = FieldAppController(
        context: session.context,
        deviceId: session.deviceId,
        userId: session.userId,
        deviceState: DeviceTrustState.trusted,
        authority: session.authority,
        outboxStore: store,
        deviceSigner: signer,
        deviceKeyId: signer.keyId,
        syncClient: SyncClient(
          store: store,
          transport: HttpSyncTransport(endpoint: Uri.parse(_syncEndpoint)),
        ),
      );
      final workPack = _workPack('flutter-live-${session.deviceId}');
      controller.beginInspection(workPack);
      controller.recordResponse(
        item: workPack.items.single,
        response: 'acceptable',
      );
      expect(await controller.queueForSync(notes: 'provisioned Flutter acceptance'), isTrue);
      expect(await controller.flushOutbox(), [SyncOutcome.applied]);

      final ciphertext = utf8.encode('Flutter provisioned evidence ciphertext');
      final plaintext = utf8.encode('Flutter provisioned evidence plaintext');
      final evidenceID = 'flutter-live-evidence-${session.deviceId}';
      final body = {
        'tenant_id': session.context.tenantId,
        'organization_id': session.context.organizationId,
        'evidence_id': evidenceID,
        'inspection_id': workPack.inspectionId,
        'content_type': 'application/octet-stream',
        'plaintext_sha256': crypto.sha256.convert(plaintext).toString(),
        'ciphertext_sha256': crypto.sha256.convert(ciphertext).toString(),
        'base64_blob': base64Encode(ciphertext),
      };
      final first = await http.post(
        Uri.parse(_evidenceEndpoint),
        headers: const {'Content-Type': 'application/json'},
        body: jsonEncode(body),
      );
      expect(first.statusCode, 200);
      expect(jsonDecode(first.body)['outcome'], 'APPLIED');
      final duplicate = await http.post(
        Uri.parse(_evidenceEndpoint),
        headers: const {'Content-Type': 'application/json'},
        body: jsonEncode(body),
      );
      expect(duplicate.statusCode, 200);
      expect(jsonDecode(duplicate.body)['outcome'], 'DUPLICATE');
    },
    skip: shouldRun ? false : 'Set all INTEGIN_LIVE_* endpoints to run live acceptance.',
  );
}
