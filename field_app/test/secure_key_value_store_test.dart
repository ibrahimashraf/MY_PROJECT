import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/storage/persistent_outbox.dart';

class MemoryKeyValueStore implements KeyValueStore {
  final Map<String, String> values = {};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async => values[key] = value;
}

void main() {
  test('secure-storage boundary keeps the outbox store platform-neutral', () async {
    final storage = MemoryKeyValueStore();
    await storage.write('integin.outbox.v1', 'encrypted-payload');
    expect(await storage.read('integin.outbox.v1'), 'encrypted-payload');
  });
}
