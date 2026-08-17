import 'dart:convert';

import 'package:crypto/crypto.dart';
import 'package:cryptography/cryptography.dart' hide Hmac;
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

String canonicalTransaction({
  required String transactionId,
  required String tenantId,
  required String deviceId,
  required String userId,
  required int sequenceNumber,
  required String operation,
  required String payloadHash,
}) =>
    '$transactionId|$tenantId|$deviceId|$userId|$sequenceNumber|$operation|$payloadHash';

String canonicalTransactionV1({
  required String transactionId,
  required String tenantId,
  required String organizationId,
  required String environment,
  required String deviceId,
  required String userId,
  required int sequenceNumber,
  required String operation,
  required String entityId,
  required String payloadHash,
  required String authorityId,
  required int authorityEpoch,
  required DateTime capturedAt,
  required String signatureAlgorithm,
  required String keyId,
  String protocolVersion = 'v1',
}) =>
    '$protocolVersion|$transactionId|$tenantId|$organizationId|$environment|$deviceId|$userId|$sequenceNumber|$operation|$entityId|$payloadHash|$authorityId|$authorityEpoch|${canonicalTimestamp(capturedAt)}|$signatureAlgorithm|$keyId';

String canonicalTimestamp(DateTime value) {
  final iso = value.toUtc().toIso8601String();
  if (!iso.contains('.')) return iso;
  final parts = iso.split('.');
  final fraction = parts[1].replaceFirst('Z', '').replaceFirst(RegExp(r'0+$'), '');
  return fraction.isEmpty ? '${parts[0]}Z' : '${parts[0]}.${fraction}Z';
}

String signTransaction({
  required String secret,
  required String transactionId,
  required String tenantId,
  required String deviceId,
  required String userId,
  required int sequenceNumber,
  required String operation,
  required String payloadHash,
}) {
  final canonical = canonicalTransaction(
    transactionId: transactionId,
    tenantId: tenantId,
    deviceId: deviceId,
    userId: userId,
    sequenceNumber: sequenceNumber,
    operation: operation,
    payloadHash: payloadHash,
  );
  return Hmac(sha256, utf8.encode(secret))
      .convert(utf8.encode(canonical))
      .toString();
}

class DeviceSigner {
  DeviceSigner(this.keyPair, {this.keyId});

  final SimpleKeyPair keyPair;
  final String? keyId;

  Future<String> publicKeyBase64() async {
    final publicKey = await keyPair.extractPublicKey();
    return base64Encode(publicKey.bytes);
  }

  Future<String> signV1({
    required String transactionId,
    required String tenantId,
    required String organizationId,
    required String environment,
    required String deviceId,
    required String userId,
    required int sequenceNumber,
    required String operation,
    required String entityId,
    required String payloadHash,
    required String authorityId,
    required int authorityEpoch,
    required DateTime capturedAt,
    String? keyId,
  }) async {
    final resolvedKeyId = keyId ?? this.keyId;
    if (resolvedKeyId == null || resolvedKeyId.isEmpty) {
      throw StateError('device key id is required for Ed25519 signing');
    }
    final canonical = canonicalTransactionV1(
      transactionId: transactionId,
      tenantId: tenantId,
      organizationId: organizationId,
      environment: environment,
      deviceId: deviceId,
      userId: userId,
      sequenceNumber: sequenceNumber,
      operation: operation,
      entityId: entityId,
      payloadHash: payloadHash,
      authorityId: authorityId,
      authorityEpoch: authorityEpoch,
      capturedAt: capturedAt,
      signatureAlgorithm: 'Ed25519',
      keyId: resolvedKeyId,
    );
    final signature = await Ed25519().sign(
      utf8.encode(canonical),
      keyPair: keyPair,
    );
    return base64Encode(signature.bytes);
  }
}

abstract interface class SecretValueStore {
  Future<String?> read(String key);

  Future<void> write(String key, String value);
}

class FlutterSecretValueStore implements SecretValueStore {
  FlutterSecretValueStore({FlutterSecureStorage? storage}) : storage = storage ?? const FlutterSecureStorage();

  final FlutterSecureStorage storage;

  @override
  Future<String?> read(String key) => storage.read(key: key);

  @override
  Future<void> write(String key, String value) => storage.write(key: key, value: value);
}

class SecureDeviceKeyStore {
  SecureDeviceKeyStore({SecretValueStore? storage}) : storage = storage ?? FlutterSecretValueStore();

  final SecretValueStore storage;
  final Ed25519 algorithm = Ed25519();

  Future<DeviceSigner> loadOrCreate(String deviceId) async {
    if (deviceId.trim().isEmpty) throw ArgumentError.value(deviceId, 'deviceId');
    final key = 'integin.device.$deviceId.ed25519.seed';
    final storedSeed = await storage.read(key);
    final keyPair = storedSeed == null
        ? await algorithm.newKeyPair()
        : await algorithm.newKeyPairFromSeed(base64Decode(storedSeed));
    if (storedSeed == null) {
      await storage.write(key, base64Encode(await keyPair.extractPrivateKeyBytes()));
    }
    final publicKey = await keyPair.extractPublicKey();
    return DeviceSigner(
      keyPair,
      keyId: sha256.convert(publicKey.bytes).toString(),
    );
  }
}
