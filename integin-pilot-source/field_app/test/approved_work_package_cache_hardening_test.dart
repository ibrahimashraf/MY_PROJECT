import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/storage/persistent_outbox.dart';
import 'package:integin_field_app/workpackages/approved_work_package_cache.dart';

class _MemoryKeyValueStore implements KeyValueStore {
  final Map<String, String> values = <String, String>{};
  bool rejectWrites = false;

  @override
  Future<String?> read(String key) async => values[key];

  @override
  Future<void> write(String key, String value) async {
    if (rejectWrites) throw StateError('simulated persistence failure');
    values[key] = value;
  }
}

CachedApprovedWorkPackage _cached({
  required String inspectionId,
  required String hashCharacter,
  required DateTime cachedAt,
  DateTime? expiresAt,
}) =>
    CachedApprovedWorkPackage(
      workPack: InspectionWorkPack(
        inspectionId: inspectionId,
        rootAssetId: 'asset-1',
        inspectionType: 'thorough-inspection',
        procedureVersion: 'v1',
        packageId: 'package-1',
        packageVersion: 1,
        schemaVersion: 1,
        packageHash: 'sha256:${hashCharacter * 64}',
        scheduledDate: cachedAt,
        items: const <ChecklistItem>[
          ChecklistItem(
            id: 'item-1',
            sectionId: 'section-1',
            prompt: 'Condition',
            assetId: 'asset-1',
          ),
        ],
      ),
      cachedAt: cachedAt,
      expiresAt: expiresAt ?? cachedAt.add(const Duration(hours: 4)),
      authorityEpoch: 7,
      signingKeyId: 'key-1',
    );

void main() {
  test('retains immutable historic bindings for different inspections',
      () async {
    final cache = ApprovedWorkPackageCache(store: _MemoryKeyValueStore());
    final now = DateTime.utc(2026, 8, 17, 9);
    final first = _cached(
      inspectionId: 'inspection-1',
      hashCharacter: 'a',
      cachedAt: now,
    );
    final second = _cached(
      inspectionId: 'inspection-2',
      hashCharacter: 'b',
      cachedAt: now.add(const Duration(minutes: 1)),
    );
    await cache.save(first);
    await cache.save(second);

    expect((await cache.loadAll(includeExpired: true)).length, 2);
    expect(
      (await cache.loadForInspection('inspection-1', now: now))
          ?.workPack
          .packageHash,
      first.workPack.packageHash,
    );
    expect(
      (await cache.loadForInspection('inspection-2', now: now))
          ?.workPack
          .packageHash,
      second.workPack.packageHash,
    );
  });

  test(
      'reports expiring and expired cached state without treating expired data as usable',
      () {
    final now = DateTime.utc(2026, 8, 17, 9);
    final expiring = _cached(
      inspectionId: 'inspection-1',
      hashCharacter: 'a',
      cachedAt: now,
      expiresAt: now.add(const Duration(minutes: 30)),
    );
    expect(expiring.stateAt(now), ApprovedWorkPackageState.verifiedExpiring);
    expect(expiring.isUsableAt(now), isTrue);
    expect(
      expiring.stateAt(now.add(const Duration(minutes: 31))),
      ApprovedWorkPackageState.expired,
    );
    expect(expiring.isUsableAt(now.add(const Duration(minutes: 31))), isFalse);
  });

  test('write failure preserves the prior serialized cache entry', () async {
    final store = _MemoryKeyValueStore();
    final cache = ApprovedWorkPackageCache(store: store);
    final now = DateTime.utc(2026, 8, 17, 9);
    final first = _cached(
      inspectionId: 'inspection-1',
      hashCharacter: 'a',
      cachedAt: now,
    );
    await cache.save(first);
    store.rejectWrites = true;
    await expectLater(
      cache.save(
        _cached(
          inspectionId: 'inspection-2',
          hashCharacter: 'b',
          cachedAt: now.add(const Duration(minutes: 1)),
        ),
      ),
      throwsStateError,
    );
    store.rejectWrites = false;
    final retained = await cache.loadForInspection('inspection-1', now: now);
    expect(retained?.workPack.packageHash, first.workPack.packageHash);
    expect((await cache.loadAll(includeExpired: true)).length, 1);
  });
}
