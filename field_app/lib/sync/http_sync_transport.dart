import 'dart:convert';

import 'package:http/http.dart' as http;

import '../domain/models.dart';
import '../security/endpoint_guard.dart';
import '../security/pinned_http_client.dart';
import 'sync_client.dart';

class HttpSyncTransport implements SyncTransport {
  HttpSyncTransport({
    required this.endpoint,
    http.Client? client,
    bool allowLoopbackHttp = false,
  })  : client = client ?? PinnedHttpClient.forEndpoint(endpoint, allowLoopbackHttp: allowLoopbackHttp) {
    assertEndpointSafe(endpoint, allowLoopbackHttp: allowLoopbackHttp);
  }

  final Uri endpoint;
  final http.Client client;

  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async {
    try {
      final response = await client.post(
        endpoint,
        headers: {
          'content-type': 'application/json',
          // Stripe-style dedupe: the transaction id is stable across retries,
          // so a retried flush replays instead of double-applying. The server
          // hard-requires this header (400 idempotency_key_required without it).
          'Idempotency-Key': mutation.transactionId,
        },
        body: jsonEncode(mutation.toJson()),
      ).timeout(const Duration(seconds: 15));
      if (response.statusCode < 200 || response.statusCode >= 300) {
        return SyncResponse(
          outcome: SyncOutcome.rejected,
          reason: 'sync endpoint returned HTTP ${response.statusCode}',
        );
      }
      final payload = jsonDecode(response.body) as Map<String, dynamic>;
      return SyncResponse(
        outcome: _outcome(payload['outcome'] as String?),
        reason: payload['reason'] as String?,
        expectedSequence: payload['expected_sequence'] as int?,
      );
    } catch (error) {
      return SyncResponse(
        outcome: SyncOutcome.queued,
        reason: 'sync transport unavailable: $error',
      );
    }
  }

  SyncOutcome _outcome(String? value) => SyncOutcome.values.firstWhere(
        (candidate) => candidate.name.toUpperCase() == value,
        orElse: () => SyncOutcome.rejected,
      );
}
