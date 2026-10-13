import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/security/transaction_signer.dart';

void main() {
  test('manifest-read proof canonicalization matches shared Go/Dart vector',
      () async {
    final file = File('../contracts/vectors/v1/manifest_read_proof_v1.json');
    final vector = jsonDecode(await file.readAsString()) as Map<String, dynamic>;
    expect(vector['protocol_version'], manifestReadProofProtocolVersion);
    expect(vector['purpose'], manifestReadProofPurpose);
    expect(vector['signature_algorithm'], manifestReadProofSignatureAlgorithm);
    expect(
      canonicalManifestProof(
        requestId: vector['request_id'] as String,
        deviceId: vector['device_id'] as String,
        authorityId: vector['authority_id'] as String,
        authorityEpoch: vector['authority_epoch'] as int,
        inspectionId: vector['inspection_id'] as String,
        issuedAt: DateTime.parse(vector['issued_at'] as String),
        expiresAt: DateTime.parse(vector['expires_at'] as String),
        keyId: vector['key_id'] as String,
      ),
      vector['canonical_utf8'],
    );
  });
}
