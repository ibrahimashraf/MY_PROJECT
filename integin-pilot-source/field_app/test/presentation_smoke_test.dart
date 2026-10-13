import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/presentation/adaptive_scaffold.dart';
import 'package:integin_field_app/presentation/custody_handover_view.dart';
import 'package:integin_field_app/presentation/dynamic_form_view.dart';

void main() {
  testWidgets('AdaptiveScaffold renders mobile layout and navigation', (WidgetTester tester) async {
    await tester.pumpWidget(
      const MaterialApp(
        home: AdaptiveScaffold(
          title: 'INTEGIN Field App',
          drawerItems: [
            NavigationDestination(icon: Icon(Icons.work), label: 'Jobs'),
            NavigationDestination(icon: Icon(Icons.handshake), label: 'Handover'),
          ],
          body: Center(child: Text('Test Content')),
        ),
      ),
    );

    expect(find.text('INTEGIN Field App'), findsOneWidget);
    expect(find.text('Test Content'), findsOneWidget);
    expect(find.text('Jobs'), findsOneWidget);
    expect(find.text('Handover'), findsOneWidget);
  });

  testWidgets('CustodyHandoverView renders form controls and sign button', (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: CustodyHandoverView(
            workOrderId: 'WO-12345',
            assetId: 'ASSET-999',
            onHandoverCompleted: (_) {},
          ),
        ),
      ),
    );

    expect(find.text('Custody & Site Handover'), findsOneWidget);
    expect(find.text('Sign & Seal Handover Transition'), findsOneWidget);
    expect(find.text('Releasing Party Inspector ID / Name *'), findsOneWidget);
    expect(find.text('Accepting Site Representative *'), findsOneWidget);
    expect(find.byType(CheckboxListTile), findsNWidgets(3));
  });

  testWidgets('DynamicFormEngineView renders versioned form fields', (WidgetTester tester) async {
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: SingleChildScrollView(
            child: DynamicFormEngineView(
              formTitle: 'Visual Condition Checklist',
              fields: const [
                DynamicFormFieldDefinition(
                  fieldId: 'checklist_1',
                  label: 'Check Structural Fasteners',
                  type: FieldPrimitiveType.booleanPassFail,
                ),
                DynamicFormFieldDefinition(
                  fieldId: 'torque_value',
                  label: 'Fastener Torque (Nm)',
                  type: FieldPrimitiveType.numericTolerance,
                  minValue: 50.0,
                  maxValue: 120.0,
                  unit: 'Nm',
                ),
              ],
              onSave: (_) {},
            ),
          ),
        ),
      ),
    );

    expect(find.text('Visual Condition Checklist'), findsOneWidget);
    expect(find.text('Check Structural Fasteners'), findsOneWidget);
    expect(find.text('Fastener Torque (Nm) *'), findsOneWidget);
    expect(find.text('Save Form to Canonical Outbox'), findsOneWidget);
  });
}
