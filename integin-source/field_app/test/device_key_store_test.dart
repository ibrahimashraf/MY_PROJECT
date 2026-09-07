import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/security/transaction_signer.dart';

void main() {
  test('secure device key store creates and reloads a stable Ed25519 identity', () async {
    final storage = _MemorySecretStore();
    final first = await SecureDeviceKeyStore(storage: storage).loadOrCreate('device-1');
    final second = await SecureDeviceKeyStore(storage: storage).loadOrCreate('device-1');

    expect(first.keyId, isNotNull);
    expect(second.keyId, first.keyId);
    expect(storage.values, hasLength(1));
  });
}

class _MemorySecretStore implements SecretValueStore {
  final Map<String, String> values = {};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async {
    values[key] = value;
  }
}
