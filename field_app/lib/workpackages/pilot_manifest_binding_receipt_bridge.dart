import 'package_manifest_binding_observation.dart';

/// Explicit isolated-pilot destination for source-redacted Field binding facts.
/// A future pilot bootstrap must supply this bridge directly; no default Field
/// startup path constructs it from environment values.
abstract interface class PilotManifestBindingReceiptPublisher {
  Future<void> publishVerifiedCached({
    required String candidateRunId,
    required DateTime occurredAt,
  });
}

/// Converts only a post-cache [PackageManifestBindingObservation] into a
/// pilot-only publication attempt. Publication is advisory evidence: failures
/// are caught locally so successful verify/bind/cache behavior remains intact.
class PilotManifestBindingReceiptBridge
    implements PackageManifestBindingObserver {
  PilotManifestBindingReceiptBridge({
    required String candidateRunId,
    required PilotManifestBindingReceiptPublisher publisher,
  })  : _candidateRunId = _validateRunId(candidateRunId),
        _publisher = publisher;

  final String _candidateRunId;
  final PilotManifestBindingReceiptPublisher _publisher;
  Future<void> _latestAttempt = Future<void>.value();

  /// Allows a guarded pilot test to await advisory publication without making
  /// Field binding wait for it. It has no authority or workflow effect.
  Future<void> get settled => _latestAttempt;

  @override
  void onManifestBinding(PackageManifestBindingObservation observation) {
    if (observation.outcome != PackageManifestBindingOutcome.verifiedCached) {
      return;
    }
    _latestAttempt = _publishIgnoringFailure(observation.occurredAt.toUtc());
  }

  Future<void> _publishIgnoringFailure(DateTime occurredAt) async {
    try {
      await _publisher.publishVerifiedCached(
        candidateRunId: _candidateRunId,
        occurredAt: occurredAt,
      );
    } catch (_) {
      // The binding has already verified and cached successfully. Evidence
      // publication must not downgrade, delete, or otherwise affect it.
    }
  }

  static String _validateRunId(String value) {
    if (!RegExp(r'^[a-f0-9]{32}$').hasMatch(value)) {
      throw ArgumentError.value(
        value,
        'candidateRunId',
        'must be a lower-case 32-hex opaque pilot correlation ID',
      );
    }
    return value;
  }
}
