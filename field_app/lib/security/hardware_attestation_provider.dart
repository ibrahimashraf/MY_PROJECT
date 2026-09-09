import 'dart:convert';

import 'package:flutter/services.dart';

import 'package:integin_field_app/security/attestation.dart';

/// Native hardware attestation seam for real devices.
///
/// Bridges the Android KeyStore key-attestation mint (StrongBox-first, TEE
/// fallback) via the "integin.attestation/v1" MethodChannel implemented in
/// MainActivity.kt. The chain is a genuine Google-signed hardware
/// certification (leaf-first PEMs) bound to the enrollment nonce via the
/// Keymaster attestationChallenge; the Go verifier (pkg/onboarding) accepts
/// TEE(1) and StrongBox(2) chains offline against the Google roots.
///
/// Lives in its own file (not attestation.dart) so the source of the hardware
/// bridge stays importable by the bare-Dart round-trip harness, which runs
/// outside the Flutter engine and cannot load dart:ui.
class HardwareAttestationProvider implements AttestationProvider {
  static const MethodChannel _channel = MethodChannel('integin.attestation/v1');

  @override
  Future<AttestationBundle> attestEnrollment({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  }) async {
    final result = await _channel.invokeMethod<Map<Object?, Object?>>(
      'mintChain',
      nonceHex,
    );
    if (result == null) {
      throw StateError(
        'hardware attestation unavailable: the native mintChain channel '
        'returned nothing (run on a real Android device)',
      );
    }
    final error = result['error'] as String?;
    if (error != null && error.isNotEmpty) {
      throw StateError('hardware attestation mint failed: $error');
    }
    final chainPem =
        ((result['chain_pem'] as List<Object?>?) ?? const []).cast<String>();
    final securityLevel = (result['security_level'] as String?) ?? 'UNKNOWN';
    // Key-origin mapping. The server's STRONGBOX origin means "Android
    // hardware, verify the chain", and VerifyAttestationChain records the
    // *verified* level (StrongBox or TEE) — the claimed label is not what
    // gates trust. TEE is therefore enrolled as STRONGBOX so the chain path
    // runs; anything non-hardware is SOFTWARE and is never claimed as
    // hardware. Never map TEE to anything weaker.
    final keyOrigin = switch (securityLevel) {
      'STRONGBOX' || 'TEE' => 'STRONGBOX',
      _ => 'SOFTWARE',
    };
    final spkiHex = (result['public_key_spki_hex'] as String?) ?? '';
    final fragment = spkiHex.length >= 6 ? spkiHex.substring(0, 6) : spkiHex;
    return AttestationBundle(
      devicePublicKeyHex: spkiHex,
      deviceFingerprint: 'android-hw-$fragment',
      signedNonceHex: (result['nonce_signature_hex'] as String?) ?? '',
      deviceModel: deviceModel,
      keyOrigin: keyOrigin,
      biometricBound: false,
      osVersion: (result['os_version'] as String?) ?? '',
      keyAlias: (result['key_alias'] as String?) ?? '',
      attestationBlob: jsonEncode({'chain_pem': chainPem}),
    );
  }
}
