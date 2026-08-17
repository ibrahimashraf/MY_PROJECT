import 'dart:convert';
import 'dart:io';

import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/security/transaction_signer.dart';

void main() {
  test('v1 signed transaction fixture is canonical and rejects byte alteration', () async {
    final fixtureFile = File('../contracts/vectors/v1/signed_transaction_ed25519_v1.json');
    expect(await fixtureFile.exists(), isTrue, reason: 'Generate the pilot Go fixture before running Flutter parity tests.');

    final fixture = jsonDecode(await fixtureFile.readAsString()) as Map<String, dynamic>;
    final valid = fixture['valid'] as Map<String, dynamic>;
    final byteAltered = fixture['byte_altered'] as Map<String, dynamic>;

    final validCanonical = valid['canonical_utf8'] as String;
    final parts = validCanonical.split('|');
    expect(parts, hasLength(16));
    expect(parts[0], 'v1');

    final rebuiltCanonical = canonicalTransactionV1(
      transactionId: parts[1],
      tenantId: parts[2],
      organizationId: parts[3],
      environment: parts[4],
      deviceId: parts[5],
      userId: parts[6],
      sequenceNumber: int.parse(parts[7]),
      operation: parts[8],
      entityId: parts[9],
      payloadHash: parts[10],
      authorityId: parts[11],
      authorityEpoch: int.parse(parts[12]),
      capturedAt: DateTime.parse(parts[13]),
      signatureAlgorithm: parts[14],
      keyId: parts[15],
    );
    expect(rebuiltCanonical, validCanonical);

    final algorithm = Ed25519();
    final publicKey = SimplePublicKey(
      base64Decode(valid['public_key_base64'] as String),
      type: KeyPairType.ed25519,
    );
    final signature = Signature(
      base64Decode(valid['signature_base64'] as String),
      publicKey: publicKey,
    );

    expect(
      await algorithm.verify(utf8.encode(validCanonical), signature: signature),
      valid['expected_accepted'],
    );
    expect(
      await algorithm.verify(utf8.encode(byteAltered['canonical_utf8'] as String), signature: signature),
      isFalse,
    );
    expect(byteAltered['expected_reason_code'], 'SIGNATURE_INVALID');
  });
}
