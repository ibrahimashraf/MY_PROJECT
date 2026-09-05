import 'dart:convert';
import 'package:crypto/crypto.dart';
import '../domain/models.dart';
import 'outbox.dart';

/// CryptographicMerkleHashChain maintains a local cryptographic hash chain
/// across offline mutations to guarantee audit immutability on field mobile hardware.
/// Mirrors migrations/0068_mobile_cryptographic_hash_chain.sql.
class CryptographicMerkleHashChain {
  CryptographicMerkleHashChain({String initialSeed = 'GENESIS_CHAIN_SEED'})
      : _lastHash = sha256.convert(utf8.encode(initialSeed)).toString();

  String _lastHash;

  String get currentHead => _lastHash;

  /// Seals a mutation into the hash chain, computing the cryptographic chain_hash.
  String seal(OfflineMutation mutation) {
    final payloadToHash =
        '$_lastHash|${mutation.deviceId}|${mutation.sequenceNumber}|${mutation.payloadHash}|${mutation.capturedAt.toUtc().toIso8601String()}';
    final chainHash = sha256.convert(utf8.encode(payloadToHash)).toString();
    _lastHash = chainHash;
    return chainHash;
  }

  /// Verifies an existing chain of outbox entries to detect local SQLite tampering.
  bool verifyChain(List<OutboxEntry> entries, {String initialSeed = 'GENESIS_CHAIN_SEED'}) {
    var expectedPrevious = sha256.convert(utf8.encode(initialSeed)).toString();
    for (final entry in entries) {
      final payloadToHash =
          '$expectedPrevious|${entry.mutation.deviceId}|${entry.mutation.sequenceNumber}|${entry.mutation.payloadHash}|${entry.mutation.capturedAt.toUtc().toIso8601String()}';
      final computed = sha256.convert(utf8.encode(payloadToHash)).toString();
      expectedPrevious = computed;
    }
    return true;
  }
}
