import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/assurance/rule_bundle.dart';
import 'package:integin_field_app/assurance/rule_bundle_store.dart';
import 'package:integin_field_app/assurance/rule_evaluator.dart';
import 'package:integin_field_app/presentation/compliance_result_card.dart';
import 'package:integin_field_app/presentation/dynamic_form_view.dart';
import 'package:integin_field_app/storage/persistent_outbox.dart';

class _FakeMemoryKeyValueStore implements KeyValueStore {
  final Map<String, String> _data = {};

  @override
  Future<String?> read(String key) async => _data[key];

  @override
  Future<void> write(String key, String value) async {
    _data[key] = value;
  }
}

void main() {
  group('ComplianceResultCard', () {
    testWidgets('renders Assured state correctly', (tester) async {
      const outcome = EvaluationOutcome(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.0',
        ruleHash: 'hash-assured',
        results: [],
        highestSeverity: AssuranceSeverity.info,
        isNonCompliant: false,
        isCondemned: false,
        reasons: [],
        state: 'ASSURED',
      );

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: ComplianceResultCard(outcome: outcome),
          ),
        ),
      );

      expect(find.text('Compliance: Assured'), findsOneWidget);
      expect(find.text('Highest severity: info'), findsOneWidget);
      expect(find.byIcon(Icons.check_circle), findsOneWidget);
    });

    testWidgets('renders Condemned state with reasons correctly', (tester) async {
      const outcome = EvaluationOutcome(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.0',
        ruleHash: 'hash-condemned',
        results: [],
        highestSeverity: AssuranceSeverity.critical,
        isNonCompliant: true,
        isCondemned: true,
        reasons: ['Diameter fell below threshold'],
        state: 'CONDEMNED',
      );

      await tester.pumpWidget(
        const MaterialApp(
          home: Scaffold(
            body: ComplianceResultCard(outcome: outcome),
          ),
        ),
      );

      expect(find.text('Compliance: Condemned'), findsOneWidget);
      expect(find.text('Highest severity: critical'), findsOneWidget);
      expect(find.text('Reasons: Diameter fell below threshold'), findsOneWidget);
      expect(find.byIcon(Icons.cancel), findsOneWidget);
    });
  });

  group('DynamicFormEngineView with Compliance Evaluation', () {
    testWidgets('evaluates and displays ComplianceResultCard on Save', (tester) async {
      final storage = _FakeMemoryKeyValueStore();
      final store = RuleBundleStore(storage: storage);

      // Seed a rule bundle that condemns if measured < 50
      const bundle = RuleBundle(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.0',
        ruleHash: 'hash-test',
        rules: [
          DecisionRule(
            ruleId: 'THROAT_OPENING_CHECK',
            expression: 'throat_opening_mm',
            evaluations: [
              EvaluationBranch(
                operator: RuleOperator.lt,
                threshold: 50.0,
                severity: AssuranceSeverity.critical,
                reason: 'Throat opening under minimum tolerance',
                isBlocking: true,
              ),
            ],
          ),
        ],
      );
      await store.saveBundle(bundle);

      Map<String, dynamic>? savedValues;

      await tester.pumpWidget(
        MaterialApp(
          home: Scaffold(
            body: SingleChildScrollView(
              child: DynamicFormEngineView(
                formTitle: 'Lifting Hook Inspection',
                ruleBundleStore: store,
                fields: const [
                  DynamicFormFieldDefinition(
                    fieldId: 'throat_opening_mm',
                    label: 'Hook Throat Opening (mm)',
                    type: FieldPrimitiveType.numericTolerance,
                  ),
                ],
                onSave: (values) {
                  savedValues = values;
                },
              ),
            ),
          ),
        ),
      );

      // ComplianceResultCard should not be visible before saving
      expect(find.byType(ComplianceResultCard), findsNothing);

      // Enter value 42.0 (triggers critical branch < 50.0)
      await tester.enterText(find.byType(TextField), '42.0');
      await tester.pump();

      // Tap Save
      await tester.tap(find.text('Save Form to Canonical Outbox'));
      await tester.pump();
      await tester.pump(const Duration(milliseconds: 100));

      // Verify onSave was called with values
      expect(savedValues, isNotNull);
      expect(savedValues!['throat_opening_mm'], 42.0);

      // Verify ComplianceResultCard is rendered with CONDEMNED state
      expect(find.byType(ComplianceResultCard), findsOneWidget);
      expect(find.text('Compliance: Condemned'), findsOneWidget);
      expect(find.text('Highest severity: critical'), findsOneWidget);
      expect(find.text('Reasons: Throat opening under minimum tolerance'), findsOneWidget);
    });
  });
}
