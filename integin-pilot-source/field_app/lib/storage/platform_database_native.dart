import 'dart:io';
import 'dart:math';

import 'package:drift/drift.dart';
import 'package:drift/native.dart';
import 'package:path/path.dart' as p;
import 'package:path_provider/path_provider.dart';

String generateEncryptionKey() {
  final random = Random.secure();
  final key = Uint8List.fromList(
    List<int>.generate(32, (_) => random.nextInt(256)),
  );
  return key.map((b) => b.toRadixString(16).padLeft(2, '0')).join();
}

Future<QueryExecutor> openPlatformExecutor({
  String? encryptionKey,
  required Future<String?> Function(String key) readKey,
  required Future<void> Function(String key, String value) writeKey,
}) async {
  String? key = encryptionKey;
  if (key == null || key.isEmpty) {
    key = await readKey('INTEGIN.database.encryption_key');
  }
  if (key == null || key.isEmpty) {
    key = generateEncryptionKey();
    await writeKey('INTEGIN.database.encryption_key', key);
  }

  final dir = await getApplicationDocumentsDirectory();
  final file = File(p.join(dir.path, 'integin_field_app.db'));

  return NativeDatabase(
    file,
    setup: (db) {
      db.execute("PRAGMA key = \"x'$key'\"");
    },
  );
}
