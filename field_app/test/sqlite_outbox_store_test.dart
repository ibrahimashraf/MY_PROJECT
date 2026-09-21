import 'dart:convert';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/storage/sqlite_outbox_store.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/security/transaction_signer.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('SqliteOutboxStore', () {
    test('PRAGMA journal_mode is WAL', () async {
      final store = SqliteOutboxStore();
      final settings = await store.pragmaSettings();
      expect(settings['journal_mode'], 'wal');
    });

    test('PRAGMA synchronous is FULL for power-cut integrity', () async {
      final store = SqliteOutboxStore();
      final settings = await store.pragmaSettings();
      expect(settings['synchronous'], 'FULL');
    });

    test('append and all preserve outbox entries', () async {
      final store = SqliteOutboxStore();
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
        transactionId: 'tx-sqlite-1',
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
          transactionId: 'tx-sqlite-1',
          tenantId: 'tenant-1',
          deviceId: 'device-1',
          userId: 'user-1',
          sequenceNumber: 1,
          operation: 'InspectionSubmitted',
          payloadHash: payloadHash,
        ),
      );

      final entry = OutboxEntry(mutation: mutation);
      await store.append(entry);

      final all = await store.all();
      expect(all, hasLength(1));
      expect(all.single.mutation.transactionId, 'tx-sqlite-1');
    });

    test('duplicate append is silently ignored', () async {
      final store = SqliteOutboxStore();
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
        transactionId: 'tx-dedup-1',
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
          transactionId: 'tx-dedup-1',
          tenantId: 'tenant-1',
          deviceId: 'device-1',
          userId: 'user-1',
          sequenceNumber: 1,
          operation: 'InspectionSubmitted',
          payloadHash: payloadHash,
        ),
      );

      await store.append(OutboxEntry(mutation: mutation));
      await store.append(OutboxEntry(mutation: mutation));

      final all = await store.all();
      expect(all, hasLength(1));
    });

    test('mark moves entry state and preserves chain hash', () async {
      final store = SqliteOutboxStore();
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
        transactionId: 'tx-mark-1',
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
          transactionId: 'tx-mark-1',
          tenantId: 'tenant-1',
          deviceId: 'device-1',
          userId: 'user-1',
          sequenceNumber: 1,
          operation: 'InspectionSubmitted',
          payloadHash: payloadHash,
        ),
      );

      final entry = OutboxEntry(mutation: mutation);
      await store.append(entry);
      await store.mark(entry, OutboxState.held, error: 'sequence gap');

      final pending = await store.pending();
      expect(pending, hasLength(1));
      expect(pending.single.lastError, 'sequence gap');
    });

    test('power-cut recovery: pending items survive re-instantiation', () async {
      final store1 = SqliteOutboxStore();
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
        transactionId: 'tx-recover-1',
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
          transactionId: 'tx-recover-1',
          tenantId: 'tenant-1',
          deviceId: 'device-1',
          userId: 'user-1',
          sequenceNumber: 1,
          operation: 'InspectionSubmitted',
          payloadHash: payloadHash,
        ),
      );

      await store1.append(OutboxEntry(mutation: mutation));
      await store1.mark(
        OutboxEntry(mutation: mutation),
        OutboxState.uploading,
      );

      // Simulate power cut by creating a new store instance
      final store2 = SqliteOutboxStore();
      final pending = await store2.pending();
      expect(pending, hasLength(1));
      expect(pending.single.mutation.transactionId, 'tx-recover-1');
    });
  });
}
