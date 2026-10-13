import 'package:flutter/material.dart';

import '../assurance/rule_evaluator.dart';

/// Widget that displays the result of a rule evaluation.
///
/// The card background colour and icon reflect the [EvaluationOutcome.state]
/// (ASSURED, CONDITIONAL, NON_COMPLIANT, CONDEMNED).
class ComplianceResultCard extends StatelessWidget {
  const ComplianceResultCard({
    super.key,
    required this.outcome,
  });

  final EvaluationOutcome outcome;

  Color _backgroundColor() {
    switch (outcome.state.toUpperCase()) {
      case 'ASSURED':
        return Colors.green.shade100;
      case 'CONDITIONAL':
        return Colors.yellow.shade100;
      case 'NON_COMPLIANT':
        return Colors.orange.shade100;
      case 'CONDEMNED':
        return Colors.red.shade100;
      default:
        return Colors.grey.shade200;
    }
  }

  IconData _iconData() {
    switch (outcome.state.toUpperCase()) {
      case 'ASSURED':
        return Icons.check_circle;
      case 'CONDITIONAL':
        return Icons.error_outline;
      case 'NON_COMPLIANT':
        return Icons.warning_amber;
      case 'CONDEMNED':
        return Icons.cancel;
      default:
        return Icons.info_outline;
    }
  }

  String _label() {
    switch (outcome.state.toUpperCase()) {
      case 'ASSURED':
        return 'Assured';
      case 'CONDITIONAL':
        return 'Conditional';
      case 'NON_COMPLIANT':
        return 'Non‑Compliant';
      case 'CONDEMNED':
        return 'Condemned';
      default:
        return outcome.state;
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
        subtitle: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Highest severity: ${outcome.highestSeverity.name}'),
            if (outcome.reasons.isNotEmpty)
              Text('Reasons: ${outcome.reasons.join(", ")}'),
          ],
        ),
      ),
    );
  }
}
