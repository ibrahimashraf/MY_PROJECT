import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:integin_field_app/evidence/evidence_crypto.dart';
import 'package:integin_field_app/evidence/evidence_store.dart';
import 'package:integin_field_app/evidence/evidence_upload.dart';

void main() {
  test('uploads encrypted evidence and maps the Go response', () async {
    final crypto = EvidenceCrypto();
    final evidence = await crypto.encrypt(
      evidenceId: 'photo-1',
      plaintext: [1, 2, 3],
      keyBytes: List<int>.filled(32, 7),
    );
    final client = MockClient((request) async {
      expect(request.url.path, '/evidence');
      expect(request.headers['content-type'], 'application/json');
      return http.Response('{"outcome":"APPLIED"}', 200);
    });
    final result = await HttpEvidenceUploadTransport(
      endpoint: Uri.parse('https://INTEGIN.test/evidence'),
      client: client,
    ).upload(
          scope: const EvidenceScope(tenantId: 'tenant-1', organizationId: 'org-1'),
          evidence: evidence,
          inspectionId: 'inspection-1',
          contentType: 'application/octet-stream',
        );
    expect(result.outcome, EvidenceUploadOutcome.applied);
  });

  test('retries transient queued uploads up to the configured bound', () async {
    var calls = 0;
    final transport = _FakeUploadTransport(() {
      calls += 1;
      return calls < 3
          ? const EvidenceUploadResult(outcome: EvidenceUploadOutcome.queued)
          : const EvidenceUploadResult(outcome: EvidenceUploadOutcome.duplicate);
    });
    final result = await RetryingEvidenceUploader(transport: transport, maxAttempts: 3).upload(
          scope: const EvidenceScope(tenantId: 'tenant-1', organizationId: 'org-1'),
          evidence: const EncryptedEvidence(evidenceId: 'photo-1', sha256: 'digest', base64Blob: 'blob'),
          inspectionId: 'inspection-1',
          contentType: 'application/octet-stream',
        );
    expect(calls, 3);
    expect(result.outcome, EvidenceUploadOutcome.duplicate);
  });

  test('network and server failures are queued, while conflicts are not retryable', () async {
    final network = await HttpEvidenceUploadTransport(
      endpoint: Uri.parse('https://INTEGIN.test/evidence'),
      client: MockClient((_) async => throw StateError('offline')),
    ).upload(
      scope: const EvidenceScope(tenantId: 'tenant-1', organizationId: 'org-1'),
      evidence: const EncryptedEvidence(evidenceId: 'photo-1', sha256: 'digest', base64Blob: 'blob'),
      inspectionId: 'inspection-1',
      contentType: 'application/octet-stream',
    );
    expect(network.outcome, EvidenceUploadOutcome.queued);
    expect(network.retryable, isTrue);

    final server = await HttpEvidenceUploadTransport(
      endpoint: Uri.parse('https://INTEGIN.test/evidence'),
      client: MockClient((_) async => http.Response('unavailable', 503)),
    ).upload(
      scope: const EvidenceScope(tenantId: 'tenant-1', organizationId: 'org-1'),
      evidence: const EncryptedEvidence(evidenceId: 'photo-1', sha256: 'digest', base64Blob: 'blob'),
      inspectionId: 'inspection-1',
      contentType: 'application/octet-stream',
    );
    expect(server.outcome, EvidenceUploadOutcome.queued);

    const conflict = EvidenceUploadResult(outcome: EvidenceUploadOutcome.conflict);
    expect(conflict.retryable, isFalse);
  });
}

class _FakeUploadTransport implements EvidenceUploadTransport {
  _FakeUploadTransport(this.next);

  final EvidenceUploadResult Function() next;

  @override
  Future<EvidenceUploadResult> upload({
    required EvidenceScope scope,
    required EncryptedEvidence evidence,
    required String inspectionId,
    required String contentType,
  }) async => next();
}
