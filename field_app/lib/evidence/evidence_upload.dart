import 'dart:convert';

import 'package:http/http.dart' as http;

import 'evidence_crypto.dart';
import 'evidence_store.dart';

enum EvidenceUploadOutcome { applied, duplicate, conflict, rejected, queued }

class EvidenceUploadResult {
  const EvidenceUploadResult({required this.outcome, this.reason});

  final EvidenceUploadOutcome outcome;
  final String? reason;

  bool get retryable => outcome == EvidenceUploadOutcome.queued;
}

abstract interface class EvidenceUploadTransport {
  Future<EvidenceUploadResult> upload({
    required EvidenceScope scope,
    required EncryptedEvidence evidence,
    required String inspectionId,
    required String contentType,
  });
}

class HttpEvidenceUploadTransport implements EvidenceUploadTransport {
  HttpEvidenceUploadTransport({required this.endpoint, http.Client? client}) : client = client ?? http.Client();

  final Uri endpoint;
  final http.Client client;

  @override
  Future<EvidenceUploadResult> upload({
    required EvidenceScope scope,
    required EncryptedEvidence evidence,
    required String inspectionId,
    required String contentType,
  }) async {
    try {
      final response = await client.post(
        endpoint,
        headers: const {'content-type': 'application/json'},
        body: jsonEncode({
          'tenant_id': scope.tenantId,
          'organization_id': scope.organizationId,
          'evidence_id': evidence.evidenceId,
          'inspection_id': inspectionId,
          'content_type': contentType,
          'plaintext_sha256': evidence.plaintextSha256,
          'ciphertext_sha256': evidence.ciphertextSha256,
          'base64_blob': evidence.base64Blob,
        }),
      );
      final outcome = response.statusCode >= 500
          ? EvidenceUploadOutcome.queued
          : response.statusCode == 409
          ? EvidenceUploadOutcome.conflict
          : _parseOutcome(response.body, response.statusCode);
      return EvidenceUploadResult(outcome: outcome, reason: response.body);
    } catch (error) {
      return EvidenceUploadResult(outcome: EvidenceUploadOutcome.queued, reason: 'upload unavailable: $error');
    }
  }

  EvidenceUploadOutcome _parseOutcome(String body, int statusCode) {
    if (statusCode < 200 || statusCode >= 300) return EvidenceUploadOutcome.rejected;
    try {
      final value = (jsonDecode(body) as Map<String, dynamic>)['outcome'] as String?;
      return EvidenceUploadOutcome.values.firstWhere(
        (candidate) => candidate.name.toUpperCase() == value,
        orElse: () => EvidenceUploadOutcome.rejected,
      );
    } catch (_) {
      return EvidenceUploadOutcome.rejected;
    }
  }
}

class RetryingEvidenceUploader {
  const RetryingEvidenceUploader({required this.transport, this.maxAttempts = 3});

  final EvidenceUploadTransport transport;
  final int maxAttempts;

  Future<EvidenceUploadResult> upload({
    required EvidenceScope scope,
    required EncryptedEvidence evidence,
    required String inspectionId,
    required String contentType,
  }) async {
    EvidenceUploadResult? result;
    for (var attempt = 0; attempt < maxAttempts; attempt += 1) {
      result = await transport.upload(
        scope: scope,
        evidence: evidence,
        inspectionId: inspectionId,
        contentType: contentType,
      );
      if (!result.retryable || attempt == maxAttempts - 1) return result;
    }
    return result!;
  }
}
