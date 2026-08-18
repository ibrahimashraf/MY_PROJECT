// INTEGIN Field manifest binding: source-redacted post-bind evidence only.

/// The only successful binding state suitable for later non-secret candidate
/// evidence. It is emitted only after a manifest has been verified and cached.
enum PackageManifestBindingOutcome { verifiedCached }

/// Source-redacted Field evidence produced after trusted manifest binding.
///
/// This type intentionally has no manifest, tenant, organization, device, user,
/// key, signature, proof, endpoint, or package fields. A future isolated-pilot
/// receipt bridge can correlate this public signal through a separately
/// provisioned opaque run identifier without changing Field workflow authority.
class PackageManifestBindingObservation {
  const PackageManifestBindingObservation.verifiedCached({
    required this.occurredAt,
  })  : outcome = PackageManifestBindingOutcome.verifiedCached,
        reasonCode = 'verified_cached';

  final PackageManifestBindingOutcome outcome;
  final String reasonCode;
  final DateTime occurredAt;
}

/// Receives source-redacted manifest-binding observations. Implementations are
/// advisory evidence sinks only and must not change Field workflow state.
abstract interface class PackageManifestBindingObserver {
  void onManifestBinding(PackageManifestBindingObservation observation);
}
