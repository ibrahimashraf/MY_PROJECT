import 'package:flutter/material.dart';

import '../application/field_app_controller.dart';
import '../domain/inspection_draft.dart';

class FieldHomePage extends StatefulWidget {
  const FieldHomePage({super.key, required this.controller});

  final FieldAppController controller;

  @override
  State<FieldHomePage> createState() => _FieldHomePageState();
}

class _FieldHomePageState extends State<FieldHomePage> {
  late final TextEditingController responseController;
  late final TextEditingController notesController;
  late final InspectionWorkPack workPack;

  @override
  void initState() {
    super.initState();
    responseController = TextEditingController();
    notesController = TextEditingController();
    workPack = InspectionWorkPack(
      inspectionId: 'assigned-inspection',
      rootAssetId: 'assigned-asset',
      inspectionType: 'Lifting equipment inspection',
      procedureVersion: 'prepared-work-pack',
      packageId: 'assigned-work-package',
      packageVersion: 1,
      schemaVersion: 1,
      packageHash: 'sha256:assigned-package-hash-pending-authority',
      scheduledDate: DateTime.now().toUtc(),
      items: const [
        ChecklistItem(
          id: 'hook-condition',
          sectionId: 'lifting-hook',
          prompt: 'Record the observed hook condition.',
          assetId: 'assigned-asset',
        ),
      ],
    );
    widget.controller.addListener(_refresh);
  }

  @override
  void dispose() {
    widget.controller.removeListener(_refresh);
    responseController.dispose();
    notesController.dispose();
    super.dispose();
  }

  void _refresh() => setState(() {});

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    final draft = controller.activeDraft;
    return Scaffold(
      appBar: AppBar(
        title: const Text('INTEGIN Field'),
        actions: [
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Center(
                child: Text(
                    '${controller.connectivity.name} · ${controller.queuedCount} queued')),
          ),
        ],
      ),
      body: LayoutBuilder(
        builder: (context, constraints) => SingleChildScrollView(
          padding: const EdgeInsets.all(24),
          child: ConstrainedBox(
            constraints: BoxConstraints(
                maxWidth:
                    constraints.maxWidth > 900 ? 900 : constraints.maxWidth),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                _StatusCard(controller: controller),
                const SizedBox(height: 20),
                if (draft == null)
                  _WorkPackCard(controller: controller, workPack: workPack)
                else
                  _CaptureCard(
                      controller: controller,
                      draft: draft,
                      responseController: responseController,
                      notesController: notesController),
                const SizedBox(height: 16),
                _PilotOperatorReviewCard(
                  controller: controller,
                  inspectionId: workPack.inspectionId,
                ),
                if (controller.lastError != null) ...[
                  const SizedBox(height: 16),
                  Text(controller.lastError!,
                      style: TextStyle(
                          color: Theme.of(context).colorScheme.error)),
                ],
              ],
            ),
          ),
        ),
      ),
    );
  }
}

class _StatusCard extends StatelessWidget {
  const _StatusCard({required this.controller});

  final FieldAppController controller;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Wrap(
          spacing: 24,
          runSpacing: 12,
          children: [
            const Text('Offline authority',
                style: TextStyle(fontWeight: FontWeight.bold)),
            Text(controller.canWorkOffline ? 'Trusted and valid' : 'Blocked'),
            Text('Expires ${controller.authority.expiresAt.toLocal()}'),
            Text('Outbox ${controller.queuedCount}'),
            Text('Failures ${controller.failedCount}'),
            Text('Tenant ${controller.context.tenantId}'),
            FilledButton.tonalIcon(
              onPressed:
                  controller.isSyncConfigured && controller.queuedCount > 0
                      ? () => controller.flushOutbox()
                      : null,
              icon: const Icon(Icons.sync),
              label: Text(controller.isSyncConfigured
                  ? 'Sync outbox'
                  : 'Sync not configured'),
            ),
          ],
        ),
      ),
    );
  }
}

class _WorkPackCard extends StatelessWidget {
  const _WorkPackCard({required this.controller, required this.workPack});

