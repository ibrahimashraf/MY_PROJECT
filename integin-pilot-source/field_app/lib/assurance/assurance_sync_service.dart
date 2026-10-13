import 'dart:convert';
import 'dart:typed_data';

import 'package:http/http.dart' as http;

import '../security/pinned_http_client.dart';
import '../security/transaction_signer.dart';
import 'canonical_sync_payload.dart';

enum AssuranceSyncStatus {
  accepted,
  rejected,
  quarantined,
  networkError,
}

class AssuranceSyncResult {
  const AssuranceSyncResult({
    required this.status,
    this.statusCode,
    this.message,
  });

  final AssuranceSyncStatus status;
  final int? statusCode;
  final String? message;

  bool get isSuccess => status == AssuranceSyncStatus.accepted;
}

/// Service that serializes, cryptographically signs, and posts assurance evaluations
/// to the backend /api/v1/assurance/sync endpoint.
class AssuranceSyncService {
  AssuranceSyncService({
    required this.endpoint,
    http.Client? client,
  }) : _client = client ?? PinnedHttpClient.forEndpoint(endpoint, allowLoopbackHttp: true);

  final Uri endpoint;
  final http.Client _client;

  /// Cryptographically signs and submits an offline or online evaluation to the backend.
  Future<AssuranceSyncResult> submitSync({
    required String tenantId,
    required String assetId,
    required String signerId,
    required String state,
    required int lamportClock,
    required int revision,
    required DateTime effectiveAt,
    required Uint8List reasonsBytes,
    required Uint8List evidenceChainBytes,
    required DeviceSigner signer,
    bool isOfflineOrigin = true,
    String? bearerToken,
  }) async {
    final effectiveAtSeconds = effectiveAt.toUtc().millisecondsSinceEpoch ~/ 1000;

    // 1. Construct the exact canonical structure required by the Go backend
    final canonical = CanonicalSyncPayload(
      tenantId: tenantId,
      assetId: assetId,
      state: state,
      lamportClock: lamportClock,
      revision: revision,
      effectiveAtSeconds: effectiveAtSeconds,
      reasonsBytes: reasonsBytes,
      evidenceChainBytes: evidenceChainBytes,
    );

    // 2. Sign canonical bytes with the device Ed25519 key
    final signatureBytes = await signer.signRawBytes(canonical.toCanonicalBytes());

    // 3. Assemble HTTP IngestSyncCommand matching Go unmarshaling contract
    final requestBody = <String, dynamic>{
      'tenant_id': tenantId,
      'asset_id': assetId,
      'signer_id': signerId,
      'state': state,
      'lamport_clock': lamportClock,
      'revision': revision,
      'effective_at': effectiveAt.toUtc().toIso8601String(),
      'reasons': base64Encode(reasonsBytes),
      'evidence_chain': base64Encode(evidenceChainBytes),
      'signature': base64Encode(signatureBytes),
      'is_offline_origin': isOfflineOrigin,
    };

    final headers = <String, String>{
      'Content-Type': 'application/json',
      if (bearerToken != null && bearerToken.isNotEmpty)
        'Authorization': 'Bearer $bearerToken',
    };

    final syncUri = endpoint.resolve('/api/v1/assurance/sync');

    try {
      final response = await _client.post(
        syncUri,
        headers: headers,
        body: jsonEncode(requestBody),
      );

      if (response.statusCode >= 200 && response.statusCode < 300) {
        return const AssuranceSyncResult(status: AssuranceSyncStatus.accepted);
      } else if (response.statusCode == 400 || response.statusCode == 403) {
        return AssuranceSyncResult(
          status: AssuranceSyncStatus.rejected,
          statusCode: response.statusCode,
          message: response.body,
        );
      } else {
        return AssuranceSyncResult(
          status: AssuranceSyncStatus.rejected,
          statusCode: response.statusCode,
          message: 'Server error: ${response.statusCode}',
        );
      }
    } catch (e) {
      return AssuranceSyncResult(
        status: AssuranceSyncStatus.networkError,
        message: e.toString(),
      );
    }
  }
}
