import 'dart:convert';

import 'package:crypto/crypto.dart' as crypto;
import 'package:flutter/services.dart';

import 'package:integin_field_app/security/attestation.dart';

/// Key-origin descriptors mirrored from `pkg/onboarding` KeyOrigin. The
/// server (pkg/onboarding AttestationPolicy) gates these origins on
/// hardware-attestation posture when RequireHardware is set.
const keyOriginSecureEnclave = 'SECURE_ENCLAVE';
const keyOriginStrongBox = 'STRONGBOX';

/// Native hardware attestation channel shared by all real-device mints.
const MethodChannel _attestationChannel = MethodChannel('integin.attestation/v1');

/// Raw hardware mint outcome handed back from the platform keystore bridge
/// (Android KeyMint/StrongBox, Apple App Attest) or a hermetic test fixture.
class HardwareAttestationEvidence {
  const HardwareAttestationEvidence({
    required this.nonceHex,
    required this.publicKeySpkiHex,
    required this.nonceSignatureHex,
    this.chainPem = const [],
    this.appAttestCborBase64 = '',
    this.osVersion = '',
    this.securityLevel = 'UNKNOWN',
  });

  /// The challenge nonce the mint was generated for. The provider rejects any
  /// evidence minted for a different nonce (no cross-submission replay).
  final String nonceHex;

  /// Hex-encoded SPKI of the hardware-bound public key.
  final String publicKeySpkiHex;

  /// Hex-encoded signature the hardware key produced over the nonce.
  final String nonceSignatureHex;

  /// Android key-attestation chain, leaf-first PEM-encoded (STRONGBOX).
  final List<String> chainPem;

  /// Apple App Attest attestation object, base64-encoded (SECURE_ENCLAVE).
  final String appAttestCborBase64;
  final String osVersion;

  /// Native-reported security level: STRONGBOX | TEE | UNKNOWN.
  final String securityLevel;
}

/// Platform seam that mints raw hardware evidence for one challenge nonce.
/// Inject a mock in hermetic unit tests; the default is the native
/// MethodChannel bridge (Android StrongBox/KeyMint).
typedef HardwareAttestationMint = Future<HardwareAttestationEvidence> Function({
  required String challengeId,
  required String inspectorId,
  required String nonceHex,
  required String deviceModel,
});

/// Real hardware key-attestation provider for the [keyOriginSecureEnclave]
/// (Apple Secure Enclave, App Attest) and [keyOriginStrongBox] (Android
/// StrongBox/KeyMint) descriptors. Each origin packages its payload through
/// [packageHardwareEvidence], a canonical base64 envelope carrying the
/// platform evidence plus the nonce binding.
class HardwareKeyAttestationProvider implements AttestationProvider {
  HardwareKeyAttestationProvider({
    required this.keyOrigin,
    HardwareAttestationMint? mint,
  }) : mint = mint ?? mintFromMethodChannel {
    if (keyOrigin != keyOriginSecureEnclave && keyOrigin != keyOriginStrongBox) {
      throw ArgumentError.value(
        keyOrigin,
        'keyOrigin',
        'must be one of $keyOriginSecureEnclave, $keyOriginStrongBox',
      );
    }
  }

  final String keyOrigin;
  final HardwareAttestationMint mint;

  @override
  Future<AttestationBundle> attestEnrollment({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  }) async {
    final evidence = await mint(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: deviceModel,
    );
    if (evidence.nonceHex != nonceHex) {
      throw StateError(
        'hardware attestation mint returned evidence for a different nonce — '
        'rejecting the bundle',
      );
    }
    if (evidence.publicKeySpkiHex.isEmpty || evidence.nonceSignatureHex.isEmpty) {
      throw StateError(
          'hardware attestation mint returned no public key or nonce signature');
    }
    final spkiHex = evidence.publicKeySpkiHex;
    final fragment = spkiHex.length >= 6 ? spkiHex.substring(0, 6) : spkiHex;
    final fingerprint = keyOrigin == keyOriginStrongBox
        ? 'android-hw-$fragment'
        : 'ios-hw-$fragment';
    return AttestationBundle(
      devicePublicKeyHex: spkiHex,
      deviceFingerprint: fingerprint,
      signedNonceHex: evidence.nonceSignatureHex,
      deviceModel: deviceModel,
      keyOrigin: keyOrigin,
      biometricBound: false,
      osVersion: evidence.osVersion,
      keyAlias: '${keyOrigin.toLowerCase()}-$fragment',
      attestationBlob: packageHardwareEvidence(evidence),
    );
  }
}

