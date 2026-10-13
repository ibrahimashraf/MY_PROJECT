import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/assurance/rule_bundle.dart';
import 'package:integin_field_app/assurance/rule_evaluator.dart';

void main() {
  group('DynamicRuleEvaluator', () {
    const evaluator = DynamicRuleEvaluator();

    test('evaluates arithmetic expressions with variables', () {
      final variables = <String, double>{
        'finding.measured_value': 18.0,
        'asset.nominal_diameter': 20.0,
      };

      final result = evaluator.evaluateExpression(
        'finding.measured_value / asset.nominal_diameter',
        variables,
      );

      expect(result, closeTo(0.9, 1e-9));
    });

    test('throws StateError on division by zero', () {
      final variables = <String, double>{
        'finding.measured_value': 18.0,
        'asset.nominal_diameter': 0.0,
      };

      expect(
        () => evaluator.evaluateExpression(
          'finding.measured_value / asset.nominal_diameter',
          variables,
        ),
        throwsStateError,
      );
    });

    test('evaluates comparison operators correctly', () {
      const branchLt = EvaluationBranch(
        operator: RuleOperator.lt,
        threshold: 0.90,
        severity: AssuranceSeverity.critical,
        reason: 'Below 90%',
        isBlocking: true,
      );

      const branchGte = EvaluationBranch(
        operator: RuleOperator.gte,
        threshold: 0.90,
        severity: AssuranceSeverity.info,
        reason: 'OK',
        isBlocking: false,
      );

      expect(evaluator.matchesBranch(0.89, branchLt), isTrue);
      expect(evaluator.matchesBranch(0.90, branchLt), isFalse);
      expect(evaluator.matchesBranch(0.90, branchGte), isTrue);
      expect(evaluator.matchesBranch(0.91, branchGte), isTrue);
    });

    test('evaluates ISO 4309 dynamic rule bundle to CONDEMNED on critical failure', () {
      final bundle = RuleBundle(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.0',
        ruleHash: 'sha256:dynamic-ISO4309_LIFTING_ROPES-1.0.0',
        rules: [
          const DecisionRule(
            ruleId: 'ISO4309_DIAMETER_REDUCTION',
            expression: 'finding.measured_value / asset.nominal_diameter',
            evaluations: [
              EvaluationBranch(
                operator: RuleOperator.lt,
                threshold: 0.90,
                severity: AssuranceSeverity.critical,
                reason: 'Diameter fell below 90% of nominal',
                isBlocking: true,
              ),
              EvaluationBranch(
                operator: RuleOperator.lt,
                threshold: 0.95,
                severity: AssuranceSeverity.minor,
                reason: 'Diameter reduction exceeds 5%',
                isBlocking: false,
              ),
            ],
          ),
        ],
      );

      // 1. Critical Case: 17.8mm / 20.0mm = 0.89 (< 0.90) -> CONDEMNED
      final criticalOutcome = evaluator.evaluateBundle(
        bundle: bundle,
        variables: {
          'finding.measured_value': 17.8,
          'asset.nominal_diameter': 20.0,
        },
      );

      expect(criticalOutcome.isCondemned, isTrue);
      expect(criticalOutcome.isNonCompliant, isTrue);
      expect(criticalOutcome.state, 'CONDEMNED');
      expect(criticalOutcome.highestSeverity, AssuranceSeverity.critical);
      expect(criticalOutcome.reasons, contains('Diameter fell below 90% of nominal'));

      // 2. Minor Case: 18.6mm / 20.0mm = 0.93 (>= 0.90, < 0.95) -> CONDITIONAL
      final minorOutcome = evaluator.evaluateBundle(
        bundle: bundle,
        variables: {
          'finding.measured_value': 18.6,
          'asset.nominal_diameter': 20.0,
        },
      );

      expect(minorOutcome.isCondemned, isFalse);
      expect(minorOutcome.isNonCompliant, isFalse);
      expect(minorOutcome.state, 'CONDITIONAL');
      expect(minorOutcome.highestSeverity, AssuranceSeverity.minor);

      // 3. Compliant Case: 19.8mm / 20.0mm = 0.99 (>= 0.95) -> ASSURED
      final passOutcome = evaluator.evaluateBundle(
        bundle: bundle,
        variables: {
          'finding.measured_value': 19.8,
          'asset.nominal_diameter': 20.0,
        },
      );

      expect(passOutcome.isCondemned, isFalse);
      expect(passOutcome.isNonCompliant, isFalse);
      expect(passOutcome.state, 'ASSURED');
      expect(passOutcome.reasons, isEmpty);
    });

    test('evaluates arbitrary non-ISO standard (e.g. pressure vessel thickness) dynamically', () {
      final pressureBundle = RuleBundle(
        bundleId: 'ASME_SECTION_VIII_PRESSURE_VESSEL',
        version: '2.1.0',
        ruleHash: 'sha256:dynamic-ASME-2.1.0',
        rules: [
          const DecisionRule(
            ruleId: 'MINIMUM_WALL_THICKNESS',
            expression: 'wall.measured_thickness - wall.corrosion_allowance',
            evaluations: [
              EvaluationBranch(
                operator: RuleOperator.lt,
                threshold: 12.5,
                severity: AssuranceSeverity.major,
                reason: 'Wall thickness below ASME minimum threshold',
                isBlocking: true,
              ),
            ],
          ),
        ],
      );

      // 14.0mm - 2.0mm = 12.0mm (< 12.5) -> NON_COMPLIANT
      final outcome = evaluator.evaluateBundle(
        bundle: pressureBundle,
        variables: {
          'wall.measured_thickness': 14.0,
          'wall.corrosion_allowance': 2.0,
        },
      );

      expect(outcome.isNonCompliant, isTrue);
      expect(outcome.state, 'NON_COMPLIANT');
      expect(outcome.highestSeverity, AssuranceSeverity.major);
    });

    test('serializes and deserializes EvaluationOutcome and RuleEvaluationResult round-trip', () {
      const outcome = EvaluationOutcome(
        bundleId: 'ISO4309_LIFTING_ROPES',
        version: '1.0.0',
        ruleHash: 'sha256:dynamic-ISO4309_LIFTING_ROPES-1.0.0',
        results: [
          RuleEvaluationResult(
            ruleId: 'ISO4309_DIAMETER_REDUCTION',
            computedValue: 0.89,
            matchedBranch: EvaluationBranch(
              operator: RuleOperator.lt,
              threshold: 0.90,
              severity: AssuranceSeverity.critical,
              reason: 'Diameter reduction critical',
              isBlocking: true,
            ),
            passed: false,
          ),
        ],
        highestSeverity: AssuranceSeverity.critical,
        isNonCompliant: true,
        isCondemned: true,
        reasons: ['Diameter reduction critical'],
        state: 'CONDEMNED',
      );

      final json = outcome.toJson();
      final decoded = EvaluationOutcome.fromJson(json);

      expect(decoded.bundleId, outcome.bundleId);
      expect(decoded.version, outcome.version);
      expect(decoded.ruleHash, outcome.ruleHash);
      expect(decoded.highestSeverity, outcome.highestSeverity);
      expect(decoded.isNonCompliant, outcome.isNonCompliant);
      expect(decoded.isCondemned, outcome.isCondemned);
      expect(decoded.reasons, outcome.reasons);
      expect(decoded.state, outcome.state);
      expect(decoded.results.length, 1);
      expect(decoded.results.first.ruleId, 'ISO4309_DIAMETER_REDUCTION');
      expect(decoded.results.first.computedValue, 0.89);
      expect(decoded.results.first.passed, isFalse);
      expect(decoded.results.first.matchedBranch?.operator, RuleOperator.lt);
      expect(decoded.results.first.matchedBranch?.severity, AssuranceSeverity.critical);
    });
  });
}
