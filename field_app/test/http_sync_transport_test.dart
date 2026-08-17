import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/sync/http_sync_transport.dart';

void main() {
  test('normalizes an Applied response from the Go sync route', () async {
    final client = MockClient((request) async {
      expect(request.url.path, '/sync');
      expect(request.headers['content-type'], 'application/json');
      return http.Response(
        '{"outcome":"APPLIED","reason":"transaction applied"}',
        200,
      );
    });
    final response = await HttpSyncTransport(
      endpoint: Uri.parse('https://integin.test/sync'),
      client: client,
    ).submit(_mutation());

    expect(response.outcome, SyncOutcome.applied);
    expect(response.reason, 'transaction applied');
  });

  test('turns an unavailable endpoint into an explicit rejected outcome', () async {
    final response = await HttpSyncTransport(
      endpoint: Uri.parse('https://integin.test/sync'),
      client: MockClient((_) async => http.Response('unavailable', 503)),
    ).submit(_mutation());

    expect(response.outcome, SyncOutcome.rejected);
    expect(response.reason, contains('503'));
  });

  test('turns a network exception into a retryable queued outcome', () async {
    final response = await HttpSyncTransport(
      endpoint: Uri.parse('https://integin.test/sync'),
      client: MockClient((_) async => throw StateError('offline')),
    ).submit(_mutation());

    expect(response.outcome, SyncOutcome.queued);
    expect(response.reason, contains('transport unavailable'));
  });
}

OfflineMutation _mutation() {
  const context = TenantContext(
    tenantId: 'tenant-1',
    organizationId: 'org-1',
    environment: 'LIVE',
  );
  return OfflineMutation(
    transactionId: 'tx-1',
    context: context,
    deviceId: 'device-1',
    userId: 'user-1',
    sequenceNumber: 1,
    operation: 'InspectionSubmitted',
    entityId: 'inspection-1',
    payload: const {'inspection_id': 'inspection-1'},
    capturedAt: DateTime.utc(2026, 8, 13, 10),
    authorityId: 'authority-1',
    authorityEpoch: 1,
    signature: 'signature',
  );
}
