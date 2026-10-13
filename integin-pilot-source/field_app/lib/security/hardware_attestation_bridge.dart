import 'package:flutter/services.dart';

enum KeyOrigin {
  secureEnclave('SECURE_ENCLAVE'),
  strongBox('STRONGBOX'),
  tee('TEE'),
  software('SOFTWARE'),
  none('NONE');

  final String wireValue;
  const KeyOrigin(this.wireValue);

  static KeyOrigin fromWire(String raw) {
    return KeyOrigin.values.firstWhere(
      (e) => e.wireValue == raw,
      orElse: () => throw SecurityException('Unrecognized key origin: $raw'),
    );
  }

  bool get isHardware =>
      this == KeyOrigin.secureEnclave ||
      this == KeyOrigin.strongBox ||
      this == KeyOrigin.tee;
}

class BootSessionState {
  final String sessionID;
  final int monoNanos;

  const BootSessionState({
    required this.sessionID,
    required this.monoNanos,
  });
}

class AttestationResult {
  final String alias;
  final Uint8List publicKeyDer;
  final KeyOrigin keyOrigin;
  final List<String> certificateChainPem;

  const AttestationResult({
    required this.alias,
    required this.publicKeyDer,
    required this.keyOrigin,
    required this.certificateChainPem,
  });
}

class SecurityException implements Exception {
  final String message;
  final String? code;
  SecurityException(this.message, [this.code]);

  @override
  String toString() => 'SecurityException: $message (code: $code)';
}

abstract class HardwareAttestationProvider {
  Future<AttestationResult> generateAttestedKey({
    required String alias,
    required Uint8List challenge,
    required bool requireUserAuth,
    required int authValidityDurationSeconds,
  });

  Future<Uint8List> signDigest({
    required String alias,
    required Uint8List precomputed32ByteDigest,
  });

  Future<BootSessionState> getBootSession();
}

class MethodChannelHardwareAttestation implements HardwareAttestationProvider {
  static const MethodChannel _channel =
      MethodChannel('com.integin.field/security');

  @override
  Future<AttestationResult> generateAttestedKey({
    required String alias,
    required Uint8List challenge,
    required bool requireUserAuth,
    required int authValidityDurationSeconds,
  }) async {
    try {
      final Map<dynamic, dynamic>? result =
          await _channel.invokeMethod('generateAttestedKey', {
        'alias': alias,
        'challenge': challenge,
        'requireUserAuth': requireUserAuth,
        'authValidityDurationSeconds': authValidityDurationSeconds,
      });

      if (result == null) {
        throw SecurityException('Platform returned null attestation result');
      }

      final rawChain = (result['certificateChainPem'] as List<dynamic>?)
              ?.map((e) => e as String)
              .toList() ??
          <String>[];

      return AttestationResult(
        alias: result['alias'] as String,
        publicKeyDer: Uint8List.fromList(
          (result['publicKeyDer'] as List).cast<int>(),
        ),
        keyOrigin: KeyOrigin.fromWire(result['keyOrigin'] as String),
        certificateChainPem: rawChain,
      );
    } on PlatformException catch (e) {
      throw SecurityException(e.message ?? 'Unknown attestation error', e.code);
    }
  }

  @override
  Future<Uint8List> signDigest({
    required String alias,
    required Uint8List precomputed32ByteDigest,
  }) async {
    if (precomputed32ByteDigest.length != 32) {
      throw SecurityException(
        'Digest must be exactly 32 bytes; got ${precomputed32ByteDigest.length}',
      );
    }

    try {
      final Uint8List? signature =
          await _channel.invokeMethod('signDigest', {
        'alias': alias,
        'digest': precomputed32ByteDigest,
      });

      if (signature == null || signature.isEmpty) {
        throw SecurityException('Platform channel returned empty signature');
      }

      return signature;
    } on PlatformException catch (e) {
      if (e.code == 'AUTH_REQUIRED') {
        throw SecurityException(
          'User authentication required or expired - re-authenticate',
          'AUTH_REQUIRED',
        );
      }
      throw SecurityException(e.message ?? 'Cryptographic signing failure', e.code);
    }
  }

  @override
  Future<BootSessionState> getBootSession() async {
    try {
      final Map<dynamic, dynamic>? result =
          await _channel.invokeMethod('getBootSession');

      if (result == null) {
        throw SecurityException('Platform returned null boot session state');
      }

      return BootSessionState(
        sessionID: result['sessionID'] as String,
        monoNanos: (result['monoNanos'] as num).toInt(),
      );
    } on PlatformException catch (e) {
      throw SecurityException(e.message ?? 'Failed to read kernel boot state', e.code);
    }
  }
}
