import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/sync/sync_client.dart';
import 'package:integin_field_app/sync/sync_guard.dart';

class FakeTransport implements SyncTransport {
  FakeTransport(this.response);

  final SyncResponse response;
  int submitted = 0;

  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async {
    submitted += 1;
    return response;
  }
}

class ThrowingTransport implements SyncTransport {
  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async => throw StateError('offline');
}

OfflineMutation makeMutation() {
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
  return OfflineMutation(
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
}

SyncGuard makeGuard() {
  final issued = DateTime.utc(2026, 8, 13, 9);
  const context = TenantContext(
    tenantId: 'tenant-1',
    organizationId: 'org-1',
    environment: 'LIVE',
  );
  return SyncGuard(
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
      scopes: const ['assigned-work'],
      capabilities: const ['inspection.perform'],
      procedureVersion: 'proc-1',
      issuedAt: issued,
      expiresAt: issued.add(const Duration(hours: 4)),
      signature: 'signature',
    ),
    lastSequence: 0,
  );
}

void main() {
  test('flush maps a server Applied response and preserves acknowledgement', () async {
    final store = InMemoryOutboxStore();
    final entry = OutboxEntry(mutation: makeMutation());
    await store.append(entry);
    final transport = FakeTransport(const SyncResponse(outcome: SyncOutcome.applied));
    final results = await SyncClient(store: store, transport: transport).flush(
          guard: makeGuard(),
          at: DateTime.utc(2026, 8, 13, 10),
        );

    expect(results, [SyncOutcome.applied]);
    expect(transport.submitted, 1);
    expect(entry.state, OutboxState.applied);
    expect(entry.acknowledgedAt, isNotNull);
  });

  test('local guard failure becomes a visible security outcome without transport call', () async {
    final store = InMemoryOutboxStore();
    final entry = OutboxEntry(mutation: makeMutation());
    await store.append(entry);
    final transport = FakeTransport(const SyncResponse(outcome: SyncOutcome.applied));
    final results = await SyncClient(store: store, transport: transport).flush(
          guard: SyncGuard(
            context: makeGuard().context,
            deviceId: 'device-1',
            userId: 'user-1',
            deviceState: DeviceTrustState.revoked,
            authority: makeGuard().authority,
            lastSequence: 0,
          ),
          at: DateTime.utc(2026, 8, 13, 10),
        );

    expect(results, [SyncOutcome.securityFailure]);
    expect(transport.submitted, 0);
    expect(entry.state, OutboxState.securityFailure);
  });

  test('transport failure returns a retryable queued outcome', () async {
    final store = InMemoryOutboxStore();
    final entry = OutboxEntry(mutation: makeMutation());
    await store.append(entry);
    final results = await SyncClient(store: store, transport: ThrowingTransport()).flush(
      guard: makeGuard(),
      at: DateTime.utc(2026, 8, 13, 10),
    );

    expect(results, [SyncOutcome.queued]);
    expect(entry.state, OutboxState.queued);
    expect(entry.lastError, contains('Transport unavailable'));
  });
}
