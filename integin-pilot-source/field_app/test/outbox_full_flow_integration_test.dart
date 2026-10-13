import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/application/field_app_controller.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/sync/sync_client.dart';
import 'package:integin_field_app/sync/sync_guard.dart';

class RecordingTransport implements SyncTransport {
  final List<SyncResponse> responses;
  int callIndex = 0;
  final List<String> submittedTxIds = [];

  RecordingTransport(this.responses);

  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async {
    submittedTxIds.add(mutation.transactionId);
    final response = responses[callIndex];
    callIndex += 1;
    return response;
  }
}

class FailingTransport implements SyncTransport {
  int calls = 0;

  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async {
    calls += 1;
    throw StateError('network unreachable');
  }
}

TenantContext makeContext() => const TenantContext(
      tenantId: 'tenant-int',
      organizationId: 'org-int',
      environment: 'LIVE',
    );

OfflineAuthority makeAuthority({int epoch = 1}) {
  final issued = DateTime.now().toUtc().subtract(const Duration(minutes: 5));
  return OfflineAuthority(
    id: 'auth-int-$epoch',
    deviceId: 'device-int',
    context: makeContext(),
    userId: 'user-int',
    epoch: epoch,
    scopes: const ['assigned-work'],
    capabilities: const ['inspection.perform'],
    procedureVersion: 'proc-v1',
    issuedAt: issued,
    expiresAt: issued.add(const Duration(hours: 8)),
    signature: 'sig-int',
  );
}

InspectionWorkPack makeWorkPack(String id) => InspectionWorkPack(
      inspectionId: id,
      rootAssetId: 'asset-int',
      inspectionType: 'integration',
      procedureVersion: 'proc-v1',
      packageId: 'pkg-int',
      packageVersion: 1,
      schemaVersion: 1,
      packageHash: 'sha256:pkg-hash-int',
      scheduledDate: DateTime.now().toUtc(),
      items: const [
        ChecklistItem(
          id: 'item-int-1',
          sectionId: 'section-int',
          prompt: 'Integration item',
          assetId: 'asset-int',
        ),
      ],
    );

Future<DeviceSigner> makeSigner() async {
  final algorithm = Ed25519();
  final keyPair = await algorithm.newKeyPair();
  final publicKey = await keyPair.extractPublicKey();
  return DeviceSigner(
    keyPair,
    keyId: 'int-key-${publicKey.bytes.length}',
  );
}

