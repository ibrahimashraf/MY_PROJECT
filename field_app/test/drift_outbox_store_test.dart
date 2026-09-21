import 'package:drift/drift.dart' hide isNotNull;
import 'package:drift/native.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/outbox.dart';
import 'package:integin_field_app/storage/app_database.dart';
import 'package:integin_field_app/storage/drift_outbox_store.dart';

/// Creates an in-memory test database.
AppDatabase testDatabase() => AppDatabase(NativeDatabase.memory());

TenantContext _testContext() => const TenantContext(
      tenantId: 'tenant-test',
      organizationId: 'org-test',
      environment: 'LIVE',
    );

OfflineMutation _testMutation({
  String transactionId = 'tx-test-1',
  int sequenceNumber = 1,
}) {
  return OfflineMutation(
    transactionId: transactionId,
    context: _testContext(),
    deviceId: 'device-test',
    userId: 'user-test',
    sequenceNumber: sequenceNumber,
    operation: 'InspectionSubmitted',
    entityId: 'inspection-test',
    payload: const {'inspection_id': 'inspection-test'},
    capturedAt: DateTime.utc(2026, 9, 21, 12),
    authorityId: 'authority-test',
    authorityEpoch: 1,
    signature: 'sig-test',
  );
}

void main() {
  group('DriftOutboxStore', () {
    late AppDatabase db;
    late DriftOutboxStore store;

    setUp(() {
      db = testDatabase();
      store = DriftOutboxStore(db);
    });

    tearDown(() async {
      await store.close();
    });

    test('append and retrieve all entries', () async {
      final mutation = _testMutation();
      await store.append(OutboxEntry(mutation: mutation));

      final all = await store.all();
      expect(all, hasLength(1));
      expect(all.single.mutation.transactionId, 'tx-test-1');
      expect(all.single.state, OutboxState.queued);
    });

    test('append ignores duplicate transactionId', () async {
      final mutation = _testMutation();
      await store.append(OutboxEntry(mutation: mutation));
      await store.append(OutboxEntry(mutation: mutation));

      final all = await store.all();
      expect(all, hasLength(1));
    });

    test('pending returns only QUEUED and UPLOADING entries', () async {
      final m1 = _testMutation(transactionId: 'tx-1');
      final m2 = _testMutation(transactionId: 'tx-2', sequenceNumber: 2);
      final m3 = _testMutation(transactionId: 'tx-3', sequenceNumber: 3);

      final e1 = OutboxEntry(mutation: m1);
      final e2 = OutboxEntry(mutation: m2);
      final e3 = OutboxEntry(mutation: m3);

      await store.append(e1);
      await store.append(e2);
      await store.append(e3);

      // Mark e3 as HELD — should not appear in pending.
      await store.mark(e3, OutboxState.held);

      final pending = await store.pending();
      expect(pending, hasLength(2));
      expect(
        pending.map((e) => e.mutation.transactionId),
        containsAll(['tx-1', 'tx-2']),
      );
    });

    test('mark updates state and auto-increments attempts', () async {
      final mutation = _testMutation();
      final entry = OutboxEntry(mutation: mutation);
      await store.append(entry);

      expect(entry.attempts, 0);

      await store.mark(entry, OutboxState.uploading);
      final afterFirstMark = await store.all();
      expect(afterFirstMark.single.state, OutboxState.uploading);
      expect(afterFirstMark.single.attempts, 1);

      await store.mark(entry, OutboxState.applied);
      final afterSecondMark = await store.all();
      expect(afterSecondMark.single.state, OutboxState.applied);
      expect(afterSecondMark.single.attempts, 2);
      expect(afterSecondMark.single.acknowledgedAt, isNotNull);
    });

    test('mark with error stores lastError', () async {
      final mutation = _testMutation();
      final entry = OutboxEntry(mutation: mutation);
      await store.append(entry);

      await store.mark(entry, OutboxState.rejected, error: 'server error');
      final after = await store.all();
      expect(after.single.lastError, 'server error');
      expect(after.single.state, OutboxState.rejected);
    });

    test('corrupt JSON returns securityFailure entry', () async {
      // Insert raw corrupt data directly into the database.
      // Use drift's companion to avoid DateTimeColumn parsing issues.
      await db.into(db.outboxRows).insert(
            OutboxRowsCompanion.insert(
              transactionId: 'tx-corrupt',
              mutation: '{bad json',
              state: 'QUEUED',
              createdAt: Value(DateTime.now()),
            ),
          );

      final all = await store.all();
      expect(all, hasLength(1));
      expect(all.single.state, OutboxState.securityFailure);
      expect(all.single.lastError, contains('Corrupt mutation data'));
    });

    test('close disposes the underlying database', () async {
      final mutation = _testMutation();
      await store.append(OutboxEntry(mutation: mutation));
      await store.close();

      // After close, operations should throw.
      expect(
        () => store.all(),
        throwsA(anything),
      );
    });
  });
}
