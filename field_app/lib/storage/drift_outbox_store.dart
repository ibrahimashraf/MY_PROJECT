import 'dart:convert';

import '../domain/models.dart';
import '../outbox/outbox.dart' show OutboxEntry, OutboxState, OutboxStore;
import 'app_database.dart' hide OutboxEntry;

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
      acknowledgedAt: entry.acknowledgedAt?.toUtc().toIso8601String(),
      createdAt: DateTime.now().toUtc().toIso8601String(),
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
    await _db.markOutboxEntry(
      transactionId: entry.mutation.transactionId,
      state: state.name.toUpperCase(),
      attempts: entry.attempts,
      lastError: error,
      acknowledgedAt:
          (state == OutboxState.applied || state == OutboxState.duplicate)
              ? DateTime.now().toUtc().toIso8601String()
              : null,
    );
  }

  OutboxEntry _rowToEntry(dynamic row) {
    final mutation = OfflineMutation.fromJson(
      Map<String, Object?>.from(jsonDecode(row.mutation as String)),
    );
    return OutboxEntry(
      mutation: mutation,
      chainHash: row.chainHash,
    )..state = OutboxState.values.firstWhere(
        (v) => v.name.toUpperCase() == row.state,
        orElse: () => OutboxState.queued,
      )
      ..attempts = row.attempts
      ..lastError = row.lastError
      ..acknowledgedAt = row.acknowledgedAt != null
          ? DateTime.parse(row.acknowledgedAt as String)
          : null;
  }
}
