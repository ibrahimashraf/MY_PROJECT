import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/storage/persistent_outbox.dart';

class MemoryKeyValueStore implements KeyValueStore {
  final Map<String, String> values = {};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async => values[key] = value;
}

void main() {
  test('JSON outbox survives a new store instance and preserves state', () async {
    const context = TenantContext(
      tenantId: 'tenant-1',
      organizationId: 'org-1',
      environment: 'LIVE',
    );
    final payload = <String, Object?>{'inspection_id': 'inspection-1'};
    final payloadHash = OfflineMutation(
      transactionId: 'hash',
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      sequenceNumber: 1,
      operation: 'InspectionSubmitted',
      entityId: 'inspection-1',
      payload: payload,
      capturedAt: DateTime.utc(2026, 8, 13, 10),
      authorityId: 'authority-1',
      authorityEpoch: 1,
      signature: '',
    ).payloadHash;
    final mutation = OfflineMutation(
      transactionId: 'tx-1',
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      sequenceNumber: 1,
      operation: 'InspectionSubmitted',
      entityId: 'inspection-1',
      payload: payload,
      capturedAt: DateTime.utc(2026, 8, 13, 10),
      authorityId: 'authority-1',
      authorityEpoch: 1,
      signature: signTransaction(
        secret: 'secret',
        transactionId: 'tx-1',
        tenantId: 'tenant-1',
        deviceId: 'device-1',
        userId: 'user-1',
        sequenceNumber: 1,
        operation: 'InspectionSubmitted',
        payloadHash: payloadHash,
      ),
    );
    final storage = MemoryKeyValueStore();
    final first = JsonOutboxStore(storage: storage);
    await first.append(OutboxEntry(mutation: mutation));

    final second = JsonOutboxStore(storage: storage);
    final pending = await second.pending();
    expect(pending, hasLength(1));
    expect(pending.single.mutation.transactionId, 'tx-1');
    await second.mark(pending.single, OutboxState.held, error: 'sequence gap');
    expect((await second.pending()).single.lastError, 'sequence gap');
  });

  test('JSON outbox recovers an interrupted upload into a retryable queue state', () async {
    const context = TenantContext(tenantId: 'tenant-1', organizationId: 'org-1', environment: 'LIVE');
    final mutation = OfflineMutation(
      transactionId: 'tx-recover',
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      sequenceNumber: 1,
      operation: 'InspectionSubmitted',
      entityId: 'inspection-1',
      payload: const {'inspection_id': 'inspection-1'},
      capturedAt: DateTime.utc(2026, 8, 13, 10),
      authorityId: 'authority-1',
      authorityEpoch: 1,
      signature: 'signature',
    );
    final storage = MemoryKeyValueStore();
    final first = JsonOutboxStore(storage: storage);
    final entry = OutboxEntry(mutation: mutation);
    await first.append(entry);
    await first.mark(entry, OutboxState.uploading);

    final second = JsonOutboxStore(storage: storage);
    final pending = await second.pending();
    expect(pending, hasLength(1));
    expect(pending.single.state, OutboxState.queued);
    expect(pending.single.lastError, contains('interrupted upload'));
  });
}