void main() {
  test('full outbox lifecycle: queue → flush applied → summary reflects ack',
      () async {
    final store = InMemoryOutboxStore();
    final signer = await makeSigner();
    final transport = RecordingTransport([
      const SyncResponse(outcome: SyncOutcome.applied),
    ]);
    final authority = makeAuthority();
    final controller = FieldAppController(
      context: makeContext(),
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: authority,
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: transport),
    );
    await controller.restoreOutbox();

    final workPack = makeWorkPack('insp-int-1');
    controller.beginInspection(workPack);
    controller.recordResponse(
      item: workPack.items.single,
      response: 'ok',
    );

    final queued = await controller.queueForSync(notes: 'integration test');
    expect(queued, isTrue);
    expect(controller.queuedCount, 1);
    expect(controller.lastError, isNull);

    final outcomes = await controller.flushOutbox();
    expect(outcomes, [SyncOutcome.applied]);
    expect(controller.queuedCount, 0);
    expect(transport.submittedTxIds.length, 1);

    final all = await store.all();
    expect(all.length, 1);
    expect(all.first.state, OutboxState.applied);
    expect(all.first.acknowledgedAt, isNotNull);
  });

  test('transport failure preserves entry for retry', () async {
    final store = InMemoryOutboxStore();
    final signer = await makeSigner();
    final transport = FailingTransport();
    final controller = FieldAppController(
      context: makeContext(),
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: makeAuthority(),
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: transport),
    );
    await controller.restoreOutbox();

    final workPack = makeWorkPack('insp-int-fail');
    controller.beginInspection(workPack);
    controller.recordResponse(item: workPack.items.single, response: 'fail');
    await controller.queueForSync();

    final outcomes = await controller.flushOutbox();
    expect(outcomes, [SyncOutcome.queued]);
    expect(transport.calls, greaterThanOrEqualTo(1));

    final pending = await store.pending();
    expect(pending.length, 1);
    expect(pending.first.state, OutboxState.queued);
    expect(pending.first.lastError, contains('Transport unavailable'));
    expect(pending.first.attempts, greaterThanOrEqualTo(1));
  });

  test('second flush after transport recovery sends previously failed entry',
      () async {
    final store = InMemoryOutboxStore();
    final signer = await makeSigner();
    final transport = FailingTransport();
    final controller = FieldAppController(
      context: makeContext(),
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: makeAuthority(),
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: transport),
    );
    await controller.restoreOutbox();

    final workPack = makeWorkPack('insp-int-retry');
    controller.beginInspection(workPack);
    controller.recordResponse(item: workPack.items.single, response: 'retry');
    await controller.queueForSync();
    await controller.flushOutbox();

    expect(transport.calls, 1);

    final recoveredTransport = RecordingTransport([
      const SyncResponse(outcome: SyncOutcome.applied),
    ]);
    final controller2 = FieldAppController(
      context: makeContext(),
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: makeAuthority(),
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: recoveredTransport),
    );
    await controller2.restoreOutbox();

    final outcomes = await controller2.flushOutbox();
    expect(outcomes, [SyncOutcome.applied]);
    expect(recoveredTransport.submittedTxIds.length, 1);
    expect(controller2.queuedCount, 0);
  });

  test('duplicate server response marks entry as duplicate, not applied',
      () async {
    final store = InMemoryOutboxStore();
    final signer = await makeSigner();
    final transport = RecordingTransport([
      const SyncResponse(outcome: SyncOutcome.duplicate),
    ]);
    final controller = FieldAppController(
      context: makeContext(),
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: makeAuthority(),
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: transport),
    );
    await controller.restoreOutbox();

    final workPack = makeWorkPack('insp-int-dup');
    controller.beginInspection(workPack);
    controller.recordResponse(item: workPack.items.single, response: 'dup');
    await controller.queueForSync();
    final outcomes = await controller.flushOutbox();

    expect(outcomes, [SyncOutcome.duplicate]);
    final all = await store.all();
    expect(all.first.state, OutboxState.duplicate);
    expect(all.first.acknowledgedAt, isNotNull);
  });

  test('sequence gap holds entry without transport call', () async {
    final store = InMemoryOutboxStore();
    final signer = await makeSigner();
    final transport = RecordingTransport([
      const SyncResponse(outcome: SyncOutcome.applied),
    ]);

    final context = makeContext();
    final authority = makeAuthority();

    final controller = FieldAppController(
      context: context,
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: authority,
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: transport),
    );

    final workPack = makeWorkPack('insp-int-seq');
    controller.beginInspection(workPack);
    controller.recordResponse(item: workPack.items.single, response: 'seq');
    await controller.queueForSync();

    final all = await store.all();
    expect(all.length, 1);
    expect(all.first.mutation.sequenceNumber, 1);

    final guard = SyncGuard(
      context: context,
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: authority,
      lastSequence: 5,
    );
    final outcomes = await SyncClient(store: store, transport: transport).flush(
      guard: guard,
      at: DateTime.now().toUtc(),
    );

    expect(outcomes, [SyncOutcome.held]);
    expect(transport.submittedTxIds, isEmpty);

    final held = await store.all();
    expect(held.first.state, OutboxState.held);
  });

  test('multiple entries flush in sequence order', () async {
    final store = InMemoryOutboxStore();
    final signer = await makeSigner();
    final transport = RecordingTransport([
      const SyncResponse(outcome: SyncOutcome.applied),
      const SyncResponse(outcome: SyncOutcome.applied),
      const SyncResponse(outcome: SyncOutcome.duplicate),
    ]);
    final controller = FieldAppController(
      context: makeContext(),
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: makeAuthority(),
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: transport),
    );
    await controller.restoreOutbox();

    for (var i = 0; i < 3; i++) {
      final wp = makeWorkPack('insp-multi-$i');
      controller.beginInspection(wp);
      controller.recordResponse(item: wp.items.single, response: 'v$i');
      await controller.queueForSync();
    }

    expect(controller.queuedCount, 3);
    final outcomes = await controller.flushOutbox();
    expect(outcomes, [
      SyncOutcome.applied,
      SyncOutcome.applied,
      SyncOutcome.duplicate,
    ]);
    expect(controller.queuedCount, 0);
    expect(transport.submittedTxIds.length, 3);

    final all = await store.all();
    expect(all.length, 3);
    for (final entry in all) {
      expect(entry.acknowledgedAt, isNotNull);
    }
  });

  test('connectivity transitions through sync lifecycle', () async {
    final store = InMemoryOutboxStore();
    final signer = await makeSigner();
    final transport = RecordingTransport([
      const SyncResponse(outcome: SyncOutcome.applied),
    ]);
    final controller = FieldAppController(
      context: makeContext(),
      deviceId: 'device-int',
      userId: 'user-int',
      deviceState: DeviceTrustState.trusted,
      authority: makeAuthority(),
      outboxStore: store,
      deviceSigner: signer,
      deviceKeyId: signer.keyId,
      syncClient: SyncClient(store: store, transport: transport),
    );
    await controller.restoreOutbox();

    expect(controller.connectivity, ConnectivityState.offline);

    final wp = makeWorkPack('insp-conn');
    controller.beginInspection(wp);
    controller.recordResponse(item: wp.items.single, response: 'conn');
    await controller.queueForSync();
    controller.connectivity = ConnectivityState.online;
    expect(controller.connectivity, ConnectivityState.online);

    await controller.flushOutbox();
    expect(controller.connectivity, ConnectivityState.online);
  });
}
