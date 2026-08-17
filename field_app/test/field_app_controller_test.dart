import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/application/field_app_controller.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/sync/sync_client.dart';

class _AppliedTransport implements SyncTransport {
  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async =>
      const SyncResponse(outcome: SyncOutcome.applied);
}

FieldAppController _controller({SyncClient? syncClient}) {
  final issued = DateTime.now().toUtc().subtract(const Duration(minutes: 1));
  const context = TenantContext(
      tenantId: 'tenant-1', organizationId: 'org-1', environment: 'LIVE');
  return FieldAppController(
    context: context,
    deviceId: 'device-1',
    userId: 'user-1',
    deviceState: DeviceTrustState.trusted,
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
    final controller = _controller(
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
    final controller = _controller();

    expect(await controller.flushOutbox(), isEmpty);
    expect(controller.lastError, contains('not configured'));
  });
}
