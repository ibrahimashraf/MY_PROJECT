import 'dart:convert';

import 'package:http/http.dart' as http;

import '../domain/models.dart';
import 'sync_client.dart';

class HttpSyncTransport implements SyncTransport {
  HttpSyncTransport({required this.endpoint, http.Client? client})
      : client = client ?? http.Client();

  final Uri endpoint;
  final http.Client client;

  @override
  Future<SyncResponse> submit(OfflineMutation mutation) async {
    try {
      final response = await client.post(
        endpoint,
        headers: const {'content-type': 'application/json'},
        body: jsonEncode(mutation.toJson()),
      );
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
