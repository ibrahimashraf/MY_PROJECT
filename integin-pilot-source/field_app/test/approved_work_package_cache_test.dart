// INTEGIN Field cache tests: prove cached packages remain version/hash-bound and expire safely.
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/storage/persistent_outbox.dart';
import 'package:integin_field_app/workpackages/approved_work_package_cache.dart';

class _MemoryKeyValueStore implements KeyValueStore {
  final Map<String, String> values = {};

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async {
    values[key] = value;
  }
}

CachedApprovedWorkPackage _cachedPackage({DateTime? expiresAt}) {
  final now = DateTime.utc(2026, 8, 17, 9);
  return CachedApprovedWorkPackage(
    workPack: InspectionWorkPack(
      inspectionId: 'inspection-1',
      rootAssetId: 'asset-1',
      inspectionType: 'thorough-inspection',
      procedureVersion: 'v1',
      packageId: 'package-1',
      packageVersion: 1,
      schemaVersion: 1,
      packageHash: 'sha256:${'a' * 64}',
      scheduledDate: now,
      items: const [
        ChecklistItem(
          id: 'item-1',
          sectionId: 'section-1',
          prompt: 'Condition',
          assetId: 'asset-1',
        ),
      ],
    ),
    cachedAt: now,
    expiresAt: expiresAt ?? now.add(const Duration(hours: 12)),
    authorityEpoch: 7,
    signingKeyId: 'key-1',
  );
}

void main() {
  test('round trips a valid approved work package without losing binding',
      () async {
    final cache = ApprovedWorkPackageCache(store: _MemoryKeyValueStore());
    final cached = _cachedPackage();

    await cache.save(cached);
    final loaded =
        await cache.load('package-1', now: DateTime.utc(2026, 8, 17, 10));

    expect(loaded, isNotNull);
    expect(loaded!.workPack.packageVersion, 1);
    expect(loaded.workPack.packageHash, cached.workPack.packageHash);
    expect(loaded.workPack.items.single.id, 'item-1');
  });

  test('does not return an expired package for offline use', () async {
    final cache = ApprovedWorkPackageCache(store: _MemoryKeyValueStore());
    final cached = _cachedPackage(expiresAt: DateTime.utc(2026, 8, 17, 9, 5));

    await cache.save(cached);
    final loaded =
        await cache.load('package-1', now: DateTime.utc(2026, 8, 17, 9, 6));

    expect(loaded, isNull);
  });

  test('rejects package data without a valid deterministic digest', () {
    final invalid = _cachedPackage().workPack;
    final cached = CachedApprovedWorkPackage(
      workPack: InspectionWorkPack(
        inspectionId: invalid.inspectionId,
        rootAssetId: invalid.rootAssetId,
        inspectionType: invalid.inspectionType,
        procedureVersion: invalid.procedureVersion,
        packageId: invalid.packageId,
        packageVersion: invalid.packageVersion,
        schemaVersion: invalid.schemaVersion,
        packageHash: 'not-a-digest',
        scheduledDate: invalid.scheduledDate,
        items: invalid.items,
      ),
      cachedAt: DateTime.utc(2026, 8, 17, 9),
      expiresAt: DateTime.utc(2026, 8, 17, 10),
      authorityEpoch: 7,
      signingKeyId: 'key-1',
    );

    expect(cached.validateForStorage, throwsArgumentError);
  });
}
