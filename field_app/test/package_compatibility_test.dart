// INTEGIN Field policy tests: updates never silently change a historic package-bound draft.
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/workpackages/package_compatibility.dart';

InspectionWorkPack _package({
  required int version,
  required String hash,
  List<ChecklistItem>? items,
}) =>
    InspectionWorkPack(
      inspectionId: 'inspection-1',
      rootAssetId: 'asset-1',
      inspectionType: 'thorough-inspection',
      procedureVersion: 'v$version',
      packageId: 'package-1',
      packageVersion: version,
      schemaVersion: version,
      packageHash: hash,
      scheduledDate: DateTime.utc(2026, 8, 17),
      items: items ??
          const [
            ChecklistItem(
              id: 'condition',
              sectionId: 'main',
              prompt: 'Condition',
              assetId: 'asset-1',
            ),
          ],
    );

void main() {
  final now = DateTime.utc(2026, 8, 17, 9);
  final source = _package(version: 1, hash: 'sha256:${'a' * 64}');

  test('keeps the exact assigned package current', () {
    final decision = evaluatePackageCompatibility(
      draftPackage: source,
      assignedPackage: source,
      now: now,
      packageExpiresAt: now.add(const Duration(hours: 1)),
      cachedAuthorityEpoch: 7,
      requiredAuthorityEpoch: 7,
    );

    expect(decision.state, PackageCompatibilityState.current);
    expect(decision.allowsAuthoritativeSync, isTrue);
    expect(decision.blocksFinalCompletion, isFalse);
  });

  test('permits original-draft sync when newer package adds no required fields',
      () {
    final newer = _package(
      version: 2,
      hash: 'sha256:${'b' * 64}',
      items: const [
        ChecklistItem(
            id: 'condition',
            sectionId: 'main',
            prompt: 'Condition',
            assetId: 'asset-1'),
        ChecklistItem(
            id: 'optional-note',
            sectionId: 'main',
            prompt: 'Optional note',
            assetId: 'asset-1',
            required: false),
      ],
    );
    final decision = evaluatePackageCompatibility(
      draftPackage: source,
      assignedPackage: newer,
      now: now,
      packageExpiresAt: now.add(const Duration(hours: 1)),
      cachedAuthorityEpoch: 7,
      requiredAuthorityEpoch: 7,
    );

    expect(decision.state, PackageCompatibilityState.updateAvailable);
    expect(decision.historicDraftRemainsBound, isTrue);
    expect(decision.allowsAuthoritativeSync, isTrue);
  });

  test('requires an explicit form update for a new required field', () {
    final newer = _package(
      version: 2,
      hash: 'sha256:${'b' * 64}',
      items: const [
        ChecklistItem(
            id: 'condition',
            sectionId: 'main',
            prompt: 'Condition',
            assetId: 'asset-1'),
        ChecklistItem(
            id: 'capacity',
            sectionId: 'main',
            prompt: 'Capacity',
            assetId: 'asset-1'),
      ],
    );
    final decision = evaluatePackageCompatibility(
      draftPackage: source,
      assignedPackage: newer,
      now: now,
      packageExpiresAt: now.add(const Duration(hours: 1)),
      cachedAuthorityEpoch: 7,
      requiredAuthorityEpoch: 7,
    );

    expect(decision.state, PackageCompatibilityState.needsFormUpdate);
    expect(decision.requiredNewItemIds, ['capacity']);
    expect(decision.allowsAuthoritativeSync, isFalse);
  });

  test(
      'preserves the draft but blocks final work for expired package or stale authority',
      () {
    final expired = evaluatePackageCompatibility(
      draftPackage: source,
      assignedPackage: source,
      now: now,
      packageExpiresAt: now,
      cachedAuthorityEpoch: 7,
      requiredAuthorityEpoch: 7,
    );
    final staleAuthority = evaluatePackageCompatibility(
      draftPackage: source,
      assignedPackage: source,
      now: now,
      packageExpiresAt: now.add(const Duration(hours: 1)),
      cachedAuthorityEpoch: 6,
      requiredAuthorityEpoch: 7,
    );

    expect(expired.state, PackageCompatibilityState.packageExpired);
    expect(staleAuthority.state, PackageCompatibilityState.authorityStale);
    expect(expired.historicDraftRemainsBound, isTrue);
    expect(staleAuthority.blocksFinalCompletion, isTrue);
  });
}
