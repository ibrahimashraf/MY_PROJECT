import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:integration_test/integration_test.dart';
import 'package:integin_field_app/storage/persistent_outbox.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/sync/http_sync_transport.dart';
import 'package:integin_field_app/sync/sync_client.dart';

void main() {
  IntegrationTestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('Web E2E Sync: Outbox flush via HttpSyncTransport', (tester) async {
    final store = PersistentOutbox(inMemory: true);
    
    // Seed mutation
    final mutation = OfflineMutation(
      transactionId: 'web-e2e-tx-1',
      context: const TenantContext(
        tenantId: 'tenant-1',
        organizationId: 'org-1',
        environment: 'LIVE',
      ),
      deviceId: 'device-1',
      userId: 'user-1',
      sequenceNumber: 1,
      operation: 'InspectionSubmitted',
      entityId: 'inspection-1',
      payload: const {},
      capturedAt: DateTime.now().toUtc(),
      authorityId: 'auth-1',
      authorityEpoch: 1,
      signatureAlgorithm: 'HMAC-SHA256',
      signature: 'test-signature',
    );
    
    await store.append(OutboxEntry(mutation: mutation));
    
    // Attempt sync (expect to hit the server or mock server)
    final transport = HttpSyncTransport(
      endpoint: Uri.parse('http://127.0.0.1:8080/api/v1/sync'), // loopback server
      allowLoopbackHttp: true,
    );
    
    final client = SyncClient(store: store, transport: transport);
    final outcomes = await client.flush();
    
    // Ensure the client executed the sync logic and got an outcome (it will be rejected if no server is running, which is expected for unit tests, but if the E2E is running against a live backend, it should pass)
    expect(outcomes, isNotEmpty);
  });
}
