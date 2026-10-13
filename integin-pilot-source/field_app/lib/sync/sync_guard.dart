import '../domain/models.dart';

class SyncGuardResult {
  const SyncGuardResult.allowed() : allowed = true, reason = null;
  const SyncGuardResult.denied(this.reason) : allowed = false;

  final bool allowed;
  final String? reason;
}

class SyncGuard {
  const SyncGuard({required this.context, required this.deviceId, required this.userId, required this.deviceState, required this.authority, required this.lastSequence});

  final TenantContext context;
  final String deviceId;
  final String userId;
  final DeviceTrustState deviceState;
  final OfflineAuthority authority;
  final int lastSequence;

  SyncGuardResult check(OfflineMutation mutation, DateTime at) {
    if (!context.isValid || mutation.context.tenantId != context.tenantId || mutation.context.organizationId != context.organizationId || mutation.context.environment != context.environment) {
      return const SyncGuardResult.denied('tenant or organization mismatch');
    }
    if (mutation.deviceId != deviceId || mutation.userId != userId) {
      return const SyncGuardResult.denied('transaction identity mismatch');
    }
    if (deviceState != DeviceTrustState.trusted || mutation.authorityId != authority.id || mutation.authorityEpoch != authority.epoch) {
      return const SyncGuardResult.denied('device authority is not current');
    }
    if (!authority.isValidAt(at)) {
      return const SyncGuardResult.denied('offline authority is expired or not yet valid');
    }
    if (mutation.sequenceNumber != lastSequence + 1) {
      return const SyncGuardResult.denied('sequence is not contiguous');
    }
    if (mutation.payloadHash.isEmpty || mutation.signature.isEmpty) {
      return const SyncGuardResult.denied('signed payload is incomplete');
    }
    return const SyncGuardResult.allowed();
  }
}
