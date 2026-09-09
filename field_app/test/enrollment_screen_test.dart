import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:integin_field_app/presentation/enrollment_screen.dart';

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
    return http.Response('not found', 404);
  });
}

Future<void> _pumpScreen(WidgetTester tester, http.Client client) async {
  await tester.pumpWidget(
    MaterialApp(
      home: EnrollmentScreen(
        tenantId: 'tenant-1',
        client: client,
      ),
    ),
  );
}

void main() {
  testWidgets('enrollment reaches success state with SOFTWARE posture',
      (WidgetTester tester) async {
    await _pumpScreen(tester, _fakeServer(failChallenge: false));

    expect(find.text('Device Enrollment'), findsOneWidget);
    expect(find.text(simEnrollmentPosture), findsOneWidget);

    final fields = find.byType(TextField);
    await tester.enterText(fields.at(0), 'inspector-1');
    await tester.enterText(fields.at(1), 'RuggedPad X1');
    await tester.pump();

    await tester.tap(find.text('Enroll device'));
    await tester.pumpAndSettle();

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
    await tester.enterText(fields.at(0), 'inspector-1');
    await tester.enterText(fields.at(1), 'RuggedPad X1');
    await tester.pump();

    await tester.tap(find.text('Enroll device'));
    await tester.pumpAndSettle();

    expect(
      find.textContaining('enrollment challenge failed: HTTP 500'),
      findsOneWidget,
    );
    expect(find.text('Enrolled'), findsNothing);
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });
}