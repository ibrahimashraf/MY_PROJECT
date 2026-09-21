import 'package:drift/drift.dart';

import 'tables.dart';
import 'platform_database_native.dart'
    if (dart.library.js_interop) 'platform_database_web.dart';

part 'app_database.g.dart';

@DriftDatabase(tables: [OutboxRows])
class AppDatabase extends _$AppDatabase {
  AppDatabase(super.e);

  @override
  int get schemaVersion => 1;

  @override
  MigrationStrategy get migration => MigrationStrategy(
        onCreate: (m) => m.createAll(),
        onUpgrade: (m, from, to) async {},
        beforeOpen: (details) async {
          await customStatement('PRAGMA journal_mode = WAL');
          await customStatement('PRAGMA synchronous = FULL');
          await customStatement('PRAGMA foreign_keys = ON');
        },
      );

  Future<void> insertOutboxEntry({
    required String transactionId,
    required String mutationJson,
    required String state,
    int attempts = 0,
    String? lastError,
    String? chainHash,
    DateTime? acknowledgedAt,
  }) async {
    await into(outboxRows).insert(
      OutboxRowsCompanion.insert(
        transactionId: transactionId,
        mutation: mutationJson,
        state: state,
        attempts: Value(attempts),
        lastError: Value(lastError),
        chainHash: Value(chainHash),
        acknowledgedAt: Value(acknowledgedAt),
      ),
      mode: InsertMode.insertOrIgnore,
    );
  }

  Future<List<OutboxRow>> allOutboxEntries() => select(outboxRows).get();

  Future<List<OutboxRow>> pendingOutboxEntries() =>
      (select(outboxRows)
            ..where((t) => t.state.isIn(['QUEUED', 'UPLOADING'])))
          .get();

  Future<void> markOutboxEntry({
    required String transactionId,
    required String state,
    required int attempts,
    String? lastError,
    DateTime? acknowledgedAt,
  }) async {
    await (update(outboxRows)
          ..where((t) => t.transactionId.equals(transactionId)))
        .write(
      OutboxRowsCompanion(
        state: Value(state),
        attempts: Value(attempts + 1),
        lastError: Value(lastError),
        acknowledgedAt: Value(acknowledgedAt),
      ),
    );
  }

  @override
  Future<void> close() async {
    await super.close();
  }
}

Future<AppDatabase> openAppDatabase({
  String? encryptionKey,
  required Future<String?> Function(String key) readKey,
  required Future<void> Function(String key, String value) writeKey,
}) async {
  final executor = await openPlatformExecutor(
    encryptionKey: encryptionKey,
    readKey: readKey,
    writeKey: writeKey,
  );
  return AppDatabase(executor);
}
