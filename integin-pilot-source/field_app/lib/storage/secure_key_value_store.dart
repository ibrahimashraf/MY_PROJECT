import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'persistent_outbox.dart';

class SecureKeyValueStore implements KeyValueStore {
  SecureKeyValueStore({FlutterSecureStorage? storage})
      : storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage storage;

  @override
  Future<String?> read(String key) => storage.read(key: key);

  @override
  Future<void> write(String key, String value) => storage.write(key: key, value: value);
}
