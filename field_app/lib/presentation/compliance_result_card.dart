import 'package:flutter/material.dart';
import '../assurance/rule_evaluator.dart';

/// Widget that displays the result of a rule evaluation.
///
/// The card background colour and icon reflect the [EvaluationOutcome.state]
/// (ASSURED, CONDITIONAL, NON_COMPLIANT, CONDEMNED).
class ComplianceResultCard extends StatelessWidget {
  const ComplianceResultCard({
    Key? key,
    required this.outcome,
  }) : super(key: key);

  final EvaluationOutcome outcome;

  Color _backgroundColor() {
    switch (outcome.state) {
      case EvaluationState.assured:
        return Colors.green.shade100;
      case EvaluationState.conditional:
        return Colors.yellow.shade100;
      case EvaluationState.nonCompliant:
        return Colors.orange.shade100;
      case EvaluationState.condemned:
        return Colors.red.shade100;
    }
  }

  IconData _iconData() {
    switch (outcome.state) {
      case EvaluationState.assured:
        return Icons.check_circle;
      case EvaluationState.conditional:
        return Icons.error_outline;
      case EvaluationState.nonCompliant:
        return Icons.warning_amber;
      case EvaluationState.condemned:
        return Icons.cancel;
    }
  }

  String _label() {
    switch (outcome.state) {
      case EvaluationState.assured:
        return 'Assured';
      case EvaluationState.conditional:
        return 'Conditional';
      case EvaluationState.nonCompliant:
        return 'Non‑Compliant';
      case EvaluationState.condemned:
        return 'Condemned';
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      color: _backgroundColor(),
      margin: const EdgeInsets.symmetric(vertical: 12, horizontal: 8),
      child: ListTile(
        leading: Icon(_iconData(), size: 32),
        title: Text('Compliance: ${_label()}'),
        subtitle: Text('Highest severity: ${outcome.highestSeverity.name}'),
      ),
    );
  }
}
