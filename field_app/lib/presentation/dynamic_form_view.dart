import 'package:flutter/material.dart';

enum FieldPrimitiveType {
  textInput,
  numericTolerance,
  booleanPassFail,
  photoEvidence,
  qrConfirmation,
}

class DynamicFormFieldDefinition {
  const DynamicFormFieldDefinition({
    required this.fieldId,
    required this.label,
    required this.type,
    this.required = true,
    this.helpText,
    this.minValue,
    this.maxValue,
    this.unit,
  });

  final String fieldId;
  final String label;
  final FieldPrimitiveType type;
  final bool required;
  final String? helpText;
  final double? minValue;
  final double? maxValue;
  final String? unit;
}

class DynamicFormEngineView extends StatefulWidget {
  const DynamicFormEngineView({
    super.key,
    required this.formTitle,
    required this.fields,
    required this.onSave,
  });

  final String formTitle;
  final List<DynamicFormFieldDefinition> fields;
  final ValueChanged<Map<String, dynamic>> onSave;

  @override
  State<DynamicFormEngineView> createState() => _DynamicFormEngineViewState();
}

class _DynamicFormEngineViewState extends State<DynamicFormEngineView> {
  final Map<String, dynamic> _values = {};
  final Map<String, TextEditingController> _controllers = {};

  @override
  void initState() {
    super.initState();
    for (final field in widget.fields) {
      if (field.type == FieldPrimitiveType.textInput ||
          field.type == FieldPrimitiveType.numericTolerance) {
        _controllers[field.fieldId] = TextEditingController();
      } else if (field.type == FieldPrimitiveType.booleanPassFail) {
        _values[field.fieldId] = true; // Default Pass
      }
    }
  }

  @override
  void dispose() {
    for (final controller in _controllers.values) {
      controller.dispose();
    }
    super.dispose();
  }

  Widget _buildFieldWidget(DynamicFormFieldDefinition field) {
    switch (field.type) {
      case FieldPrimitiveType.textInput:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 8.0),
          child: TextField(
            controller: _controllers[field.fieldId],
            decoration: InputDecoration(
              labelText: field.label + (field.required ? ' *' : ''),
              helperText: field.helpText,
              border: const OutlineInputBorder(),
            ),
            onChanged: (val) => _values[field.fieldId] = val,
          ),
        );

      case FieldPrimitiveType.numericTolerance:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 8.0),
          child: Row(
            children: [
              Expanded(
                child: TextField(
                  controller: _controllers[field.fieldId],
                  keyboardType:
                      const TextInputType.numberWithOptions(decimal: true),
                  decoration: InputDecoration(
                    labelText: field.label + (field.required ? ' *' : ''),
                    helperText: field.helpText ??
                        'Allowed range: ${field.minValue ?? '-∞'} to ${field.maxValue ?? '+∞'}',
                    suffixText: field.unit,
                    border: const OutlineInputBorder(),
                  ),
                  onChanged: (val) {
                    final numVal = double.tryParse(val);
                    _values[field.fieldId] = numVal;
                  },
                ),
              ),
            ],
          ),
        );

      case FieldPrimitiveType.booleanPassFail:
        final bool currentVal = _values[field.fieldId] as bool? ?? true;
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 8.0),
          child: Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              border: Border.all(color: Colors.grey.shade400),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(field.label,
                          style: const TextStyle(fontWeight: FontWeight.bold)),
                      if (field.helpText != null)
                        Text(field.helpText!,
                            style: Theme.of(context).textTheme.bodySmall),
                    ],
                  ),
                ),
                SegmentedButton<bool>(
                  segments: const [
                    ButtonSegment(
                        value: true,
                        label: Text('PASS'),
                        icon: Icon(Icons.check_circle, color: Colors.green)),
                    ButtonSegment(
                        value: false,
                        label: Text('FAIL'),
                        icon: Icon(Icons.cancel, color: Colors.red)),
                  ],
                  selected: {currentVal},
                  onSelectionChanged: (newSelection) {
                    setState(() {
                      _values[field.fieldId] = newSelection.first;
                    });
                  },
                ),
              ],
            ),
          ),
        );

      case FieldPrimitiveType.photoEvidence:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 8.0),
          child: Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border: Border.all(color: Colors.grey.shade400),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(field.label,
                    style: const TextStyle(fontWeight: FontWeight.bold)),
                const SizedBox(height: 8),
                OutlinedButton.icon(
                  onPressed: () {
                    // Simulates capturing encrypted evidence
                    _values[field.fieldId] =
                        'evidence-sha256:${DateTime.now().millisecondsSinceEpoch}';
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                          content: Text('Encrypted Photo Evidence Attached')),
                    );
                  },
                  icon: const Icon(Icons.camera_alt),
                  label: const Text('Capture Encrypted Photo (#q=)'),
                ),
              ],
            ),
          ),
        );

      case FieldPrimitiveType.qrConfirmation:
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 8.0),
          child: Container(
            padding: const EdgeInsets.all(16),
            decoration: BoxDecoration(
              border: Border.all(color: Colors.grey.shade400),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Row(
              children: [
                const Icon(Icons.qr_code_scanner, size: 36),
                const SizedBox(width: 16),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(field.label,
                          style: const TextStyle(fontWeight: FontWeight.bold)),
                      Text(field.helpText ?? 'Scan physical tag to confirm',
                          style: Theme.of(context).textTheme.bodySmall),
                    ],
                  ),
                ),
                FilledButton.tonal(
                  onPressed: () {
                    _values[field.fieldId] = 'TAG-AUTH-OK';
                    ScaffoldMessenger.of(context).showSnackBar(
                      const SnackBar(
                          content: Text('Tag Verification Confirmed')),
                    );
                  },
                  child: const Text('Verify Tag'),
                ),
              ],
            ),
          ),
        );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Text(
                    widget.formTitle,
                    style: Theme.of(context).textTheme.titleLarge,
                  ),
                ),
                const SizedBox(width: 8),
                const Chip(
                  label: Text('Schema v1'),
                  avatar: Icon(Icons.verified_user, size: 16),
                ),
              ],
            ),
            const Divider(height: 32),
            ListView.separated(
              shrinkWrap: true,
              physics: const NeverScrollableScrollPhysics(),
              itemCount: widget.fields.length,
              separatorBuilder: (context, index) => const SizedBox(height: 8),
              itemBuilder: (context, index) =>
                  _buildFieldWidget(widget.fields[index]),
            ),
            const SizedBox(height: 24),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                FilledButton.icon(
                  onPressed: () => widget.onSave(_values),
                  icon: const Icon(Icons.save),
                  label: const Text('Save Form to Canonical Outbox'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
