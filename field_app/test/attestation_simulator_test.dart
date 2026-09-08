import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/security/attestation.dart';

void main() {
  const nonceHex = 'deadbeefcafe';
  const challengeId = 'chal_$nonceHex';
  const inspectorId = 'insp_dev-sim-01';

  Future<AttestationBundle> enroll() async {
    final provider = SimulatedAttestationProvider();
    return provider.attestEnrollment(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: 'android-tablet-sim',
    );
  }

  test('simulator bundle verifies: server ed25519.Verify will accept it',
      () async {
    final bundle = await enroll();
    final publicKey = SimplePublicKey(
      _hexDecode(bundle.devicePublicKeyHex),
      type: KeyPairType.ed25519,
    );
    final valid = await Ed25519().verify(
      utf8.encode(nonceHex),
      signature: Signature(
        _hexDecode(bundle.signedNonceHex),
        publicKey: publicKey,
      ),
    );
    expect(valid, isTrue);
  });

  test('submission map keys exactly equal the Go contract set', () async {
    final bundle = await enroll();
    final submission = buildEnrollmentSubmission(
      challengeId: challengeId,
      inspectorId: inspectorId,
      bundle: bundle,
    );

    expect(
      submission.keys.toSet(),
      {
        'challenge_id',
        'inspector_id',
        'device_public_key',
        'device_fingerprint',
        'device_model',
        'signed_nonce',
        'attestation',
      },
    );
    final attestation = submission['attestation'] as Map<String, dynamic>;
    expect(attestation.keys.toSet(), {'key_origin', 'biometric_bound', 'key_alias'});
  });

  test('keyOrigin is always SOFTWARE across 3 enrollments', () async {
    for (var i = 0; i < 3; i++) {
      final bundle = await enroll();
      expect(bundle.keyOrigin, simulatedKeyOrigin);
      final submission = buildEnrollmentSubmission(
        challengeId: challengeId,
        inspectorId: inspectorId,
        bundle: bundle,
      );
      final attestation = submission['attestation'] as Map<String, dynamic>;
      expect(attestation['key_origin'], 'SOFTWARE');
      expect(attestation.containsKey('chain_pem'), isFalse);
      expect(attestation.containsKey('apple_attest_cbor'), isFalse);
    }
  });

  test('two enrollments produce different keys', () async {
    final first = await enroll();
    final second = await enroll();
    expect(first.devicePublicKeyHex, isNot(second.devicePublicKeyHex));
    expect(first.signedNonceHex, isNot(second.signedNonceHex));
  });

  test('HardwareAttestationProvider throws UnimplementedError', () async {
    final provider = HardwareAttestationProvider();
    await expectLater(
      provider.attestEnrollment(
        challengeId: challengeId,
        inspectorId: inspectorId,
        nonceHex: nonceHex,
        deviceModel: 'hardware-seam',
      ),
      throwsUnimplementedError,
    );
  });

  test('JSON round-trip preserves all submission fields', () async {
    final bundle = await enroll();
    final submission = buildEnrollmentSubmission(
      challengeId: challengeId,
      inspectorId: inspectorId,
      bundle: bundle,
    );
    final decoded = jsonDecode(jsonEncode(submission)) as Map<String, dynamic>;
    expect(decoded, equals(submission));
    expect(decoded['device_public_key'], bundle.devicePublicKeyHex);
    expect(decoded['device_fingerprint'], bundle.deviceFingerprint);
    expect(decoded['signed_nonce'], bundle.signedNonceHex);
    final attestation = decoded['attestation'] as Map<String, dynamic>;
    expect(attestation['key_origin'], 'SOFTWARE');
    expect(attestation['biometric_bound'], isFalse);
    expect(attestation['key_alias'], bundle.keyAlias);
  });
}

List<int> _hexDecode(String hex) {
  final bytes = <int>[];
  for (var i = 0; i < hex.length; i += 2) {
    bytes.add(int.parse(hex.substring(i, i + 2), radix: 16));
  }
  return bytes;
}