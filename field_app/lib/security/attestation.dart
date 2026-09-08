import 'dart:convert';

import 'package:cryptography/cryptography.dart';

/// The only key origin this file may emit. Simulated claims are permanently
/// labeled SOFTWARE so they enroll as CLAIMED-unverified, never VERIFIED: the
/// Go server (pkg/onboarding) only sets AttestationVerified when a hardware
/// chain actually verifies offline. This value is a const and no parameter
/// exists to override it — there is deliberately no code path that can emit
/// SECURE_ENCLAVE or STRONGBOX from the simulator.
const simulatedKeyOrigin = 'SOFTWARE';

/// Evidence bundle produced by an [AttestationProvider] for one enrollment.
class AttestationBundle {
  const AttestationBundle({
    required this.devicePublicKeyHex,
    required this.deviceFingerprint,
    required this.signedNonceHex,
    required this.deviceModel,
    required this.keyOrigin,
    required this.biometricBound,
    this.osVersion = '',
    this.keyAlias = '',
    this.attestationBlob = '',
  });

  /// Hex-encoded Ed25519 public key (lowercase, no 0x prefix).
  final String devicePublicKeyHex;
  final String deviceFingerprint;
  final String signedNonceHex;
  final String deviceModel;
  final String keyOrigin;
  final bool biometricBound;
  final String osVersion;
  final String keyAlias;
  final String attestationBlob;
}

/// Seam for hardware-backed enrollment attestation.
///
/// The simulator is the default dev posture. Hardware providers must fetch a
/// nonce-bound key certificate from the platform keystore and return it as
/// evidence for the server's offline verification.
abstract interface class AttestationProvider {
  Future<AttestationBundle> attestEnrollment({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  });
}

/// Dev-only provider that simulates enrollment attestation with a fresh
/// software Ed25519 keypair per enrollment, signing the challenge nonce.
///
/// Simulated claims are labeled [simulatedKeyOrigin] (SOFTWARE), so the server
/// enrolls them as CLAIMED but never VERIFIED, and receipt policy remains free
/// to reject non-hardware enrollment. This provider must never be used for
/// production hardware evidence.
class SimulatedAttestationProvider implements AttestationProvider {
  final Ed25519 _algorithm = Ed25519();

  @override
  Future<AttestationBundle> attestEnrollment({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  }) async {
    final keyPair = await _algorithm.newKeyPair();
    final publicKey = await keyPair.extractPublicKey();
    final publicKeyHex = _toHex(publicKey.bytes);
    final signature = await _algorithm.sign(
      utf8.encode(nonceHex),
      keyPair: keyPair,
    );
    final fragment = publicKeyHex.substring(0, 6);
    return AttestationBundle(
      devicePublicKeyHex: publicKeyHex,
      deviceFingerprint: 'simulator-$fragment',
      signedNonceHex: _toHex(signature.bytes),
      deviceModel: deviceModel,
      keyOrigin: simulatedKeyOrigin,
      biometricBound: false,
      keyAlias: 'sim-dev-$fragment',
      attestationBlob: '',
    );
  }
}

/// Native hardware attestation seam for real devices.
///
/// Not yet implemented — throws until Android KeyStore StrongBox and iOS
/// Secure Enclave support land. No native code exists in this task.
class HardwareAttestationProvider implements AttestationProvider {
  @override
  Future<AttestationBundle> attestEnrollment({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  }) async {
    throw UnimplementedError(
      'HardwareAttestationProvider is the native seam for Android KeyStore '
      'StrongBox (Keymaster attestation chain via MethodChannel) and iOS '
      'Secure Enclave (flutter_secure_storage iOptions + App Attest '
      'DCAppAttestService). No native code exists yet — enroll with '
      'SimulatedAttestationProvider.',
    );
  }
}

/// Builds the snake_case enrollment submission matching the Go server
/// contract (pkg/onboarding/contracts.go DeviceEnrollmentSubmission).
///
/// Optional claim fields are omitted when empty; the simulator never emits
/// chain_pem or apple_attest_cbor.
Map<String, dynamic> buildEnrollmentSubmission({
  required String challengeId,
  required String inspectorId,
  required AttestationBundle bundle,
}) {
  final attestation = <String, dynamic>{
    'key_origin': bundle.keyOrigin,
    'biometric_bound': bundle.biometricBound,
    if (bundle.osVersion.isNotEmpty) 'os_version': bundle.osVersion,
    if (bundle.attestationBlob.isNotEmpty)
      'attestation_blob': bundle.attestationBlob,
    if (bundle.keyAlias.isNotEmpty) 'key_alias': bundle.keyAlias,
  };
  return <String, dynamic>{
    'challenge_id': challengeId,
    'inspector_id': inspectorId,
    'device_public_key': bundle.devicePublicKeyHex,
    'device_fingerprint': bundle.deviceFingerprint,
    'device_model': bundle.deviceModel,
    'signed_nonce': bundle.signedNonceHex,
    'attestation': attestation,
  };
}

String _toHex(List<int> bytes) {
  final buffer = StringBuffer();
  for (final byte in bytes) {
    buffer.write(byte.toRadixString(16).padLeft(2, '0'));
  }
  return buffer.toString();
}