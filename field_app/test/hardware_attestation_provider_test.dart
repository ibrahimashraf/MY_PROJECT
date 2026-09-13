import 'dart:convert';

import 'package:crypto/crypto.dart' as crypto;
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/security/attestation.dart';
import 'package:integin_field_app/security/hardware_attestation_provider.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const challengeId = 'chal_hw_test';
  const inspectorId = 'insp_hw_test';
  const nonceHex = 'aabbccddeeff0011223344556677889900112233445566778899aabbccddeeff';
  const deviceModel = 'Rugged Tablet 9';

  HardwareKeyAttestationProvider providerFor(String origin) =>
      HardwareKeyAttestationProvider(
        keyOrigin: origin,
        mint: mockHardwareAttestationMint(keyOrigin: origin),
      );

  test('STRONGBOX provider packages chain_pem into the base64 envelope',
      () async {
    final bundle = await providerFor(keyOriginStrongBox).attestEnrollment(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: deviceModel,
    );

    expect(bundle.keyOrigin, keyOriginStrongBox);
    expect(bundle.deviceFingerprint, startsWith('android-hw-'));
    expect(bundle.deviceModel, deviceModel);
    expect(bundle.biometricBound, isFalse);

    final envelope =
        jsonDecode(utf8.decode(base64Decode(bundle.attestationBlob)))
            as Map<String, dynamic>;
    expect(envelope['nonce'], nonceHex);
    expect(envelope['chain_pem'], isA<List<dynamic>>());
    expect(envelope['chain_pem'], contains(mockLeafPem));
    expect(envelope['chain_pem'], contains(mockRootPem));
    expect(envelope, isNot(contains('apple_attest_cbor_base64')));
  });

  test('SECURE_ENCLAVE provider packages app attest CBOR into the envelope',
      () async {
    final bundle = await providerFor(keyOriginSecureEnclave).attestEnrollment(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: deviceModel,
    );

    expect(bundle.keyOrigin, keyOriginSecureEnclave);
    expect(bundle.deviceFingerprint, startsWith('ios-hw-'));

    final envelope =
        jsonDecode(utf8.decode(base64Decode(bundle.attestationBlob)))
            as Map<String, dynamic>;
    expect(envelope['nonce'], nonceHex);
    expect(envelope['apple_attest_cbor_base64'], isNotEmpty);
    expect(envelope, isNot(contains('chain_pem')));
  });

  test('nonce signature and SPKI are hex-wrapped and deterministic', () async {
    final first = await providerFor(keyOriginStrongBox).attestEnrollment(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: deviceModel,
    );
    final second = await providerFor(keyOriginStrongBox).attestEnrollment(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: deviceModel,
    );

    final expectedSignature =
        crypto.sha256.convert(utf8.encode(nonceHex)).toString();
    expect(first.signedNonceHex, expectedSignature);
    expect(first.signedNonceHex, second.signedNonceHex);
    expect(first.attestationBlob, second.attestationBlob);
    expect(first.devicePublicKeyHex,
        '3059301306072a8648ce3d020106082a8648ce3d030107');
  });

  test('provider rejects unrecognized key origins', () {
    expect(
      () => HardwareKeyAttestationProvider(
        keyOrigin: 'TPM',
        mint: mockHardwareAttestationMint(keyOrigin: keyOriginStrongBox),
      ),
      throwsArgumentError,
    );
  });

  test('evidence minted for a different nonce is rejected (no replay)',
      () async {
    final tampered = HardwareKeyAttestationProvider(
      keyOrigin: keyOriginStrongBox,
      mint: ({
        required challengeId,
        required inspectorId,
        required nonceHex,
        required deviceModel,
      }) async =>
          const HardwareAttestationEvidence(
        nonceHex: 'deadbeef',
        publicKeySpkiHex: '3059301306072a8648ce3d020106082a8648ce3d030107',
        nonceSignatureHex: '00',
      ),
    );
    await expectLater(
      tampered.attestEnrollment(
        challengeId: challengeId,
        inspectorId: inspectorId,
        nonceHex: nonceHex,
        deviceModel: deviceModel,
      ),
      throwsStateError,
    );
  });

  test('empty mint evidence fails closed with StateError', () async {
    final empty = HardwareKeyAttestationProvider(
      keyOrigin: keyOriginStrongBox,
      mint: ({
        required challengeId,
        required inspectorId,
        required nonceHex,
        required deviceModel,
      }) async =>
          HardwareAttestationEvidence(
        nonceHex: nonceHex,
        publicKeySpkiHex: '',
        nonceSignatureHex: '',
      ),
    );
    await expectLater(
      empty.attestEnrollment(
        challengeId: challengeId,
        inspectorId: inspectorId,
        nonceHex: nonceHex,
        deviceModel: deviceModel,
      ),
      throwsStateError,
    );
  });

  test('mock bundle feeds the Go enrollment submission contract', () async {
    final bundle = await providerFor(keyOriginStrongBox).attestEnrollment(
      challengeId: challengeId,
      inspectorId: inspectorId,
      nonceHex: nonceHex,
      deviceModel: deviceModel,
    );
    final submission = buildEnrollmentSubmission(
      challengeId: challengeId,
      inspectorId: inspectorId,
      bundle: bundle,
    );
    final attestation =
        submission['attestation'] as Map<String, dynamic>;
    expect(attestation['key_origin'], keyOriginStrongBox);
    expect(submission['signed_nonce'], bundle.signedNonceHex);
    expect(attestation['attestation_blob'], bundle.attestationBlob);
  });
}