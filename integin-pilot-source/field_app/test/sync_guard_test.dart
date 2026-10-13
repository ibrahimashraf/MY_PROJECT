import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/sync/sync_guard.dart';

void main() {
  const context = TenantContext(
    tenantId: 'tenant-1',
    organizationId: 'org-1',
    environment: 'LIVE',
  );

  OfflineAuthority authority() {
    final issued = DateTime.utc(2026, 8, 13, 10);
    return OfflineAuthority(
      id: 'authority-1',
      deviceId: 'device-1',
      context: context,
      userId: 'user-1',
      epoch: 4,
      scopes: const ['site-1'],
      capabilities: const ['inspection.perform'],
      procedureVersion: 'proc-1',
      issuedAt: issued,
      expiresAt: issued.add(const Duration(hours: 4)),
      signature: 'server-signature',
    );
  }

  OfflineMutation mutation({int sequence = 1, String tenant = 'tenant-1', String environment = 'LIVE'}) {
    final payload = <String, Object?>{'inspection_id': 'inspection-1'};
    final payloadHash = OfflineMutation(
      transactionId: 'hash-only',
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      sequenceNumber: sequence,
      operation: 'FindingRecorded',
      entityId: 'inspection-1',
      payload: payload,
      capturedAt: DateTime.utc(2026, 8, 13, 10, 1),
      authorityId: 'authority-1',
      authorityEpoch: 4,
      signature: 'placeholder',
    ).payloadHash;
    return OfflineMutation(
      transactionId: 'tx-$sequence',
      context: TenantContext(tenantId: tenant, organizationId: 'org-1', environment: environment),
      deviceId: 'device-1',
      userId: 'user-1',
      sequenceNumber: sequence,
      operation: 'FindingRecorded',
      entityId: 'inspection-1',
      payload: payload,
      capturedAt: DateTime.utc(2026, 8, 13, 10, 1),
      authorityId: 'authority-1',
      authorityEpoch: 4,
      signature: signTransaction(
        secret: 'secret',
        transactionId: 'tx-$sequence',
        tenantId: tenant,
        deviceId: 'device-1',
        userId: 'user-1',
        sequenceNumber: sequence,
        operation: 'FindingRecorded',
        payloadHash: payloadHash,
      ),
    );
  }

  test('allows current tenant, authority, identity, and contiguous sequence', () {
    final guard = SyncGuard(
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.trusted,
      authority: authority(),
      lastSequence: 0,
    );
    expect(guard.check(mutation(), DateTime.utc(2026, 8, 13, 11)).allowed, isTrue);
  });

  test('rejects tenant mismatch, stale sequence, and non-trusted device', () {
    final base = SyncGuard(
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.trusted,
      authority: authority(),
      lastSequence: 0,
    );
    expect(base.check(mutation(tenant: 'tenant-2'), DateTime.utc(2026, 8, 13, 11)).allowed, isFalse);
    expect(base.check(mutation(sequence: 2), DateTime.utc(2026, 8, 13, 11)).allowed, isFalse);
    expect(base.check(mutation(environment: 'TESTING'), DateTime.utc(2026, 8, 13, 11)).allowed, isFalse);

    final revoked = SyncGuard(
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.revoked,
      authority: authority(),
      lastSequence: 0,
    );
    expect(revoked.check(mutation(), DateTime.utc(2026, 8, 13, 11)).allowed, isFalse);
  });

  test('rejects expired authority, identity mismatch, stale authority, and incomplete payload', () {
    final base = SyncGuard(
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.trusted,
      authority: authority(),
      lastSequence: 0,
    );
    expect(base.check(mutation(), DateTime.utc(2026, 8, 13, 14)).reason, contains('expired'));
    final wrongIdentity = OfflineMutation.fromJson({...mutation().toJson(), 'user_id': 'user-2'});
    expect(base.check(wrongIdentity, DateTime.utc(2026, 8, 13, 11)).reason, contains('identity'));
    final wrongEpoch = OfflineMutation.fromJson({...mutation().toJson(), 'authority_epoch': 3});
    expect(base.check(wrongEpoch, DateTime.utc(2026, 8, 13, 11)).reason, contains('authority'));
    final incomplete = OfflineMutation.fromJson({...mutation().toJson(), 'signature': ''});
    expect(base.check(incomplete, DateTime.utc(2026, 8, 13, 11)).reason, contains('incomplete'));
  });
}
