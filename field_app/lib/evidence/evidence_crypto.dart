import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:crypto/crypto.dart' as legacy_crypto;

class EncryptedEvidence {
  const EncryptedEvidence({
    required this.evidenceId,
    required this.base64Blob,
    String? sha256,
    String? plaintextSha256,
    String? ciphertextSha256,
  })  : plaintextSha256 = plaintextSha256 ?? sha256 ?? '',
        ciphertextSha256 = ciphertextSha256 ?? '';

  final String evidenceId;
  final String base64Blob;
  final String plaintextSha256;
  final String ciphertextSha256;

  String get sha256 => plaintextSha256;

  Map<String, Object?> toJson() => {
        'evidence_id': evidenceId,
        'plaintext_sha256': plaintextSha256,
        'ciphertext_sha256': ciphertextSha256,
        'base64_blob': base64Blob,
      };
}

class EvidenceCrypto {
  EvidenceCrypto({AesGcm? algorithm}) : algorithm = algorithm ?? AesGcm.with256bits();

  final AesGcm algorithm;

  Future<EncryptedEvidence> encrypt({
    required String evidenceId,
    required List<int> plaintext,
    required List<int> keyBytes,
  }) async {
    final secretKey = SecretKey(keyBytes);
    final box = await algorithm.encrypt(plaintext, secretKey: secretKey);
    final blob = box.concatenation();
    return EncryptedEvidence(
      evidenceId: evidenceId,
      base64Blob: base64Encode(blob),
      plaintextSha256: legacy_crypto.sha256.convert(plaintext).toString(),
      ciphertextSha256: legacy_crypto.sha256.convert(blob).toString(),
    );
  }

  Future<List<int>> decrypt({
    required EncryptedEvidence evidence,
    required List<int> keyBytes,
  }) async {
    final box = SecretBox.fromConcatenation(
      base64Decode(evidence.base64Blob),
      nonceLength: algorithm.nonceLength,
      macLength: algorithm.macAlgorithm.macLength,
    );
    final encryptedBytes = base64Decode(evidence.base64Blob);
    if (evidence.ciphertextSha256.isNotEmpty &&
        legacy_crypto.sha256.convert(encryptedBytes).toString() != evidence.ciphertextSha256) {
      throw const FormatException('encrypted evidence digest mismatch');
    }
    final plaintext = await algorithm.decrypt(box, secretKey: SecretKey(keyBytes));
    final digest = legacy_crypto.sha256.convert(plaintext).toString();
    if (digest != evidence.plaintextSha256) {
      throw const FormatException('evidence digest mismatch');
    }
    return plaintext;
  }
}
