import 'dart:convert';

import 'package:cryptography/cryptography.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/security/attestation.dart';
import 'package:integin_field_app/security/hardware_attestation_provider.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
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
    expect(attestation.keys.toSet(),
        {'key_origin', 'biometric_bound', 'key_alias'});
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

  test(
      'HardwareAttestationProvider mints via the native channel and maps '
      'TEE to the STRONGBOX chain path', () async {
    const channel = MethodChannel('integin.attestation/v1');
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
      expect(call.method, 'mintChain');
      expect(call.arguments, nonceHex);
      return const {
        'chain_pem': ['LEAFCERT', 'INTERCERT'],
        'security_level': 'TEE',
        'key_alias': 'integin_probe_abcd1234',
        'public_key_spki_hex': '3059301306072a8648ce3d020106082a8648ce3d030107',
        'nonce_signature_hex': '3044022001ff6e0609329a4e',
        'os_version': '16',
      };
    });
    addTearDown(() => TestDefaultBinaryMessengerBinding
        .instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null));

    final bundle = await HardwareAttestationProvider().attestEnrollment(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: 'MI 9',
    );

    expect(bundle.keyOrigin, 'STRONGBOX');
    expect(bundle.biometricBound, isFalse);
    expect(bundle.keyAlias, 'integin_probe_abcd1234');
    expect(bundle.osVersion, '16');
    expect(bundle.deviceModel, 'MI 9');
    expect(bundle.attestationBlob, contains('chain_pem'));
  });

  test('HardwareAttestationProvider surfaces native mint errors', () async {
    const channel = MethodChannel('integin.attestation/v1');
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, (call) async {
      return const {'error': 'StrongBox and TEE both unavailable'};
    });
    addTearDown(() => TestDefaultBinaryMessengerBinding
        .instance.defaultBinaryMessenger
        .setMockMethodCallHandler(channel, null));

    await expectLater(
      HardwareAttestationProvider().attestEnrollment(
        challengeId: challengeId,
        inspectorId: inspectorId,
        nonceHex: nonceHex,
        deviceModel: 'hardware-seam',
      ),
      throwsStateError,
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
