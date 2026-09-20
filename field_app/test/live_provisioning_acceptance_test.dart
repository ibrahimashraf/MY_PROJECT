import 'dart:convert';
import 'dart:io';

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

const _provisionEndpoint =
    String.fromEnvironment('INTEGIN_LIVE_PROVISIONING_ENDPOINT', defaultValue: String.fromEnvironment('INTEGIN_LIVE_PROVISIONING_ENDPOINT'));
const _syncEndpoint = String.fromEnvironment('INTEGIN_LIVE_SYNC_ENDPOINT', defaultValue: String.fromEnvironment('INTEGIN_LIVE_SYNC_ENDPOINT'));
const _evidenceEndpoint =
    String.fromEnvironment('INTEGIN_LIVE_EVIDENCE_ENDPOINT', defaultValue: String.fromEnvironment('INTEGIN_LIVE_EVIDENCE_ENDPOINT'));
const _receiptPath = String.fromEnvironment('INTEGIN_LIVE_RECEIPT_PATH', defaultValue: String.fromEnvironment('INTEGIN_LIVE_RECEIPT_PATH'));
const _tenantID = String.fromEnvironment('INTEGIN_LIVE_TENANT_ID', defaultValue: String.fromEnvironment('INTEGIN_LIVE_TENANT_ID'));
const _organizationID = String.fromEnvironment('INTEGIN_LIVE_ORGANIZATION_ID', defaultValue: String.fromEnvironment('INTEGIN_LIVE_ORGANIZATION_ID'));
// Bearer token for the OIDC-gated evidence route (D-02). Empty outside a
// provisioned drill harness; the test skips instead of failing unauthenticated.
const _idToken = String.fromEnvironment('INTEGIN_LIVE_ID_TOKEN', defaultValue: String.fromEnvironment('INTEGIN_LIVE_ID_TOKEN'));

InspectionWorkPack _workPack(String inspectionID) => InspectionWorkPack(
      inspectionId: inspectionID,
      rootAssetId: 'live-asset',
      inspectionType: 'live-acceptance',
      procedureVersion: 'integin-local-provision-v1',
      packageId: 'live-work-package',
      packageVersion: 1,
      schemaVersion: 1,
      packageHash: 'sha256:live-work-package-hash',
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
      _evidenceEndpoint.isNotEmpty &&
      _receiptPath.isNotEmpty &&
      _tenantID.isNotEmpty &&
      _organizationID.isNotEmpty &&
      _idToken.isNotEmpty;

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
      final plannedDeviceID = 'field-${signer.keyId!.substring(0, 32)}';
      final workPack = _workPack('flutter-live-$plannedDeviceID');
      final evidenceID = 'flutter-live-evidence-$plannedDeviceID';
      final receiptFile = File(_receiptPath);
      await receiptFile.parent.create(recursive: true);
      await receiptFile.writeAsString(jsonEncode({
        'tenant_id': _tenantID,
        'organization_id': _organizationID,
        'device_id': plannedDeviceID,
        'authority_id': '',
        'inspection_id': workPack.inspectionId,
        'evidence_id': evidenceID,
      }));
      final session = await LocalProvisioningClient(
        endpoint: Uri.parse(_provisionEndpoint),
        allowLoopbackHttp: true, // live acceptance test uses loopback server
      ).provision(signer);
      expect(session.deviceId, plannedDeviceID);
      expect(session.context.tenantId, _tenantID);
      expect(session.context.organizationId, _organizationID);
      await receiptFile.writeAsString(jsonEncode({
        'tenant_id': _tenantID,
        'organization_id': _organizationID,
        'device_id': session.deviceId,
        'authority_id': session.authority.id,
        'inspection_id': workPack.inspectionId,
        'evidence_id': evidenceID,
      }));
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
          transport: HttpSyncTransport(
            endpoint: Uri.parse(_syncEndpoint),
            allowLoopbackHttp: true, // live acceptance test uses loopback server
          ),
        ),
      );
      controller.beginInspection(workPack);
      controller.recordResponse(
        item: workPack.items.single,
        response: 'acceptable',
      );
      expect(
          await controller.queueForSync(
              notes: 'provisioned Flutter acceptance'),
          isTrue);
      expect(await controller.flushOutbox(), [SyncOutcome.applied]);

      final ciphertext = utf8.encode('Flutter provisioned evidence ciphertext');
      final plaintext = utf8.encode('Flutter provisioned evidence plaintext');
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
        headers: {
          'Content-Type': 'application/json',
          // Distinct keys per attempt: the route mandates Idempotency-Key,
          // and the test exercises server-side evidence_id dedupe
          // (APPLIED then DUPLICATE), not middleware replay.
          'Idempotency-Key': '$evidenceID-attempt-1',
          'Authorization': 'Bearer $_idToken',
        },
        body: jsonEncode(body),
      );
      expect(first.statusCode, 200);
      expect(jsonDecode(first.body)['outcome'], 'APPLIED');
      final duplicate = await http.post(
        Uri.parse(_evidenceEndpoint),
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': '$evidenceID-attempt-2',
          'Authorization': 'Bearer $_idToken',
        },
        body: jsonEncode(body),
      );
      expect(duplicate.statusCode, 200);
      expect(jsonDecode(duplicate.body)['outcome'], 'DUPLICATE');
    },
    skip: shouldRun
        ? false
        : 'Set all INTEGIN_LIVE_* endpoints, tenant context, INTEGIN_LIVE_RECEIPT_PATH, and INTEGIN_LIVE_ID_TOKEN to run live acceptance.',
  );
}
