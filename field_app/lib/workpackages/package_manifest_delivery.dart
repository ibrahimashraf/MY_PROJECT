import '../security/transaction_signer.dart';
import 'approved_work_package_cache.dart';
import 'package_manifest_client.dart';
import 'package_manifest_transport.dart';

/// The trusted Field refresh path. It deliberately performs no persistence until
/// a fetched manifest has passed the supplied signature/hash/context verifier.
class PackageManifestDelivery {
  const PackageManifestDelivery({
    required PackageManifestFetcher fetcher,
    required PackageManifestBinder binder,
    required ApprovedWorkPackageCache cache,
  })  : _fetcher = fetcher,
        _binder = binder,
        _cache = cache;

  final PackageManifestFetcher _fetcher;
  final PackageManifestBinder _binder;
  final ApprovedWorkPackageCache _cache;

  Future<CachedApprovedWorkPackage> refreshAndBind({
    required Uri endpoint,
    required DeviceSigner signer,
    required String deviceId,
    required String authorityId,
    required int authorityEpoch,
    required String inspectionId,
    required DateTime now,
  }) async {
    final manifest = await _fetcher.fetch(
      endpoint: endpoint,
      signer: signer,
      deviceId: deviceId,
      authorityId: authorityId,
      authorityEpoch: authorityEpoch,
      inspectionId: inspectionId,
      now: now,
    );
    return _binder.verifyBindAndCache(
      manifest,
      cache: _cache,
      now: now,
    );
  }
}