/// Packages raw mint evidence into the canonical envelope both origins share:
/// a base64-encoded JSON object carrying the per-platform payload
/// (`chain_pem` for STRONGBOX, `apple_attest_cbor_base64` for SECURE_ENCLAVE)
/// plus the nonce binding. Package consumers decode it back to JSON before
/// forwarding the platform evidence to the server's offline verifier.
String packageHardwareEvidence(HardwareAttestationEvidence evidence) {
  final payload = <String, dynamic>{
    'nonce': evidence.nonceHex,
    if (evidence.chainPem.isNotEmpty) 'chain_pem': evidence.chainPem,
    if (evidence.appAttestCborBase64.isNotEmpty)
      'apple_attest_cbor_base64': evidence.appAttestCborBase64,
  };
  return base64Encode(utf8.encode(jsonEncode(payload)));
}

/// Default mint bridging the Android KeyMint/StrongBox native channel
/// (`integin.attestation/v1` mintChain). The native layer reports the actual
/// verified security level (STRONGBOX or TEE); TEE evidence still reaches the
/// server's STRONGBOX chain path, where chain verification establishes the
/// real posture offline.
Future<HardwareAttestationEvidence> mintFromMethodChannel({
  required String challengeId,
  required String inspectorId,
  required String nonceHex,
  required String deviceModel,
}) async {
  final result =
      await _attestationChannel.invokeMethod<Map<Object?, Object?>>(
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
  return HardwareAttestationEvidence(
    nonceHex: nonceHex,
    publicKeySpkiHex: (result['public_key_spki_hex'] as String?) ?? '',
    nonceSignatureHex: (result['nonce_signature_hex'] as String?) ?? '',
    chainPem:
        ((result['chain_pem'] as List<Object?>?) ?? const []).cast<String>(),
    osVersion: (result['os_version'] as String?) ?? '',
    securityLevel: (result['security_level'] as String?) ?? 'UNKNOWN',
  );
}

/// Legacy Android-only channel seam: zero-arg construction, maps TEE to the
/// STRONGBOX chain path, forwards key_alias verbatim. Superseded by
/// [HardwareKeyAttestationProvider] but kept for the existing channel tests
/// and enrollment-screen wiring.
class HardwareAttestationProvider implements AttestationProvider {
  @override
  Future<AttestationBundle> attestEnrollment({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  }) async {
    final result = await _attestationChannel
        .invokeMethod<Map<Object?, Object?>>('mintChain', nonceHex);
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

/// Deterministic PEM-shaped mock certificates for hermetically exercising the
/// STRONGBOX chain path without any platform keystore.
const mockLeafPem =
    '-----BEGIN CERTIFICATE-----\nMOCKLEAFCERTIFICATE\n-----END CERTIFICATE-----\n';
const mockRootPem =
    '-----BEGIN CERTIFICATE-----\nMOCKROOTCERTIFICATE\n-----END CERTIFICATE-----\n';

/// Hermetic mock mint: no platform channels, no OS keystore, fully offline.
/// The nonce signature is a stable SHA-256 derivation of the nonce so two
/// enrollments with the same nonce produce byte-identical evidence — the
/// assertion contract for the deterministic packaging tests.
HardwareAttestationMint mockHardwareAttestationMint({
  required String keyOrigin,
  String osVersion = 'mock-os',
}) {
  return ({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  }) async {
    final signatureHex =
        crypto.sha256.convert(utf8.encode(nonceHex)).toString();
    return HardwareAttestationEvidence(
      nonceHex: nonceHex,
      publicKeySpkiHex: '3059301306072a8648ce3d020106082a8648ce3d030107',
      nonceSignatureHex: signatureHex,
      chainPem: keyOrigin == keyOriginStrongBox
          ? const [mockLeafPem, mockRootPem]
          : const [],
      appAttestCborBase64: keyOrigin == keyOriginSecureEnclave
          ? base64Encode(utf8.encode(
              jsonEncode({'nonce': nonceHex, 'app_id': 'com.integin.pilot'})))
          : '',
      osVersion: osVersion,
      securityLevel:
          keyOrigin == keyOriginStrongBox ? 'STRONGBOX' : 'UNKNOWN',
    );
  };
}