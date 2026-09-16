import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:http/http.dart' as http;

import 'endpoint_guard.dart';
import 'pinned_http_client.dart';

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

/// Pilot enrollment glue over the loopback enroll endpoint
/// (pkg/onboarding EnrollServer): drives the full challenge -> attest ->
/// submit sequence so the simulator submission reaches Go's
/// ProcessDeviceEnrollment over HTTP. This is the concrete "enrollment UI
/// path" a pilot screen can invoke — no interface, no extra abstractions.
///
/// Always uses [SimulatedAttestationProvider], so the submitted origin is the
/// const [simulatedKeyOrigin] (SOFTWARE): this function accepts no origin
/// override and can never emit SECURE_ENCLAVE or STRONGBOX.
///
/// [endpoint] is the server origin (or any absolute URI; path segments are
/// replaced). Returns the server-side trust record map (DeviceTrustRecord
/// JSON). Throws [StateError] on any transport or contract failure.
Future<Map<String, Object?>> enrollSimulatedDevice({
  required Uri endpoint,
  required String tenantId,
  required String inspectorId,
  required String deviceModel,
  http.Client? client,
}) async {
  // Pilot sim always targets loopback — but guard explicitly so a misconfigured
  // endpoint is caught before key material crosses the wire.
  assertEndpointSafe(endpoint, allowLoopbackHttp: true);
  final httpClient = client ??
      PinnedHttpClient.forEndpoint(endpoint, allowLoopbackHttp: true);

  final challengeResponse = await httpClient.post(
    endpoint.resolve('/enroll/challenge'),
    headers: const {'Content-Type': 'application/json'},
    body: jsonEncode({'tenant_id': tenantId, 'inspector_id': inspectorId}),
  );
  if (challengeResponse.statusCode != 201) {
    throw StateError(
        'enrollment challenge failed: HTTP ${challengeResponse.statusCode}');
  }
  final challengeBody = jsonDecode(challengeResponse.body);
  final challengeId = challengeBody is Map ? challengeBody['challenge_id'] : null;
  final nonce = challengeBody is Map ? challengeBody['nonce'] : null;
  if (challengeId is! String || nonce is! String) {
    throw StateError('enrollment challenge is missing challenge_id/nonce');
  }

  final provider = SimulatedAttestationProvider();
  final bundle = await provider.attestEnrollment(
    challengeId: challengeId,
    inspectorId: inspectorId,
    nonceHex: nonce,
    deviceModel: deviceModel,
  );
  final submission = buildEnrollmentSubmission(
    challengeId: challengeId,
    inspectorId: inspectorId,
    bundle: bundle,
  );

  final submitResponse = await httpClient.post(
    endpoint.resolve('/enroll/submit'),
    headers: const {'Content-Type': 'application/json'},
    body: jsonEncode(submission),
  );
  if (submitResponse.statusCode != 200) {
    throw StateError(
        'enrollment submission failed: HTTP ${submitResponse.statusCode}');
  }
  final record = jsonDecode(submitResponse.body);
  if (record is! Map) {
    throw StateError('enrollment submission returned invalid JSON');
  }
  return Map<String, Object?>.from(record);
}

/// Production enrollment submitting directly to `/api/v1/devices/enroll`.
///
/// Encapsulates the proof-of-possession challenge-response protocol:
/// obtains challenge nonce from the server, signs with client Ed25519 key,
/// and submits the signed EnrollmentVector.
Future<Map<String, Object?>> enrollProductionDevice({
  required Uri endpoint,
  required String tenantId,
  required String organizationId,
  required String userId,
  required String deviceModel,
  http.Client? client,
}) async {
  // Production enrollment must use HTTPS — reject any plaintext endpoint.
  assertEndpointSafe(endpoint, allowLoopbackHttp: false);
  final httpClient = client ??
      PinnedHttpClient.forEndpoint(endpoint, allowLoopbackHttp: false);

  final challengeResponse = await httpClient.post(
    endpoint.resolve('/enroll/challenge'),
    headers: const {'Content-Type': 'application/json'},
    body: jsonEncode({'tenant_id': tenantId, 'inspector_id': userId}),
  );
  if (challengeResponse.statusCode != 201) {
    throw StateError(
        'enrollment challenge failed: HTTP ${challengeResponse.statusCode}');
  }
  final challengeBody = jsonDecode(challengeResponse.body);
  final challengeId = challengeBody is Map ? challengeBody['challenge_id'] : null;
  final nonce = challengeBody is Map ? challengeBody['nonce'] : null;
  if (challengeId is! String || nonce is! String) {
    throw StateError('enrollment challenge is missing challenge_id/nonce');
  }

  final provider = SimulatedAttestationProvider();
  final bundle = await provider.attestEnrollment(
    challengeId: challengeId,
    inspectorId: userId,
    nonceHex: nonce,
    deviceModel: deviceModel,
  );

  final payload = <String, dynamic>{
    'request_id': challengeId,
    'tenant_id': tenantId,
    'organization_id': organizationId,
    'user_id': userId,
    'device_id': bundle.deviceFingerprint,
    'public_key': bundle.devicePublicKeyHex,
    'nonce': nonce,
    'signature': bundle.signedNonceHex,
    'requested_at': DateTime.now().toUtc().toIso8601String(),
    'attestation': {
      'type': bundle.keyOrigin,
      'issuer': 'field_app',
      'receipt': bundle.signedNonceHex,
    },
  };

  final submitResponse = await httpClient.post(
    endpoint.resolve('/api/v1/devices/enroll'),
    headers: const {'Content-Type': 'application/json'},
    body: jsonEncode(payload),
  );
  if (submitResponse.statusCode != 200 && submitResponse.statusCode != 201) {
    throw StateError(
        'production enrollment submission failed: HTTP ${submitResponse.statusCode}');
  }
  final record = jsonDecode(submitResponse.body);
  if (record is! Map) {
    throw StateError('production enrollment submission returned invalid JSON');
  }
  return Map<String, Object?>.from(record);
}

