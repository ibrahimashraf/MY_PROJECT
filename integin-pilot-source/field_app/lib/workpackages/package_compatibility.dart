// INTEGIN Field compatibility policy: preserves historic drafts and blocks only unsafe final work.
import '../domain/inspection_draft.dart';

enum PackageCompatibilityState {
  current,
  updateAvailable,
  needsFormUpdate,
  packageExpired,
  authorityStale,
}

class PackageCompatibilityDecision {
  const PackageCompatibilityDecision({
    required this.state,
    required this.requiredNewItemIds,
    required this.historicDraftRemainsBound,
    required this.allowsAuthoritativeSync,
    required this.blocksFinalCompletion,
  });

  final PackageCompatibilityState state;
  final List<String> requiredNewItemIds;
  final bool historicDraftRemainsBound;
  final bool allowsAuthoritativeSync;
  final bool blocksFinalCompletion;

  String get userMessage {
    switch (state) {
      case PackageCompatibilityState.current:
        return 'Assigned package is current.';
      case PackageCompatibilityState.updateAvailable:
        return 'A newer package is available. This draft remains bound to its original approved package.';
      case PackageCompatibilityState.needsFormUpdate:
        return 'Form update required before final completion: new required fields apply.';
      case PackageCompatibilityState.packageExpired:
        return 'The assigned package is expired. Preserve the draft and refresh the approved package.';
      case PackageCompatibilityState.authorityStale:
        return 'Device authority is stale. Preserve the draft and refresh device authority.';
    }
  }
}

PackageCompatibilityDecision evaluatePackageCompatibility({
  required InspectionWorkPack draftPackage,
  required InspectionWorkPack assignedPackage,
  required DateTime now,
  required DateTime packageExpiresAt,
  required int cachedAuthorityEpoch,
  required int requiredAuthorityEpoch,
}) {
  const historicDraftRemainsBound = true;
  if (!now.toUtc().isBefore(packageExpiresAt.toUtc())) {
    return const PackageCompatibilityDecision(
      state: PackageCompatibilityState.packageExpired,
      requiredNewItemIds: [],
      historicDraftRemainsBound: true,
      allowsAuthoritativeSync: false,
      blocksFinalCompletion: true,
    );
  }
  if (cachedAuthorityEpoch < requiredAuthorityEpoch) {
    return const PackageCompatibilityDecision(
      state: PackageCompatibilityState.authorityStale,
      requiredNewItemIds: [],
      historicDraftRemainsBound: true,
      allowsAuthoritativeSync: false,
      blocksFinalCompletion: true,
    );
  }

  final isExactPackage = draftPackage.packageId == assignedPackage.packageId &&
      draftPackage.packageVersion == assignedPackage.packageVersion &&
      draftPackage.schemaVersion == assignedPackage.schemaVersion &&
      draftPackage.packageHash == assignedPackage.packageHash;
  if (isExactPackage) {
    return const PackageCompatibilityDecision(
      state: PackageCompatibilityState.current,
      requiredNewItemIds: [],
      historicDraftRemainsBound: historicDraftRemainsBound,
      allowsAuthoritativeSync: true,
      blocksFinalCompletion: false,
    );
  }

  final historicItemIds = draftPackage.items.map((item) => item.id).toSet();
  final requiredNewItemIds = assignedPackage.items
      .where((item) => item.required && !historicItemIds.contains(item.id))
      .map((item) => item.id)
      .toList(growable: false);
  if (requiredNewItemIds.isNotEmpty) {
    return PackageCompatibilityDecision(
      state: PackageCompatibilityState.needsFormUpdate,
      requiredNewItemIds: requiredNewItemIds,
      historicDraftRemainsBound: historicDraftRemainsBound,
      allowsAuthoritativeSync: false,
      blocksFinalCompletion: true,
    );
  }
  return const PackageCompatibilityDecision(
    state: PackageCompatibilityState.updateAvailable,
    requiredNewItemIds: [],
    historicDraftRemainsBound: historicDraftRemainsBound,
    allowsAuthoritativeSync: true,
    blocksFinalCompletion: false,
  );
}
