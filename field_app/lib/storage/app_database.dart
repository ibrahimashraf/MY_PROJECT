import 'dart:io';

import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:flutter/foundation.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

import 'tables.dart';

part 'app_database.g.dart';

@DriftDatabase(tables: [OutboxEntries])
class AppDatabase extends _$AppDatabase {
  AppDatabase(super.e);

  @override
  int get schemaVersion => 1;

  // --- Outbox queries ---

  Future<void> insertOutboxEntry({
    required String transactionId,
    required String mutationJson,
    required String state,
    int attempts = 0,
    String? lastError,
    String? chainHash,
    String? acknowledgedAt,
    required String createdAt,
  }) async {
    await into(outboxEntries).insertOnConflictUpdate(
      OutboxEntriesCompanion.insert(
        transactionId: transactionId,
        mutation: mutationJson,
        state: state,
        attempts: Value(attempts),
        lastError: Value(lastError),
        chainHash: Value(chainHash),
        acknowledgedAt: Value(acknowledgedAt),
        createdAt: createdAt,
      ),
    );
  }

  Future<List<OutboxEntry>> allOutboxEntries() =>
      select(outboxEntries).get();

  Future<List<OutboxEntry>> pendingOutboxEntries() =>
      (select(outboxEntries)
            ..where((t) => t.state.isIn(['QUEUED', 'UPLOADING', 'HELD'])))
          .get();

  Future<void> markOutboxEntry({
    required String transactionId,
    required String state,
    int attempts = 0,
    String? lastError,
    String? acknowledgedAt,
  }) async {
    await (update(outboxEntries)
          ..where((t) => t.transactionId.equals(transactionId)))
        .write(
      OutboxEntriesCompanion(
        state: Value(state),
        attempts: Value(attempts),
        lastError: Value(lastError),
        acknowledgedAt: Value(acknowledgedAt),
      ),
    );
  }
}

/// Opens a platform-adaptive drift database.
///
/// Native (Android/iOS/Windows): SQLite3MC via NativeDatabase.setup callback.
///   - PRAGMA key for encryption at rest (requires [encryptionKey]).
///   - WAL + FULL sync for power-cut integrity.
///
/// Web: drift WASM via WasmDatabase.
///   - Browser handles encryption via OPFS/IndexedDB.
///   - Requires sqlite3.wasm + drift_worker.js in web/ directory.
Future<AppDatabase> openAppDatabase({String? encryptionKey}) async {
  if (kIsWeb) {
    return _openWasmDatabase();
  }
  return _openNativeDatabase(encryptionKey: encryptionKey);
}

Future<AppDatabase> _openWasmDatabase() async {
  // drift WASM uses browser's SQLite build via OPFS.
  // No encryption key needed — browser encrypts OPFS/IndexedDB at OS level.
  //
  // TODO: Add sqlite3.wasm and drift_worker.js to web/ directory.
  // See: https://drift.simonbinder.eu/web/
  throw UnsupportedError(
    'Web WASM database requires sqlite3.wasm setup. '
    'See https://drift.simonbinder.eu/web/',
  );
}

Future<AppDatabase> _openNativeDatabase({String? encryptionKey}) async {
  final dir = await getApplicationDocumentsDirectory();
  final file = File(p.join(dir.path, 'integin_field_app.db'));

  return AppDatabase(
    NativeDatabase(
      file,
      setup: (db) {
        // PRAGMA key must be first statement after open (SQLite 3.48+).
        if (encryptionKey != null && encryptionKey.isNotEmpty) {
          db.execute(
              "PRAGMA key = '${encryptionKey.replaceAll("'", "''")}'");
        }
        db.execute('PRAGMA journal_mode = WAL');
        db.execute('PRAGMA synchronous = FULL');
        db.execute('PRAGMA foreign_keys = ON');
      },
    ),
  );
}
