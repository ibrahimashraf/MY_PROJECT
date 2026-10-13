import 'dart:convert';

/// Strict severity monotonicity hierarchy matching the Go backend (Phase 1–8).
enum AssuranceSeverity {
  info(0),
  minor(10),
  major(50),
  critical(100);

  const AssuranceSeverity(this.weight);
  final int weight;

  static AssuranceSeverity fromString(String raw) {
    switch (raw.toUpperCase()) {
      case 'CRITICAL':
        return AssuranceSeverity.critical;
      case 'MAJOR':
        return AssuranceSeverity.major;
      case 'MINOR':
        return AssuranceSeverity.minor;
      case 'INFO':
      default:
        return AssuranceSeverity.info;
    }
  }
}

/// Dynamic operators supported by decision table evaluations.
enum RuleOperator {
  lt,
  lte,
  gt,
  gte,
  eq,
  neq;

  static RuleOperator fromString(String raw) {
    switch (raw.toUpperCase()) {
      case 'LT':
        return RuleOperator.lt;
      case 'LTE':
        return RuleOperator.lte;
      case 'GT':
        return RuleOperator.gt;
      case 'GTE':
        return RuleOperator.gte;
      case 'EQ':
        return RuleOperator.eq;
      case 'NEQ':
        return RuleOperator.neq;
      default:
        throw ArgumentError('Unsupported operator: $raw');
    }
  }
}

/// An individual evaluation branch within a decision rule.
class EvaluationBranch {
  const EvaluationBranch({
    required this.operator,
    required this.threshold,
    required this.severity,
    required this.reason,
    required this.isBlocking,
  });

  final RuleOperator operator;
  final double threshold;
  final AssuranceSeverity severity;
  final String reason;
  final bool isBlocking;

  factory EvaluationBranch.fromJson(Map<String, dynamic> json) {
    return EvaluationBranch(
      operator: RuleOperator.fromString(json['operator'] as String),
      threshold: (json['threshold'] as num).toDouble(),
      severity: AssuranceSeverity.fromString(json['severity'] as String),
      reason: json['reason'] as String? ?? '',
      isBlocking: json['is_blocking'] as bool? ?? false,
    );
  }

  Map<String, dynamic> toJson() => {
        'operator': operator.name.toUpperCase(),
        'threshold': threshold,
        'severity': severity.name.toUpperCase(),
        'reason': reason,
        'is_blocking': isBlocking,
      };
}

/// A decision rule with an arithmetic expression and conditional evaluation branches.
class DecisionRule {
  const DecisionRule({
    required this.ruleId,
    required this.expression,
    required this.evaluations,
  });

  final String ruleId;
  final String expression;
  final List<EvaluationBranch> evaluations;

  factory DecisionRule.fromJson(Map<String, dynamic> json) {
    final rawEvaluations = json['evaluations'] as List<dynamic>? ?? const [];
    return DecisionRule(
      ruleId: json['rule_id'] as String,
      expression: json['expression'] as String,
      evaluations: rawEvaluations
          .map((e) => EvaluationBranch.fromJson(e as Map<String, dynamic>))
          .toList(growable: false),
    );
  }

  Map<String, dynamic> toJson() => {
        'rule_id': ruleId,
        'expression': expression,
        'evaluations': evaluations.map((e) => e.toJson()).toList(),
      };
}

/// A complete rule bundle representing any standard (ISO 4309, ASME, etc.)
class RuleBundle {
  const RuleBundle({
    required this.bundleId,
    required this.version,
    required this.ruleHash,
    required this.rules,
    this.tenantId,
  });

  final String bundleId;
  final String version;
  final String ruleHash;
  final List<DecisionRule> rules;
  final String? tenantId;

  factory RuleBundle.fromJson(Map<String, dynamic> json) {
    final rawRules = json['rules'] as List<dynamic>? ?? const [];
    return RuleBundle(
      bundleId: json['bundle_id'] as String,
      version: json['version'] as String,
      ruleHash: json['rule_hash'] as String? ?? '',
      tenantId: json['tenant_id'] as String?,
      rules: rawRules
          .map((r) => DecisionRule.fromJson(r as Map<String, dynamic>))
          .toList(growable: false),
    );
  }

  Map<String, dynamic> toJson() => {
        'bundle_id': bundleId,
        'version': version,
        'rule_hash': ruleHash,
        if (tenantId != null) 'tenant_id': tenantId,
        'rules': rules.map((r) => r.toJson()).toList(),
      };

  String toJsonString() => jsonEncode(toJson());
}
