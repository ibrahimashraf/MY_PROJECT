import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/security/transaction_signer.dart';

void main() {
  test('tenant context rejects cross-environment or empty identity', () {
    expect(
      const TenantContext(
        tenantId: 'tenant-1',
        organizationId: 'org-1',
        environment: 'LIVE',
      ).isValid,
      isTrue,
    );
    expect(
      const TenantContext(
        tenantId: '',
        organizationId: 'org-1',
        environment: 'LIVE',
      ).isValid,
      isFalse,
    );
    expect(
      const TenantContext(
        tenantId: 'tenant-1',
        organizationId: 'org-1',
        environment: 'DEV',
      ).isValid,
      isFalse,
    );
  });

  test('offline authority is valid only inside its bounded window', () {
    final issued = DateTime.utc(2026, 8, 13, 10);
    final authority = OfflineAuthority(
      id: 'auth-1',
      deviceId: 'device-1',
      context: const TenantContext(
        tenantId: 'tenant-1',
        organizationId: 'org-1',
        environment: 'LIVE',
      ),
      userId: 'user-1',
      epoch: 3,
      scopes: const ['site-1'],
      capabilities: const ['inspection.perform'],
      procedureVersion: 'proc-7',
      issuedAt: issued,
      expiresAt: issued.add(const Duration(hours: 8)),
      signature: 'server-signature',
    );

    expect(authority.isValidAt(issued.add(const Duration(minutes: 1))), isTrue);
    expect(authority.isValidAt(issued), isFalse);
    expect(authority.isValidAt(issued.add(const Duration(hours: 8))), isFalse);
  });

  test('canonical JSON is stable for map insertion order', () {
    expect(
      canonicalJson({'b': 2, 'a': 1}),
      equals(canonicalJson({'a': 1, 'b': 2})),
    );
  });

  test('transaction signature follows the Go-compatible canonical form', () {
    final signature = signTransaction(
      secret: 'secret',
      transactionId: 'tx-1',
      tenantId: 'tenant-1',
      deviceId: 'device-1',
      userId: 'user-1',
      sequenceNumber: 1,
      operation: 'FindingRecorded',
      payloadHash: 'hash',
    );
    expect(signature, hasLength(64));
    expect(
      canonicalTransaction(
        transactionId: 'tx-1',
        tenantId: 'tenant-1',
        deviceId: 'device-1',
        userId: 'user-1',
        sequenceNumber: 1,
        operation: 'FindingRecorded',
        payloadHash: 'hash',
      ),
      'tx-1|tenant-1|device-1|user-1|1|FindingRecorded|hash',
    );
  });

  test('v1 envelope canonicalization matches the Go timestamp profile', () {
    expect(
      canonicalTransactionV1(
        transactionId: 'tx-ed',
        tenantId: 'tenant-1',
        organizationId: 'org-1',
        environment: 'LIVE',
        deviceId: 'device-1',
        userId: 'user-1',
        sequenceNumber: 1,
        operation: 'InspectionSubmitted',
        entityId: 'inspection-1',
        payloadHash: 'hash',
        authorityId: 'authority-1',
        authorityEpoch: 4,
        capturedAt: DateTime.utc(2026, 8, 13, 12),
        signatureAlgorithm: 'Ed25519',
        keyId: 'key-1',
      ),
      'v1|tx-ed|tenant-1|org-1|LIVE|device-1|user-1|1|InspectionSubmitted|inspection-1|hash|authority-1|4|2026-08-13T12:00:00Z|Ed25519|key-1',
    );
  });

  test('v1 envelope trims trailing fractional timestamp zeros', () {
    expect(
      canonicalTransactionV1(
        transactionId: 'tx-fraction',
        tenantId: 'tenant-1',
        organizationId: 'org-1',
        environment: 'LIVE',
        deviceId: 'device-1',
        userId: 'user-1',
        sequenceNumber: 2,
        operation: 'FindingRecorded',
        entityId: 'finding-1',
        payloadHash: 'payload-hash',
        authorityId: 'authority-1',
        authorityEpoch: 4,
        capturedAt: DateTime.utc(2026, 8, 13, 12, 0, 0, 120),
        signatureAlgorithm: 'Ed25519',
        keyId: 'key-1',
      ),
      'v1|tx-fraction|tenant-1|org-1|LIVE|device-1|user-1|2|FindingRecorded|finding-1|payload-hash|authority-1|4|2026-08-13T12:00:00.12Z|Ed25519|key-1',
    );
  });
}
