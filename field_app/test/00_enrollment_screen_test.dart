import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:integin_field_app/presentation/enrollment_screen.dart';
import 'package:integin_field_app/security/attestation.dart';

/// Fast fake that skips real Ed25519 keygen — avoids slow cryptography
/// under test-runner memory pressure in the full suite.
class _FakeAttestationProvider implements AttestationProvider {
  @override
  Future<AttestationBundle> attestEnrollment({
    required String challengeId,
    required String inspectorId,
    required String nonceHex,
    required String deviceModel,
  }) async => AttestationBundle(
        devicePublicKeyHex: 'aabbccdd' * 8,
        deviceFingerprint: 'simulator-aabbcc',
        signedNonceHex: '00' * 64,
        deviceModel: deviceModel,
        keyOrigin: 'SOFTWARE',
        biometricBound: false,
      );
}

http.Client _fakeServer({required bool failChallenge}) {
  return MockClient((request) async {
    if (request.url.path == '/enroll/challenge') {
      if (failChallenge) {
        return http.Response('boom', 500);
      }
      return http.Response(
        jsonEncode({
          'challenge_id': 'chal-1',
          'nonce': 'abcdef0123',
          'tenant_id': 'tenant-1',
          'inspector_id': 'inspector-1',
        }),
        201,
        headers: {'content-type': 'application/json'},
      );
    }
    if (request.url.path == '/enroll/submit') {
      return http.Response(
        jsonEncode({
          'device_id': 'device-42',
          'tenant_id': 'tenant-1',
          'inspector_id': 'inspector-1',
          'device_public_key': '00aa',
          'device_model': 'RuggedPad X1',
          'is_active': true,
          'attestation_origin': 'SOFTWARE',
          'attestation_biometric_bound': false,
          'attestation_verified': false,
        }),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    if (request.url.path == '/api/v1/devices/enroll') {
      final body = jsonDecode(request.body) as Map<String, dynamic>;
      return http.Response(
        jsonEncode({
          'device_id': body['device_id'] ?? 'device-prod-99',
          'tenant_id': body['tenant_id'],
          'organization_id': body['organization_id'],
          'status': 'PENDING',
          'attestation_origin': 'SOFTWARE',
          'attestation_biometric_bound': false,
          'attestation_verified': false,
        }),
        200,
        headers: {'content-type': 'application/json'},
      );
    }
    return http.Response('not found', 404);
  });
}

Future<void> _pumpScreen(WidgetTester tester, http.Client client, {bool useProduction = false}) async {
  await tester.pumpWidget(
    MaterialApp(
      home: EnrollmentScreen(
        tenantId: 'tenant-1',
        organizationId: 'org-1',
        endpoint:
            useProduction ? 'https://127.0.0.1:18080' : enrollLoopbackEndpoint,
        useProductionRoute: useProduction,
        client: client,
        attestationProvider: _FakeAttestationProvider(),
      ),
    ),
  );
}

void main() {
  // pumpAndSettle() can return while the enroll future chain is still
  // in-flight (no frame scheduled yet across an await boundary), leaving
  // the progress indicator on screen and failing the assertions below.
  // Wait for the terminal state (indicator gone, no pending frames) instead;
  // no assertion is changed.
  Future<void> pumpEnrollResult(WidgetTester tester) async {
    for (var i = 0; i < 20; i += 1) {
      await tester.pump(const Duration(milliseconds: 100));
      if (find.byType(CircularProgressIndicator).evaluate().isEmpty &&
          tester.binding.transientCallbackCount == 0) {
        break;
      }
    }
  }

  testWidgets('enrollment reaches success state with SOFTWARE posture',
      (WidgetTester tester) async {
    await _pumpScreen(tester, _fakeServer(failChallenge: false));

    expect(find.text('Device Enrollment'), findsOneWidget);
    expect(find.text(simEnrollmentPosture), findsOneWidget);

    final fields = find.byType(TextField);
    await tester.enterText(fields.at(0), 'org-1');
    await tester.enterText(fields.at(1), 'inspector-1');
    await tester.enterText(fields.at(2), 'RuggedPad X1');
    await tester.pump();

    final enrollBtn = find.text('Enroll device');
    await tester.ensureVisible(enrollBtn);
    // Settle the scroll animation first: tapping mid-scroll misses the
    // off-screen button (non-fatal warning) and the test then fails at the
    // assertions with the form still idle. No animation persists here.
    await tester.pumpAndSettle();
    await tester.tap(enrollBtn);
    await pumpEnrollResult(tester);

    expect(find.text('Device ID: device-42'), findsOneWidget);
    expect(find.text('Attestation origin: SOFTWARE'), findsOneWidget);
    expect(find.textContaining('Attestation verified: false'), findsOneWidget);
    expect(find.textContaining('never yields a verified'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });

  testWidgets('enrollment surfaces server failure as an error state',
      (WidgetTester tester) async {
    await _pumpScreen(tester, _fakeServer(failChallenge: true));

    final fields = find.byType(TextField);
    await tester.enterText(fields.at(0), 'org-1');
    await tester.enterText(fields.at(1), 'inspector-1');
    await tester.enterText(fields.at(2), 'RuggedPad X1');
    await tester.pump();

    final enrollBtn = find.text('Enroll device');
    await tester.ensureVisible(enrollBtn);
    // Settle the scroll animation first: tapping mid-scroll misses the
    // off-screen button (non-fatal warning) and the test then fails at the
    // assertions with the form still idle. No animation persists here.
    await tester.pumpAndSettle();
    await tester.tap(enrollBtn);
    await pumpEnrollResult(tester);

    expect(
      find.textContaining('enrollment challenge failed: HTTP 500'),
      findsOneWidget,
    );
    expect(find.text('Enrolled'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });

  testWidgets('enrollment successfully executes via production route (/api/v1/devices/enroll)',
      (WidgetTester tester) async {
    await _pumpScreen(tester, _fakeServer(failChallenge: false), useProduction: true);

    final fields = find.byType(TextField);
    await tester.enterText(fields.at(0), 'org-1');
    await tester.enterText(fields.at(1), 'inspector-1');
    await tester.enterText(fields.at(2), 'RuggedPad X1');
    await tester.pump();

    final enrollBtn = find.text('Enroll device');
    await tester.ensureVisible(enrollBtn);
    // Settle the scroll animation first: tapping mid-scroll misses the
    // off-screen button (non-fatal warning) and the test then fails at the
    // assertions with the form still idle. No animation persists here.
    await tester.pumpAndSettle();
    await tester.tap(enrollBtn);
    await pumpEnrollResult(tester);

    expect(find.text('Enrolled'), findsOneWidget);
    expect(find.text('Attestation origin: SOFTWARE'), findsOneWidget);
  });
}