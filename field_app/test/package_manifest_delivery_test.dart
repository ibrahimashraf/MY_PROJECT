import 'package:cryptography/cryptography.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/security/transaction_signer.dart';
import 'package:integin_field_app/storage/persistent_outbox.dart';
import 'package:integin_field_app/workpackages/approved_work_package_cache.dart';
import 'package:integin_field_app/workpackages/package_manifest_client.dart';
import 'package:integin_field_app/workpackages/package_manifest_delivery.dart';
import 'package:integin_field_app/workpackages/package_manifest_transport.dart';
import 'package:integin_field_app/workpackages/pilot_manifest_binding_receipt_bridge.dart';

class _MemoryStore implements KeyValueStore {
  final Map<String, String> values = <String, String>{};
  @override
  Future<String?> read(String key) async => values[key];
  @override
  Future<void> write(String key, String value) async => values[key] = value;
}

class _Fetcher implements PackageManifestFetcher {
  var called = false;
  @override
  Future<SignedPackageManifest> fetch({
    required Uri endpoint,
    required DeviceSigner signer,
    required String deviceId,
    required String authorityId,
    required int authorityEpoch,
    required String inspectionId,
    required DateTime now,
  }) async {
    called = true;
    return SignedPackageManifest(<String, dynamic>{});
  }
}

class _Binder implements PackageManifestBinder {
  var called = false;
  @override
  Future<CachedApprovedWorkPackage> verifyBindAndCache(
    SignedPackageManifest manifest, {
    required ApprovedWorkPackageCache cache,
    required DateTime now,
  }) async {
    called = true;
    final bound = CachedApprovedWorkPackage(
      workPack: InspectionWorkPack(
        inspectionId: 'inspection-1',
        rootAssetId: 'asset-1',
        inspectionType: 'inspection',
        procedureVersion: 'v1',
        packageId: 'package-1',
        packageVersion: 1,
        schemaVersion: 1,
        packageHash: 'sha256:${'a' * 64}',
        scheduledDate: now,
        items: const <ChecklistItem>[],
      ),
      cachedAt: now,
      expiresAt: now.add(const Duration(hours: 1)),
      authorityEpoch: 1,
      signingKeyId: 'key-1',
    );
    await cache.save(bound);
    return bound;
  }
}

class _PilotPublisher implements PilotManifestBindingReceiptPublisher {
  _PilotPublisher({this.throwOnPublish = false});

  final bool throwOnPublish;
  var publishCalls = 0;
  String? candidateRunId;
  DateTime? occurredAt;

  @override
  Future<void> publishVerifiedCached({
    required String candidateRunId,
    required DateTime occurredAt,
  }) async {
    publishCalls++;
    this.candidateRunId = candidateRunId;
    this.occurredAt = occurredAt;
    if (throwOnPublish) {
      throw StateError('synthetic advisory publication failure');
    }
  }
}

void main() {
  test('delivery refresh fetches then delegates trusted verify/cache binding',
      () async {
    final fetcher = _Fetcher();
    final binder = _Binder();
    final cache = ApprovedWorkPackageCache(store: _MemoryStore());
    final delivery = PackageManifestDelivery(
      fetcher: fetcher,
      binder: binder,
      cache: cache,
    );
    final signer = DeviceSigner(await Ed25519().newKeyPair(), keyId: 'key-1');
    final now = DateTime.utc(2026, 8, 17, 12);

    final result = await delivery.refreshAndBind(
      endpoint: Uri.parse('http://127.0.0.1:18080/work-package-manifest'),
      signer: signer,
      deviceId: 'device-1',
      authorityId: 'authority-1',
      authorityEpoch: 1,
      inspectionId: 'inspection-1',
      now: now,
    );

    expect(fetcher.called, isTrue);
    expect(binder.called, isTrue);
    expect(result.workPack.inspectionId, 'inspection-1');
    expect(
      (await cache.loadForInspection('inspection-1', now: now))
          ?.workPack
          .packageHash,
      result.workPack.packageHash,
    );
  });

  test('pilot receipt bridge observes verified cache without blocking it',
      () async {
    final fetcher = _Fetcher();
    final binder = _Binder();
    final cache = ApprovedWorkPackageCache(store: _MemoryStore());
    final publisher = _PilotPublisher(throwOnPublish: true);
    final bridge = PilotManifestBindingReceiptBridge(
      candidateRunId: '0123456789abcdef0123456789abcdef',
      publisher: publisher,
    );
    final delivery = PackageManifestDelivery(
      fetcher: fetcher,
      binder: binder,
      cache: cache,
      observer: bridge,
    );
    final signer = DeviceSigner(await Ed25519().newKeyPair(), keyId: 'key-1');
    final now = DateTime.utc(2026, 8, 19, 12);

    final result = await delivery.refreshAndBind(
      endpoint: Uri.parse('http://127.0.0.1:18080/work-package-manifest'),
      signer: signer,
      deviceId: 'device-1',
      authorityId: 'authority-1',
      authorityEpoch: 1,
      inspectionId: 'inspection-1',
      now: now,
    );
    await bridge.settled;

    expect(result.workPack.inspectionId, 'inspection-1');
    expect(
      (await cache.loadForInspection('inspection-1', now: now))
          ?.workPack
          .inspectionId,
      result.workPack.inspectionId,
    );
    expect(publisher.publishCalls, 1);
    expect(publisher.candidateRunId, '0123456789abcdef0123456789abcdef');
    expect(publisher.occurredAt, now);
  });
}