  final FieldAppController controller;
  final InspectionWorkPack workPack;

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(workPack.inspectionType,
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 8),
            Text('Assigned work pack · ${workPack.inspectionId}'),
            Text(
                'Procedure ${workPack.procedureVersion} · ${workPack.items.length} required item'),
            const SizedBox(height: 20),
            FilledButton.icon(
              onPressed: controller.canWorkOffline
                  ? () => controller.beginInspection(workPack)
                  : null,
              icon: const Icon(Icons.play_arrow),
              label: const Text('Begin offline inspection'),
            ),
          ],
        ),
      ),
    );
  }
}

class _CaptureCard extends StatelessWidget {
  const _CaptureCard(
      {required this.controller,
      required this.draft,
      required this.responseController,
      required this.notesController});

  final FieldAppController controller;
  final InspectionDraft draft;
  final TextEditingController responseController;
  final TextEditingController notesController;

  @override
  Widget build(BuildContext context) {
    final item = draft.workPack.items.first;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Capture inspection',
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 16),
            Text(item.prompt),
            const SizedBox(height: 12),
            TextField(
                controller: responseController,
                decoration: const InputDecoration(
                    labelText: 'Observation or measurement',
                    border: OutlineInputBorder())),
            const SizedBox(height: 12),
            TextField(
                controller: notesController,
                decoration: const InputDecoration(
                    labelText: 'Notes (optional)',
                    border: OutlineInputBorder())),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                OutlinedButton(
                  onPressed: () => controller.recordResponse(
                      item: item,
                      response: responseController.text,
                      note: notesController.text),
                  child: const Text('Save response locally'),
                ),
                FilledButton.icon(
                  onPressed: () =>
                      controller.queueForSync(notes: notesController.text),
                  icon: const Icon(Icons.cloud_upload_outlined),
                  label: const Text('Queue signed submission'),
                ),
              ],
            ),
            const SizedBox(height: 16),
            Text(
                'Required fields remaining: ${draft.validateLocally().missingItemIds.length}'),
            const SizedBox(height: 8),
            const Text(
                'Primary review, approval, certificate issuance, and public QR verification remain server-controlled.'),
          ],
        ),
      ),
    );
  }
}

class _PilotOperatorReviewCard extends StatelessWidget {
  const _PilotOperatorReviewCard({
    required this.controller,
    required this.inspectionId,
  });

  final FieldAppController controller;
  final String inspectionId;

  @override
  Widget build(BuildContext context) {
    final advisory = controller.advisoryResult;
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Pilot operator review',
                style: Theme.of(context).textTheme.titleLarge),
            const SizedBox(height: 8),
            Text(controller.hasAuthoritativeOutcome
                ? 'Authoritative receipt status: applied or duplicate recorded.'
                : 'Authoritative receipt status: awaiting a successful sync.'),
            const SizedBox(height: 6),
            const Text(
              'Evidence metadata: no attachment is recorded for this pilot inspection.',
            ),
            const SizedBox(height: 6),
            const Text(
              'Advisory results are non-blocking and cannot change primary field work.',
            ),
            const SizedBox(height: 14),
            Wrap(
              spacing: 12,
              runSpacing: 12,
              children: [
                OutlinedButton.icon(
                  onPressed: controller.hasAuthoritativeOutcome
                      ? controller.replayLatestAuthoritative
                      : null,
                  icon: const Icon(Icons.replay_outlined),
                  label: const Text('Re-submit last receipt'),
                ),
                FilledButton.icon(
                  onPressed: controller.advisoryLoading
                      ? null
                      : () => controller.requestPilotAdvisory(inspectionId),
                  icon: const Icon(Icons.insights_outlined),
                  label: Text(controller.advisoryLoading
                      ? 'Requesting advisory…'
                      : 'Request advisory analysis'),
                ),
              ],
            ),
            if (advisory != null) ...[
              const SizedBox(height: 14),
              Text(advisory.title,
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 4),
              Text(advisory.summary),
              const SizedBox(height: 4),
              Text('Confidence ${(advisory.confidence * 100).round()}% · '
                  'blocking: ${advisory.blocking}'),
              if (advisory.limitations.isNotEmpty) ...[
                const SizedBox(height: 4),
                Text('Limit: ${advisory.limitations.first}'),
              ],
            ],
            if (controller.advisoryError != null) ...[
              const SizedBox(height: 10),
              Text(controller.advisoryError!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error)),
            ],
          ],
        ),
      ),
    );
  }
}
