import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/application/field_app_controller.dart';
import 'package:integin_field_app/assurance/rule_bundle.dart';
import 'package:integin_field_app/assurance/rule_evaluator.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/sync/sync_client.dart';

import 'package:cryptography/cryptography.dart';
import 'package:integin_field_app/security/transaction_signer.dart';

class _AppliedTransport implements SyncTransport {
  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async =>
      const SyncResponse(outcome: SyncOutcome.applied);
}

Future<FieldAppController> _controller({SyncClient? syncClient}) async {
  final issued = DateTime.now().toUtc().subtract(const Duration(minutes: 1));
  const context = TenantContext(
      tenantId: 'tenant-1', organizationId: 'org-1', environment: 'LIVE');
  final signer = DeviceSigner(
    await Ed25519().newKeyPair(),
    keyId: 'device-key-1',
  );
  return FieldAppController(
    context: context,
    deviceId: 'device-1',
    userId: 'user-1',
    deviceState: DeviceTrustState.trusted,
    deviceSigner: signer,
    deviceKeyId: 'device-key-1',
    authority: OfflineAuthority(
      id: 'authority-1',
      deviceId: 'device-1',
      context: context,
      userId: 'user-1',
      epoch: 1,
      scopes: const ['inspection.perform'],
      capabilities: const ['inspection.perform'],
      procedureVersion: 'test',
      issuedAt: issued,
      expiresAt: issued.add(const Duration(hours: 1)),
      signature: 'test-signature',
    ),
    outboxStore: syncClient?.store ?? InMemoryOutboxStore(),
    syncClient: syncClient,
  );
}

InspectionWorkPack _workPack() => InspectionWorkPack(
      inspectionId: 'inspection-1',
      rootAssetId: 'asset-1',
      inspectionType: 'test',
      procedureVersion: 'test',
      packageId: 'test-work-package',
      packageVersion: 1,
      schemaVersion: 1,
      packageHash: 'sha256:test-work-package-hash',
      scheduledDate: DateTime.now().toUtc(),
      items: const [
        ChecklistItem(
            id: 'item-1',
            sectionId: 'section-1',
            prompt: 'Prompt',
            assetId: 'asset-1')
      ],
    );

void main() {
  test('flushOutbox maps applied result into durable controller state',
      () async {
    final store = InMemoryOutboxStore();
    final controller = await _controller(
        syncClient: SyncClient(store: store, transport: _AppliedTransport()));
    final workPack = _workPack();
    controller.beginInspection(workPack);
    controller.recordResponse(
        item: workPack.items.single, response: 'acceptable');
    expect(await controller.queueForSync(), isTrue);

    final outcomes = await controller.flushOutbox();

    expect(outcomes, [SyncOutcome.applied]);
    expect(controller.connectivity, ConnectivityState.online);
    expect(controller.queuedCount, 0);
    expect(controller.lastSuccessfulSync, isNotNull);
  });

  test('flushOutbox reports a deliberate configuration boundary', () async {
    final controller = await _controller();

    expect(await controller.flushOutbox(), isEmpty);
    expect(controller.lastError, contains('not configured'));
  });

  test('editOutboxEntry unseals queued entry back to draft and marks entry abandoned', () async {
    final store = InMemoryOutboxStore();
    final controller = await _controller(
        syncClient: SyncClient(store: store, transport: _AppliedTransport()));
    final workPack = _workPack();
    controller.beginInspection(workPack);
    controller.recordResponse(
        item: workPack.items.single, response: 'initial-val', note: 'initial-note');
    expect(await controller.queueForSync(notes: 'general-note'), isTrue);

    final entries = await store.all();
    expect(entries.length, 1);
    expect(entries.first.state, OutboxState.queued);
    expect(controller.activeDraft, isNull);

    // Operator unseals entry for editing
    final unsealed = await controller.editOutboxEntry(entries.first, workPack);
    expect(unsealed, isTrue);

    // Active draft restored with previous data
    final draft = controller.activeDraft;
    expect(draft, isNotNull);
    expect(draft!.notes, 'general-note');
    expect(draft.findings['item-1']?.response, 'initial-val');

    // Original entry marked abandoned
    final updatedEntries = await store.all();
    expect(updatedEntries.first.state, OutboxState.abandoned);
    expect(controller.queuedCount, 0);

    // Operator edits and re-queues
    controller.recordResponse(
        item: workPack.items.single, response: 'edited-val', note: 'edited-note');
    expect(await controller.queueForSync(notes: 'updated-general-note'), isTrue);

    expect(controller.activeDraft, isNull);
    final finalEntries = await store.all();
    // Replaced in-place: zero abandoned ghosts left behind
    expect(finalEntries.length, 1);
    expect(finalEntries.single.state, OutboxState.queued);
    expect(finalEntries.single.mutation.payload['notes'], 'updated-general-note');
  });

  test('clearFailures marks failed mutations as abandoned instead of applied', () async {
    final store = InMemoryOutboxStore();
    final controller = await _controller(
        syncClient: SyncClient(store: store, transport: _AppliedTransport()));
    final workPack = _workPack();
    controller.beginInspection(workPack);
    controller.recordResponse(item: workPack.items.single, response: 'val');
    await controller.queueForSync();

    final entry = (await store.all()).first;
    await store.mark(entry, OutboxState.rejected, error: 'validation error');
    controller.outboxSummary = OutboxSummary.fromEntries(await store.all());
    expect(controller.failedCount, 1);

    final cleared = await controller.clearFailures();
    expect(cleared, 1);
    expect(controller.failedCount, 0);

    final updated = (await store.all()).first;
    expect(updated.state, OutboxState.abandoned);
  });

  test('queueForSync embeds evaluationOutcome in outbox payload and verifies hash integrity', () async {
    final store = InMemoryOutboxStore();
    final controller = await _controller(
        syncClient: SyncClient(store: store, transport: _AppliedTransport()));
    final workPack = _workPack();
    controller.beginInspection(workPack);
    controller.recordResponse(item: workPack.items.single, response: 'pass');

    const outcome = EvaluationOutcome(
      bundleId: 'ISO4309_LIFTING_ROPES',
      version: '1.0.0',
      ruleHash: 'sha256:iso4309',
      results: [],
      highestSeverity: AssuranceSeverity.info,
      isNonCompliant: false,
      isCondemned: false,
      reasons: [],
      state: 'ASSURED',
    );
    controller.recordEvaluationOutcome(outcome);

    expect(await controller.queueForSync(notes: 'evaluation-test'), isTrue);

    final entries = await store.all();
    expect(entries.length, 1);
    final payload = entries.first.mutation.payload;
    expect(payload['notes'], 'evaluation-test');
    expect(payload['evaluation_outcome'], isNotNull);
    final outcomeMap = payload['evaluation_outcome'] as Map<String, dynamic>;
    expect(outcomeMap['state'], 'ASSURED');
    expect(outcomeMap['bundle_id'], 'ISO4309_LIFTING_ROPES');
  });
}
