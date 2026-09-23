import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/application/field_app_controller.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/merkle_hash_chain.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/sync/sync_guard.dart';

const _context = TenantContext(
    tenantId: 'tenant-1', organizationId: 'org-1', environment: 'LIVE');

OfflineAuthority _authority({
  required String id,
  required String deviceId,
  required int epoch,
}) {
  final issued = DateTime.now().toUtc().subtract(const Duration(minutes: 1));
  return OfflineAuthority(
    id: id,
    deviceId: deviceId,
    context: _context,
    userId: 'user-1',
    epoch: epoch,
    scopes: const ['inspection.perform'],
    capabilities: const ['inspection.perform'],
    procedureVersion: 'test',
    issuedAt: issued,
    expiresAt: issued.add(const Duration(hours: 1)),
    signature: 'test-signature',
  );
}

Future<OfflineMutation> _signedMutation({
  required DeviceSigner signer,
  required String transactionId,
  required String deviceId,
  required int sequenceNumber,
  required String authorityId,
  required int authorityEpoch,
}) async {
  final payload = <String, Object?>{
    'inspection_id': 'inspection-1',
    'response': 'acceptable',
  };
  final unsigned = OfflineMutation(
    transactionId: transactionId,
    context: _context,
    deviceId: deviceId,
    userId: 'user-1',
    sequenceNumber: sequenceNumber,
    operation: 'InspectionSubmitted',
    entityId: 'inspection-1',
    payload: payload,
    capturedAt: DateTime.now().toUtc(),
    authorityId: authorityId,
    authorityEpoch: authorityEpoch,
    signatureAlgorithm: 'Ed25519',
    keyId: 'device-key-1',
    signature: '',
  );
  final signature = await signer.signV1(
    transactionId: transactionId,
    tenantId: _context.tenantId,
    organizationId: _context.organizationId,
    environment: _context.environment,
    deviceId: deviceId,
    userId: 'user-1',
    sequenceNumber: sequenceNumber,
    operation: 'InspectionSubmitted',
    entityId: 'inspection-1',
    payloadHash: unsigned.payloadHash,
    authorityId: authorityId,
    authorityEpoch: authorityEpoch,
    capturedAt: unsigned.capturedAt,
    keyId: 'device-key-1',
  );
  return OfflineMutation(
    transactionId: transactionId,
    context: _context,
    deviceId: deviceId,
    userId: 'user-1',
    sequenceNumber: sequenceNumber,
    operation: 'InspectionSubmitted',
    entityId: 'inspection-1',
    payload: payload,
    capturedAt: unsigned.capturedAt,
    authorityId: authorityId,
    authorityEpoch: authorityEpoch,
    signatureAlgorithm: 'Ed25519',
    keyId: 'device-key-1',
    signature: signature,
  );
}

void main() {
  test(
      'authority rotation migrates stale and failed entries instead of '
      'orphaning inspector work', () async {
    final oldSigner =
        DeviceSigner(await Ed25519().newKeyPair(), keyId: 'device-key-1');
    final newSigner =
        DeviceSigner(await Ed25519().newKeyPair(), keyId: 'device-key-1');
    final store = InMemoryOutboxStore();

    final staleQueued = OutboxEntry(
      mutation: await _signedMutation(
        signer: oldSigner,
        transactionId: 'tx-old-queued',
        deviceId: 'device-old',
        sequenceNumber: 7,
        authorityId: 'authority-old',
        authorityEpoch: 1,
      ),
    );
    final staleFailed = OutboxEntry(
      mutation: await _signedMutation(
        signer: oldSigner,
        transactionId: 'tx-old-failed',
        deviceId: 'device-old',
        sequenceNumber: 8,
        authorityId: 'authority-old',
        authorityEpoch: 1,
      ),
    )..state = OutboxState.securityFailure;
    await store.append(staleQueued);
    await store.append(staleFailed);

    final controller = FieldAppController(
      context: _context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.trusted,
      deviceSigner: newSigner,
      deviceKeyId: 'device-key-1',
      authority: _authority(id: 'authority-2', deviceId: 'device-1', epoch: 1),
      outboxStore: store,
    );
    await controller.restoreOutbox();

    final migrated = await controller.reconcileOutboxIdentities();
    expect(migrated, 2);

    final entries = await store.all();
    expect(entries.map((e) => e.state),
        everyElement(OutboxState.queued));
    final sequences =
        entries.map((e) => e.mutation.sequenceNumber).toList()..sort();
    expect(sequences, [1, 2]);
    final ordered = entries.toList()
      ..sort((a, b) => a.mutation.sequenceNumber
          .compareTo(b.mutation.sequenceNumber));
    var lastSequence = 0;
    for (final entry in ordered) {
      expect(entry.mutation.deviceId, 'device-1');
      expect(entry.mutation.authorityId, 'authority-2');
      // Transaction ids are stable so server replays stay idempotent.
      expect(
          {'tx-old-queued', 'tx-old-failed'},
          contains(entry.mutation.transactionId));
      final guard = SyncGuard(
        context: _context,
        deviceId: 'device-1',
        userId: 'user-1',
        deviceState: DeviceTrustState.trusted,
        authority: controller.authority,
        lastSequence: lastSequence,
      ).check(entry.mutation, DateTime.now().toUtc());
      expect(guard.allowed, isTrue,
          reason: 'migrated entry must pass the local sync guard');
      lastSequence = entry.mutation.sequenceNumber;
    }
    expect(CryptographicMerkleHashChain().verifyChain(entries), isTrue);
  });

  test('migrated entries carry the signer key id when the controller has none',
      () async {
    final signer =
        DeviceSigner(await Ed25519().newKeyPair(), keyId: 'signer-key-9');
    final store = InMemoryOutboxStore();
    await store.append(OutboxEntry(
      mutation: await _signedMutation(
        signer: signer,
        transactionId: 'tx-nokey',
        deviceId: 'device-old',
        sequenceNumber: 4,
        authorityId: 'authority-old',
        authorityEpoch: 1,
      ),
    ));

    final controller = FieldAppController(
      context: _context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.trusted,
      deviceSigner: signer,
      deviceKeyId: null,
      authority: _authority(id: 'authority-2', deviceId: 'device-1', epoch: 1),
      outboxStore: store,
    );
    await controller.restoreOutbox();

    expect(await controller.reconcileOutboxIdentities(), 1);
    final migrated = (await store.all()).single;
    // The server rejects Ed25519 transactions with an empty key id, so the
    // fallback must land in both the signature input and the stored field.
    expect(migrated.mutation.keyId, 'signer-key-9');
    expect(migrated.mutation.signature, isNotEmpty);
  });

  test('entries already on the current identity are left untouched', () async {
    final signer =
        DeviceSigner(await Ed25519().newKeyPair(), keyId: 'device-key-1');
    final store = InMemoryOutboxStore();
    await store.append(OutboxEntry(
      mutation: await _signedMutation(
        signer: signer,
        transactionId: 'tx-current',
        deviceId: 'device-1',
        sequenceNumber: 1,
        authorityId: 'authority-2',
        authorityEpoch: 1,
      ),
    ));

    final controller = FieldAppController(
      context: _context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.trusted,
      deviceSigner: signer,
      deviceKeyId: 'device-key-1',
      authority: _authority(id: 'authority-2', deviceId: 'device-1', epoch: 1),
      outboxStore: store,
    );
    await controller.restoreOutbox();

    expect(await controller.reconcileOutboxIdentities(), 0);
    final entries = await store.all();
    expect(entries.single.mutation.sequenceNumber, 1);
    expect(entries.single.attempts, 0);
  });
}
