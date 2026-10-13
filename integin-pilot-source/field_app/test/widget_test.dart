// This is a basic Flutter widget test.
//
// To perform an interaction with a widget in your test, use the WidgetTester
// utility in the flutter_test package. For example, you can send tap and scroll
// gestures. You can also use WidgetTester to find child widgets in the widget
// tree, read text, and verify that the values of widget properties are correct.

import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/application/field_app_controller.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/main.dart';

void main() {
  testWidgets('field app launches with offline authority status',
      (WidgetTester tester) async {
    const context = TenantContext(
      tenantId: 'tenant-1',
      organizationId: 'org-1',
      environment: 'LIVE',
    );
    final issued = DateTime.now().toUtc().subtract(const Duration(minutes: 1));
    final authority = OfflineAuthority(
      id: 'authority-1',
      deviceId: 'device-1',
      context: context,
      userId: 'user-1',
      epoch: 1,
      scopes: const ['assigned-work'],
      capabilities: const ['inspection.perform'],
      procedureVersion: 'prepared-work-pack',
      issuedAt: issued,
      expiresAt: issued.add(const Duration(hours: 1)),
      signature: 'signature',
    );
    final controller = FieldAppController(
      context: context,
      deviceId: 'device-1',
      userId: 'user-1',
      deviceState: DeviceTrustState.trusted,
      authority: authority,
    );
    await tester.pumpWidget(INTEGINFieldApp(controller: controller));
    expect(find.text('INTEGIN Field'), findsOneWidget);
  });
}
