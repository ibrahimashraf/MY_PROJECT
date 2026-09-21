import 'dart:convert';

import 'package:crypto/crypto.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/evidence/evidence_crypto.dart';
import 'package:integin_field_app/evidence/evidence_store.dart';

void main() {
  test('encrypts and decrypts evidence while detecting tampering', () async {
    final crypto = EvidenceCrypto();
    final key = List<int>.generate(32, (index) => index);
    final encrypted = await crypto.encrypt(
      evidenceId: 'photo-1',
      plaintext: [1, 2, 3, 4],
      keyBytes: key,
    );
    expect(await crypto.decrypt(evidence: encrypted, keyBytes: key), [1, 2, 3, 4]);

    final tampered = EncryptedEvidence(
      evidenceId: encrypted.evidenceId,
      sha256: encrypted.sha256,
      base64Blob: '${encrypted.base64Blob.substring(0, encrypted.base64Blob.length - 2)}aa',
    );
    await expectLater(
      crypto.decrypt(evidence: tampered, keyBytes: key),
      throwsA(anything),
    );
  });

  test('evidence storage is tenant and organization scoped', () async {
    final store = InMemoryEvidenceStore();
    const evidence = EncryptedEvidence(evidenceId: 'photo-1', sha256: 'digest', base64Blob: 'blob');
    await store.put(const EvidenceScope(tenantId: 'tenant-1', organizationId: 'org-1'), evidence);
    expect(
      await store.get(const EvidenceScope(tenantId: 'tenant-1', organizationId: 'org-1'), 'photo-1'),
      isNotNull,
    );
    expect(
      await store.get(const EvidenceScope(tenantId: 'tenant-2', organizationId: 'org-1'), 'photo-1'),
      isNull,
    );
  });

  test('plaintext digest vector is stable across Go and Dart', () {
    expect(
      sha256.convert(utf8.encode('INTEGIN evidence vector')).toString(),
      '114f589228ced30cc36fd6e1be5d405007c57d056f7c44b6d9e5fbfb02584363',
    );
    const evidence = EncryptedEvidence(
      evidenceId: 'vector-1',
      plaintextSha256: '114f589228ced30cc36fd6e1be5d405007c57d056f7c44b6d9e5fbfb02584363',
      ciphertextSha256: 'ciphertext-vector',
      base64Blob: 'ZW5jcnlwdGVkLWV2aWRlbmNl',
    );
    expect(evidence.toJson()['plaintext_sha256'], '114f589228ced30cc36fd6e1be5d405007c57d056f7c44b6d9e5fbfb02584363');
    expect(evidence.toJson()['ciphertext_sha256'], 'ciphertext-vector');
  });
}
