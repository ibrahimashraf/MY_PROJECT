import 'dart:convert';

import '../domain/models.dart';
import '../outbox/outbox.dart' show OutboxEntry, OutboxState, OutboxStore;
import 'app_database.dart';

/// Drift-backed outbox store.
///
/// Native: encrypted at rest via SQLite3MC (PRAGMA key).
/// Web: browser encrypts OPFS/IndexedDB at OS level.
class DriftOutboxStore implements OutboxStore {
  DriftOutboxStore(this._db);

  final AppDatabase _db;

  @override
  Future<void> append(OutboxEntry entry) async {
    await _db.insertOutboxEntry(
      transactionId: entry.mutation.transactionId,
      mutationJson: jsonEncode(entry.mutation.toJson()),
      state: entry.state.name.toUpperCase(),
      attempts: entry.attempts,
      lastError: entry.lastError,
      chainHash: entry.chainHash,
      acknowledgedAt: entry.acknowledgedAt,
    );
  }

  @override
  Future<List<OutboxEntry>> all() async {
    final rows = await _db.allOutboxEntries();
    return rows.map(_rowToEntry).toList();
  }

  @override
  Future<List<OutboxEntry>> pending() async {
    final rows = await _db.pendingOutboxEntries();
    return rows.map(_rowToEntry).toList();
  }

  @override
  Future<void> mark(OutboxEntry entry, OutboxState state,
      {String? error}) async {
    // Read current attempts from DB to avoid stale value.
    final current = await (_db.select(_db.outboxRows)
          ..where((t) => t.transactionId.equals(entry.mutation.transactionId)))
        .getSingleOrNull();
    final currentAttempts = current?.attempts ?? entry.attempts;

    await _db.markOutboxEntry(
      transactionId: entry.mutation.transactionId,
      state: state.name.toUpperCase(),
      attempts: currentAttempts,
      lastError: error,
      acknowledgedAt:
          (state == OutboxState.applied || state == OutboxState.duplicate)
              ? DateTime.now().toUtc()
              : null,
    );
  }

  OutboxEntry _rowToEntry(OutboxRow row) {
    try {
      final mutation = OfflineMutation.fromJson(
        Map<String, Object?>.from(jsonDecode(row.mutation) as Map),
      );
      return OutboxEntry(
        mutation: mutation,
        chainHash: row.chainHash,
      )
        ..state = OutboxState.values.firstWhere(
          (v) => v.name.toUpperCase() == row.state,
          orElse: () => OutboxState.queued,
        )
        ..attempts = row.attempts
        ..lastError = row.lastError
        ..acknowledgedAt = row.acknowledgedAt;
    } on FormatException {
      return OutboxEntry(
        mutation: OfflineMutation(
          transactionId: row.transactionId,
          context: const TenantContext(
            tenantId: '',
            organizationId: '',
            environment: 'LIVE',
          ),
          deviceId: '',
          userId: '',
          sequenceNumber: 0,
          operation: 'corrupt',
          entityId: '',
          payload: const {},
          capturedAt: DateTime.now().toUtc(),
          authorityId: '',
          authorityEpoch: 0,
          signature: '',
        ),
      )
        ..state = OutboxState.securityFailure
        ..lastError = 'Corrupt mutation data: ${row.transactionId}';
    } on TypeError {
      return OutboxEntry(
        mutation: OfflineMutation(
          transactionId: row.transactionId,
          context: const TenantContext(
            tenantId: '',
            organizationId: '',
            environment: 'LIVE',
          ),
          deviceId: '',
          userId: '',
          sequenceNumber: 0,
          operation: 'corrupt',
          entityId: '',
          payload: const {},
          capturedAt: DateTime.now().toUtc(),
          authorityId: '',
          authorityEpoch: 0,
          signature: '',
        ),
      )
        ..state = OutboxState.securityFailure
        ..lastError = 'Malformed mutation data: ${row.transactionId}';
    }
  }

  @override
  Future<void> close() async {
    await _db.close();
  }
}
