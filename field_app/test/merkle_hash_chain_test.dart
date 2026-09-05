import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/outbox/merkle_hash_chain.dart';

void main() {
  group('CryptographicMerkleHashChain', () {
    test('seals consecutive offline mutations into deterministic tamper-evident chain', () {
      final chain = CryptographicMerkleHashChain();
      final context = const TenantContext(
        tenantId: 'tenant_1',
        organizationId: 'org_1',
        environment: 'LIVE',
      );

      final m1 = OfflineMutation(
        transactionId: 'tx_1',
        context: context,
        deviceId: 'dev_1',
        userId: 'user_1',
        sequenceNumber: 1,
        operation: 'InspectionSubmitted',
        entityId: 'insp_1',
        payload: {'reading': 10.5},
        capturedAt: DateTime.utc(2026, 9, 5, 12, 0, 0),
        authorityId: 'auth_1',
        authorityEpoch: 1,
        signature: 'sig_1',
      );

      final m2 = OfflineMutation(
        transactionId: 'tx_2',
        context: context,
        deviceId: 'dev_1',
        userId: 'user_1',
        sequenceNumber: 2,
        operation: 'InspectionSubmitted',
        entityId: 'insp_2',
        payload: {'reading': 20.5},
        capturedAt: DateTime.utc(2026, 9, 5, 12, 1, 0),
        authorityId: 'auth_1',
        authorityEpoch: 1,
        signature: 'sig_2',
      );

      final h1 = chain.seal(m1);
      final h2 = chain.seal(m2);

      expect(h1, isNotEmpty);
      expect(h2, isNotEmpty);
      expect(h1, isNot(equals(h2)));
      expect(chain.currentHead, equals(h2));
    });
  });
}
