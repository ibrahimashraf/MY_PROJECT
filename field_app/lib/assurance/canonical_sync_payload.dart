import 'dart:convert';
import 'dart:typed_data';

/// The exact canonical payload structure that the Flutter Field App signs
/// before dispatching to the Go backend's /api/v1/assurance/sync endpoint.
///
/// Matches the Go backend [CanonicalSyncPayload] struct bit-for-bit:
/// - tenant_id (UUID string)
/// - asset_id (UUID string)
/// - state (String enum)
/// - lamport_clock (int)
/// - revision (int)
/// - effective_at (int64 seconds)
/// - reasons (base64 encoded bytes)
/// - evidence_chain (base64 encoded bytes)
class CanonicalSyncPayload {
  const CanonicalSyncPayload({
    required this.tenantId,
    required this.assetId,
    required this.state,
    required this.lamportClock,
    required this.revision,
    required this.effectiveAtSeconds,
    required this.reasonsBytes,
    required this.evidenceChainBytes,
  });

  final String tenantId;
  final String assetId;
  final String state;
  final int lamportClock;
  final int revision;
  final int effectiveAtSeconds;
  final Uint8List reasonsBytes;
  final Uint8List evidenceChainBytes;

  /// Serializes into canonical JSON map with keys in exact matching order.
  Map<String, dynamic> toJson() {
    return <String, dynamic>{
      'tenant_id': tenantId,
      'asset_id': assetId,
      'state': state,
      'lamport_clock': lamportClock,
      'revision': revision,
      'effective_at': effectiveAtSeconds,
      'reasons': base64Encode(reasonsBytes),
      'evidence_chain': base64Encode(evidenceChainBytes),
    };
  }

  /// Encodes to UTF-8 bytes of canonical JSON for Ed25519 signing.
  Uint8List toCanonicalBytes() {
    return Uint8List.fromList(utf8.encode(jsonEncode(toJson())));
  }
}
