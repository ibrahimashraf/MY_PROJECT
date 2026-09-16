import 'dart:convert';

import 'package:http/http.dart' as http;

import '../security/endpoint_guard.dart';
import '../security/pinned_http_client.dart';

/// A constrained pilot-only reader for the existing advisory service.
class PilotAdvisoryResult {
  const PilotAdvisoryResult({
    required this.title,
    required this.summary,
    required this.confidence,
    required this.blocking,
    required this.limitations,
  });

  final String title;
  final String summary;
  final double confidence;
  final bool blocking;
  final List<String> limitations;

  factory PilotAdvisoryResult.fromJson(Map<String, dynamic> json) {
    final blocking = json['blocking'];
    if (blocking is! bool || blocking) {
      throw StateError('Pilot advisory response must remain non-blocking.');
    }
    final title = json['title'];
    final summary = json['summary'];
    final confidence = json['confidence'];
    final limitations = json['limitations'];
    if (title is! String ||
        summary is! String ||
        confidence is! num ||
        limitations is! List) {
      throw StateError('Pilot advisory response is malformed.');
    }
    return PilotAdvisoryResult(
      title: title,
      summary: summary,
      confidence: confidence.toDouble(),
      blocking: blocking,
      limitations: limitations.whereType<String>().toList(growable: false),
    );
  }
}

/// Sends only a monitoring request; it cannot approve or mutate field work.
class PilotAdvisoryClient {
  PilotAdvisoryClient({required this.endpoint, http.Client? client})
      : _client = client ??
            PinnedHttpClient.forEndpoint(endpoint,
                allowLoopbackHttp: true) {
    assertEndpointSafe(endpoint, allowLoopbackHttp: true);
  }

  final Uri endpoint;
  final http.Client _client;

  Future<PilotAdvisoryResult> requestMonitoring({
    required String tenantId,
    required String inspectionId,
    required List<String> evidenceRefs,
  }) async {
    final response = await _client.post(
      endpoint,
      headers: const {'content-type': 'application/json'},
      body: jsonEncode({
        'version': 'v1',
        'tenant_id': tenantId,
        'zone': 'MONITORING',
        'lens': 'ASSET_INTEGRITY',
        'inputs': {
          'inspection_id': inspectionId,
          'source': 'integin-field-pilot'
        },
        'evidence_refs': evidenceRefs,
      }),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw StateError(
          'Pilot advisory service returned an unavailable result.');
    }
    return PilotAdvisoryResult.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }
}
