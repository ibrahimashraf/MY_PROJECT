import 'package:flutter/material.dart';

enum HandoverRole { releasingParty, acceptingParty }

class CustodyHandoverView extends StatefulWidget {
  const CustodyHandoverView({
    super.key,
    required this.workOrderId,
    required this.assetId,
    required this.onHandoverCompleted,
  });

  final String workOrderId;
  final String assetId;
  final ValueChanged<Map<String, dynamic>> onHandoverCompleted;

  @override
  State<CustodyHandoverView> createState() => _CustodyHandoverViewState();
}

class _CustodyHandoverViewState extends State<CustodyHandoverView> {
  final TextEditingController _releasingNameController =
      TextEditingController();
  final TextEditingController _acceptingNameController =
      TextEditingController();
  final TextEditingController _siteNotesController = TextEditingController();

  bool _physicalIntegrityConfirmed = false;
  bool _calibrationSealIntact = false;
  bool _safetyInterlocksFunctional = false;

  @override
  void dispose() {
    _releasingNameController.dispose();
    _acceptingNameController.dispose();
    _siteNotesController.dispose();
    super.dispose();
  }

  bool get _canSubmit =>
      _releasingNameController.text.trim().isNotEmpty &&
      _acceptingNameController.text.trim().isNotEmpty &&
      _physicalIntegrityConfirmed &&
      _calibrationSealIntact &&
      _safetyInterlocksFunctional;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(24.0),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        'Custody & Site Handover',
                        style: Theme.of(context).textTheme.titleLarge,
                      ),
                      Text(
                        'Work Order: ${widget.workOrderId} · Asset: ${widget.assetId}',
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ],
                  ),
                ),
                const SizedBox(width: 8),
                const Chip(
                  label: Text('Migration 0058 Seam'),
                  avatar: Icon(Icons.handshake, size: 16),
                ),
              ],
            ),
            const Divider(height: 32),
            const Text(
              '1. Mandatory Dual-Party Confirmation',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _releasingNameController,
                    decoration: const InputDecoration(
                      labelText: 'Releasing Party Inspector ID / Name *',
                      border: OutlineInputBorder(),
                    ),
                    onChanged: (_) => setState(() {}),
                  ),
                ),
                const SizedBox(width: 16),
                Expanded(
                  child: TextField(
                    controller: _acceptingNameController,
                    decoration: const InputDecoration(
                      labelText: 'Accepting Site Representative *',
                      border: OutlineInputBorder(),
                    ),
                    onChanged: (_) => setState(() {}),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 24),
            const Text(
              '2. Site Condition & Integrity Gate',
              style: TextStyle(fontWeight: FontWeight.bold),
            ),
            CheckboxListTile(
              title: const Text('Physical integrity and structure confirmed intact'),
              value: _physicalIntegrityConfirmed,
              onChanged: (val) =>
                  setState(() => _physicalIntegrityConfirmed = val ?? false),
            ),
            CheckboxListTile(
              title: const Text('Calibration seal undamaged and within validity'),
              value: _calibrationSealIntact,
              onChanged: (val) =>
                  setState(() => _calibrationSealIntact = val ?? false),
            ),
            CheckboxListTile(
              title: const Text('Emergency safety interlocks verified operational'),
              value: _safetyInterlocksFunctional,
              onChanged: (val) =>
                  setState(() => _safetyInterlocksFunctional = val ?? false),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _siteNotesController,
              decoration: const InputDecoration(
                labelText: 'Handover Remarks / Site Conditions',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 24),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                FilledButton.icon(
                  onPressed: _canSubmit
                      ? () {
                          widget.onHandoverCompleted({
                            'releasing_party': _releasingNameController.text.trim(),
                            'accepting_party': _acceptingNameController.text.trim(),
                            'site_notes': _siteNotesController.text.trim(),
                            'timestamp': DateTime.now().toUtc().toIso8601String(),
                          });
                          ScaffoldMessenger.of(context).showSnackBar(
                            const SnackBar(
                              content: Text(
                                  'Dual-Party Custody Handover Transition Committed to Outbox'),
                            ),
                          );
                        }
                      : null,
                  icon: const Icon(Icons.verified),
                  label: const Text('Sign & Seal Handover Transition'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
