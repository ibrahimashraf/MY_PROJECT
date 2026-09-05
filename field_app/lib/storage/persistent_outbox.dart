import 'dart:convert';

import 'package:shared_preferences/shared_preferences.dart';

import '../outbox/merkle_hash_chain.dart';
import '../outbox/outbox.dart';

abstract interface class KeyValueStore {
  Future<String?> read(String key);
  Future<void> write(String key, String value);
}

class SharedPreferencesKeyValueStore implements KeyValueStore {
  SharedPreferencesKeyValueStore(this.preferences);

  final SharedPreferences preferences;

  @override
  Future<String?> read(String key) async => preferences.getString(key);

  @override
  Future<void> write(String key, String value) async {
    await preferences.setString(key, value);
  }
}

class JsonOutboxStore implements OutboxStore {
  JsonOutboxStore({
    required this.storage,
    this.key = 'integin.outbox.v1',
    CryptographicMerkleHashChain? hashChain,
  }) : _hashChain = hashChain ?? CryptographicMerkleHashChain();

  final KeyValueStore storage;
  final String key;
  final CryptographicMerkleHashChain _hashChain;
  final List<OutboxEntry> _entries = [];
  bool _loaded = false;

  CryptographicMerkleHashChain get hashChain => _hashChain;

  Future<void> load() async {
    if (_loaded) return;
    final encoded = await storage.read(key);
    var recoveredUpload = false;
    if (encoded != null && encoded.isNotEmpty) {
      final raw = jsonDecode(encoded) as List<dynamic>;
      _entries
        ..clear()
        ..addAll(
          raw.map(
            (item) =>
                OutboxEntry.fromJson(Map<String, Object?>.from(item as Map)),
          ),
        );
      for (final entry in _entries) {
        if (entry.state == OutboxState.uploading) {
          entry.state = OutboxState.queued;
          entry.lastError = 'Recovered after an interrupted upload.';
          recoveredUpload = true;
        }
      }
    }
    _loaded = true;
    if (recoveredUpload) await _persist();
  }

  @override
  Future<void> append(OutboxEntry entry) async {
    await load();
    if (_entries.any((candidate) =>
        candidate.mutation.transactionId == entry.mutation.transactionId)) {
      return;
    }
    // Automatically seal mutation with Merkle hash chain if not already sealed
    if (entry.chainHash == null || entry.chainHash!.isEmpty) {
      entry.chainHash = _hashChain.seal(entry.mutation);
    }
    _entries.add(entry);
    await _persist();
  }

  @override
  Future<List<OutboxEntry>> all() async {
    await load();
    return List.unmodifiable(_entries);
  }

  @override
  Future<List<OutboxEntry>> pending() async {
    await load();
    return List.unmodifiable(_entries.where((entry) => entry.isPending));
  }

  @override
  Future<void> mark(OutboxEntry entry, OutboxState state,
      {String? error}) async {
    await load();
    entry.state = state;
    entry.attempts += 1;
    entry.lastError = error;
    if (state == OutboxState.applied || state == OutboxState.duplicate) {
      entry.acknowledgedAt = DateTime.now().toUtc();
    }
    await _persist();
  }

  Future<void> _persist() async {
    await storage.write(
      key,
      jsonEncode(_entries.map((entry) => entry.toJson()).toList()),
    );
  }
}
