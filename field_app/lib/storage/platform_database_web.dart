import 'package:drift/drift.dart';
import 'package:drift/wasm.dart';

Future<QueryExecutor> openPlatformExecutor({
  String? encryptionKey,
  required Future<String?> Function(String key) readKey,
  required Future<void> Function(String key, String value) writeKey,
}) async {
  final result = await WasmDatabase.open(
    databaseName: 'integin_field_app',
    sqlite3Uri: Uri.parse('/sqlite3.wasm'),
    driftWorkerUri: Uri.parse('/drift_worker.js'),
  );
  return result.resolvedExecutor;
}
