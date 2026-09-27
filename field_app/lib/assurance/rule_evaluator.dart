import 'rule_bundle.dart';

/// The result of evaluating an individual decision rule.
class RuleEvaluationResult {
  const RuleEvaluationResult({
    required this.ruleId,
    required this.computedValue,
    required this.matchedBranch,
    required this.passed,
  });

  final String ruleId;
  final double computedValue;
  final EvaluationBranch? matchedBranch;
  final bool passed;

  AssuranceSeverity get severity =>
      matchedBranch?.severity ?? AssuranceSeverity.info;
  bool get isBlocking => matchedBranch?.isBlocking ?? false;
  String? get reason => matchedBranch?.reason;
}

/// The aggregate outcome of evaluating a complete RuleBundle against inspection facts.
class EvaluationOutcome {
  const EvaluationOutcome({
    required this.bundleId,
    required this.version,
    required this.ruleHash,
    required this.results,
    required this.highestSeverity,
    required this.isNonCompliant,
    required this.isCondemned,
    required this.reasons,
    required this.state,
  });

  final String bundleId;
  final String version;
  final String ruleHash;
  final List<RuleEvaluationResult> results;
  final AssuranceSeverity highestSeverity;
  final bool isNonCompliant;
  final bool isCondemned;
  final List<String> reasons;
  final String state; // "ASSURED", "NON_COMPLIANT", "CONDEMNED", etc.
}

/// Pure Dart, generic, standard-agnostic evaluation engine.
/// Evaluates any arbitrary RuleBundle against field measurements.
class DynamicRuleEvaluator {
  const DynamicRuleEvaluator();

  /// Evaluates an arithmetic expression using dot-notation variables.
  /// Supports: +, -, *, /, and direct variable lookup.
  double evaluateExpression(String expression, Map<String, double> variables) {
    final trimmed = expression.trim();
    if (trimmed.isEmpty) {
      throw ArgumentError('Empty expression cannot be evaluated');
    }

    // Direct variable reference
    if (variables.containsKey(trimmed)) {
      return variables[trimmed]!;
    }

    // Simple binary arithmetic expression (var / var, var * var, etc.)
    for (final op in ['/', '*', '+', '-']) {
      final parts = trimmed.split(op);
      if (parts.length == 2) {
        final left = evaluateExpression(parts[0].trim(), variables);
        final right = evaluateExpression(parts[1].trim(), variables);
        switch (op) {
          case '/':
            if (right == 0.0) {
              throw StateError('Division by zero in rule expression: $expression');
            }
            return left / right;
          case '*':
            return left * right;
          case '+':
            return left + right;
          case '-':
            return left - right;
        }
      }
    }

    // Literal number
    final literal = double.tryParse(trimmed);
    if (literal != null) {
      return literal;
    }

    throw ArgumentError('Unknown variable or malformed expression: $trimmed');
  }

  /// Compares a computed value against an evaluation branch threshold.
  bool matchesBranch(double value, EvaluationBranch branch) {
    const epsilon = 1e-9;
    switch (branch.operator) {
      case RuleOperator.lt:
        return value < (branch.threshold - epsilon);
      case RuleOperator.lte:
        return value <= (branch.threshold + epsilon);
      case RuleOperator.gt:
        return value > (branch.threshold + epsilon);
      case RuleOperator.gte:
        return value >= (branch.threshold - epsilon);
      case RuleOperator.eq:
        return (value - branch.threshold).abs() <= epsilon;
      case RuleOperator.neq:
        return (value - branch.threshold).abs() > epsilon;
    }
  }

  /// Evaluates a full bundle against the provided variables.
  EvaluationOutcome evaluateBundle({
    required RuleBundle bundle,
    required Map<String, double> variables,
  }) {
    final results = <RuleEvaluationResult>[];
    final reasons = <String>[];
    var highestSeverity = AssuranceSeverity.info;
    var hasBlockingFailure = false;
    var hasCriticalFailure = false;

    for (final rule in bundle.rules) {
      final computedValue = evaluateExpression(rule.expression, variables);
      EvaluationBranch? matched;

      // Evaluate branches in priority order
      for (final branch in rule.evaluations) {
        if (matchesBranch(computedValue, branch)) {
          matched = branch;
          break;
        }
      }

      final isPass = matched == null ||
          (!matched.isBlocking && matched.severity == AssuranceSeverity.info);

      if (matched != null) {
        if (matched.severity.weight > highestSeverity.weight) {
          highestSeverity = matched.severity;
        }
        if (matched.isBlocking) {
          hasBlockingFailure = true;
          if (matched.severity == AssuranceSeverity.critical) {
            hasCriticalFailure = true;
          }
        }
        if (matched.reason.isNotEmpty) {
          reasons.add(matched.reason);
        }
      }

      results.add(
        RuleEvaluationResult(
          ruleId: rule.ruleId,
          computedValue: computedValue,
          matchedBranch: matched,
          passed: isPass,
        ),
      );
    }

    // Resolve state using strict monotonicity
    String resolvedState;
    if (hasCriticalFailure) {
      resolvedState = 'CONDEMNED';
    } else if (hasBlockingFailure || highestSeverity == AssuranceSeverity.major) {
      resolvedState = 'NON_COMPLIANT';
    } else if (highestSeverity == AssuranceSeverity.minor) {
      resolvedState = 'CONDITIONAL';
    } else {
      resolvedState = 'ASSURED';
    }

    return EvaluationOutcome(
      bundleId: bundle.bundleId,
      version: bundle.version,
      ruleHash: bundle.ruleHash,
      results: results,
      highestSeverity: highestSeverity,
      isNonCompliant: hasBlockingFailure,
      isCondemned: hasCriticalFailure,
      reasons: reasons,
      state: resolvedState,
    );
  }
}
