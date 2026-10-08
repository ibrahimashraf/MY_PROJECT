import 'package:flutter/material.dart';

import '../application/field_app_controller.dart';
import '../domain/models.dart';
import '../domain/inspection_draft.dart';
import '../outbox/outbox.dart';
import '../sync/event_stream_client.dart';
import '../workpackages/package_compatibility.dart';
import 'adaptive_scaffold.dart';
import 'custody_handover_view.dart';
import 'dynamic_form_view.dart';
import 'enrollment_screen.dart';
import 'lift_plan_2d_view.dart';
import '../sync/in_app_camera.dart';


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

  void _refresh() {
    final draft = widget.controller.activeDraft;
    if (draft != null) {
      if (draft.notes != null && draft.notes!.isNotEmpty && notesController.text != draft.notes) {
        notesController.text = draft.notes!;
      }
      final firstItem = draft.workPack.items.isNotEmpty ? draft.workPack.items.first : null;
      if (firstItem != null) {
        final existingFinding = draft.findings[firstItem.id];
        if (existingFinding != null && responseController.text != existingFinding.response) {
          responseController.text = existingFinding.response;
        }
      }
    }
    setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    final controller = widget.controller;
    final draft = controller.activeDraft;
    final compatibility = evaluatePackageCompatibility(
      draftPackage: draft?.workPack ?? workPack,
      assignedPackage: workPack,
      now: DateTime.now().toUtc(),
      packageExpiresAt: controller.authority.expiresAt,
      cachedAuthorityEpoch: controller.authority.epoch,
      requiredAuthorityEpoch: controller.authority.epoch,
    );
    return AdaptiveScaffold(
      title: 'INTEGIN Field',
      actions: [
        IconButton(
          tooltip: 'Enroll device',
          icon: const Icon(Icons.vpn_key_outlined),
          onPressed: () => Navigator.of(context).push(
            MaterialPageRoute(
              builder: (_) => EnrollmentScreen(
                tenantId: controller.context.tenantId,
                organizationId: controller.context.organizationId,
              ),
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16),
          child: Center(
              child: Text(
                  '${controller.connectivity.name} · ${controller.queuedCount} queued')),
        ),
      ],
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _StatusCard(controller: controller, workPack: workPack),
            const SizedBox(height: 20),
            // FlashingText removed
            const SizedBox(height: 20),
            if (draft == null)
              _WorkPackCard(controller: controller, workPack: workPack)
            else ...[
              _PackageCompatibilityCard(decision: compatibility),
              const SizedBox(height: 16),
              _CaptureCard(
                controller: controller,
                draft: draft,
                compatibility: compatibility,
                responseController: responseController,
                notesController: notesController,
              ),
              const SizedBox(height: 16),
              DynamicFormEngineView(
                formTitle: 'Lifting Hook Geometry Verification (0052)',
                fields: const [
                  DynamicFormFieldDefinition(
                    fieldId: 'throat_opening_mm',
                    label: 'Hook Throat Opening (mm)',
                    type: FieldPrimitiveType.numericTolerance,
                    minValue: 45.0,
                    maxValue: 55.0,
                    unit: 'mm',
                    helpText: 'Tolerance within +/- 10% of nominal',
                  ),
                  DynamicFormFieldDefinition(
                    fieldId: 'latch_engagement',
                    label: 'Safety Latch Lock Engagement',
                    type: FieldPrimitiveType.booleanPassFail,
                    helpText: 'Latch must self-lock under spring tension',
                  ),
                  DynamicFormFieldDefinition(
                    fieldId: 'hook_tag_scan',
                    label: 'NFC/RFID Asset Identification Tag',
                    type: FieldPrimitiveType.qrConfirmation,
                    helpText: 'Verify physical asset tag against 0054 manifest',
                  ),
                  DynamicFormFieldDefinition(
                    fieldId: 'hook_crack_photo',
                    label: 'Critical Hook Throat Stress Concentration Photo',
                    type: FieldPrimitiveType.photoEvidence,
                    helpText: 'Encrypted close-up photograph under high illumination',
                  ),
                ],
                tenantId: controller.context.tenantId,
                tusClient: controller.tusClient,
                onPickPhoto: InAppCameraPhotoService(context).pickImage,
                onSaveWithOutcome: (values, outcome) {
                  if (outcome != null) {
                    controller.recordEvaluationOutcome(outcome);
                  }
                },
                onSave: (values) {
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(
                        content: Text('Dynamic Form Primitives Stored Locally')),
                  );
                },
              ),
              const SizedBox(height: 16),
              CustodyHandoverView(
                workOrderId: workPack.inspectionId,
                assetId: workPack.rootAssetId,
                onHandoverCompleted: (handoverData) async {
                  final notes = [
                    if (handoverData['releasing_party'] != null)
                      'Releasing: ${handoverData['releasing_party']}',
                    if (handoverData['accepting_party'] != null)
                      'Accepting: ${handoverData['accepting_party']}',
                    if (handoverData['site_notes'] != null &&
                        (handoverData['site_notes'] as String).isNotEmpty)
                      handoverData['site_notes'],
                  ].join(' | ');
                  if (controller.connectivity == ConnectivityState.online) {
                    final ok = await controller.submitLive(notes: notes);
                    if (!ok) {
                      await controller.queueForSync(notes: notes);
                    }
                  } else {
                    await controller.queueForSync(notes: notes);
                  }
                },
              ),
              const SizedBox(height: 16),
              const _LiftPlanPreviewCard(),
            ],
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
      secondaryBody: controller.recentEvents.isNotEmpty
          ? SingleChildScrollView(
              padding: const EdgeInsets.all(24),
              child: _StationLiveEventsCard(events: controller.recentEvents),
            )
          : null,
    );
  }
}

class _StatusCard extends StatelessWidget {
  const _StatusCard({required this.controller, required this.workPack});

  final FieldAppController controller;
  final InspectionWorkPack workPack;

  void _showOutboxReviewSheet(BuildContext context) async {
    final store = controller.outboxStore;
    if (store == null) return;
    final allEntries = (await store.all())
        .where((e) => e.state != OutboxState.abandoned)
        .toList();
    if (!context.mounted) return;

    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (sheetContext) {
        return DraggableScrollableSheet(
          initialChildSize: 0.6,
          minChildSize: 0.3,
          maxChildSize: 0.9,
          expand: false,
          builder: (scrollContext, scrollController) {
            return Padding(
              padding: const EdgeInsets.all(16),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      Text(
                        'Outbox Mutations (${allEntries.length})',
                        style: Theme.of(sheetContext).textTheme.titleLarge,
                      ),
                      IconButton(
                        icon: const Icon(Icons.close),
                        onPressed: () => Navigator.of(sheetContext).pop(),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  const Text(
                    'Unseal any pending mutation back into an active draft to edit responses, notes, or attachments before re-queuing.',
                    style: TextStyle(fontSize: 13, color: Colors.black54),
                  ),
                  const Divider(height: 24),
                  Expanded(
                    child: allEntries.isEmpty
                        ? const Center(child: Text('Outbox is empty'))
                        : ListView.separated(
                            controller: scrollController,
                            itemCount: allEntries.length,
                            separatorBuilder: (_, __) => const Divider(),
                            itemBuilder: (_, index) {
                              final entry = allEntries[index];
                              final isEditable = entry.state == OutboxState.queued ||
                                  entry.state == OutboxState.held ||
                                  entry.state.isFailure;
                              return ListTile(
                                leading: CircleAvatar(
                                  backgroundColor: entry.state == OutboxState.applied
                                      ? Colors.green.shade100
                                      : entry.state.isFailure
                                          ? Colors.red.shade100
                                          : Colors.amber.shade100,
                                  child: Icon(
                                    entry.state == OutboxState.applied
                                        ? Icons.check
                                        : entry.state.isFailure
                                            ? Icons.error_outline
                                            : Icons.access_time,
                                    color: entry.state == OutboxState.applied
                                        ? Colors.green.shade800
                                        : entry.state.isFailure
                                            ? Colors.red.shade800
                                            : Colors.amber.shade800,
                                    size: 20,
                                  ),
                                ),
                                title: Text(
                                  'Seq #${entry.mutation.sequenceNumber} · ${entry.state.name.toUpperCase()}',
                                  style: const TextStyle(fontWeight: FontWeight.bold),
                                ),
                                subtitle: Column(
                                  crossAxisAlignment: CrossAxisAlignment.start,
                                  children: [
                                    Text('Tx: ${entry.mutation.transactionId}'),
                                    if (entry.lastError != null && entry.lastError!.isNotEmpty)
                                      Text(
                                        'Error: ${entry.lastError}',
                                        style: TextStyle(color: Colors.red.shade700, fontSize: 12),
                                      ),
                                    Text(
                                      'Captured: ${entry.mutation.capturedAt.toLocal()}',
                                      style: const TextStyle(fontSize: 11),
                                    ),
                                  ],
                                ),
                                trailing: isEditable
                                    ? FilledButton.tonal(
                                        onPressed: () async {
                                          Navigator.of(sheetContext).pop();
                                          await controller.editOutboxEntry(entry, workPack);
                                          if (context.mounted) {
                                            ScaffoldMessenger.of(context).showSnackBar(
                                              SnackBar(
                                                content: Text('Unsealed Seq #${entry.mutation.sequenceNumber} back into draft for editing.'),
                                              ),
                                            );
                                          }
                                        },
                                        child: const Text('Edit'),
                                      )
                                    : null,
                              );
                            },
                          ),
                  ),
                ],
              ),
            );
          },
        );
      },
    );
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            const Text('Offline authority',
                style: TextStyle(fontWeight: FontWeight.bold, fontSize: 16)),
            const SizedBox(height: 8),
            Text('Status: ${controller.canWorkOffline ? "Trusted and valid" : "Blocked"}'),
            Text('Expires: ${controller.authority.expiresAt.toLocal()}'),
            Text('Outbox: ${controller.queuedCount}  |  Failures: ${controller.failedCount}'),
            Text('Tenant: ${controller.context.tenantId}'),
            const SizedBox(height: 16),
            Wrap(
              spacing: 12,
              runSpacing: 8,
              children: [
                FilledButton.tonalIcon(
                  onPressed:
                      controller.isSyncConfigured && controller.queuedCount > 0
                          ? () async {
                              final messenger = ScaffoldMessenger.of(context);
                              final outcomes = await controller.flushOutbox();
                              final pending = await controller.outboxStore?.pending() ?? [];
                              final errorMsg = pending.map((e) => e.lastError).where((e) => e != null && e.isNotEmpty).join(" | ");
                              if (context.mounted) {
                                messenger.showSnackBar(
                                  SnackBar(
                                    content: Text('Sync completed: ${outcomes.map((o) => o.name).join(", ")}${errorMsg.isNotEmpty ? " ($errorMsg)" : ""}'),
                                  ),
                                );
                              }
                            }
                          : null,
                  icon: const Icon(Icons.sync),
                  label: Text(controller.isSyncConfigured
                      ? 'Sync outbox'
                      : 'Sync not configured'),
                ),
                OutlinedButton.icon(
                  onPressed: () => _showOutboxReviewSheet(context),
                  icon: const Icon(Icons.playlist_play),
                  label: const Text('Review / Edit Outbox'),
                ),
                if (controller.failedCount > 0 || controller.outboxSummary.held > 0) ...[
                  OutlinedButton.icon(
                    onPressed: () async {
                      final count = await controller.retryFailedOutbox();
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text('Re-queued $count entries back to outbox')),
                        );
                      }
                    },
                    icon: const Icon(Icons.refresh),
                    label: const Text('Re-queue to outbox'),
                  ),
                  TextButton.icon(
                    onPressed: () async {
                      final cleared = await controller.clearFailures();
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text('Cleared $cleared failed submissions')),
                        );
                      }
                    },
                    icon: const Icon(Icons.delete_sweep, color: Colors.red),
                    label: const Text('Clear failures', style: TextStyle(color: Colors.red)),
                  ),
                ],
                if (controller.outboxSummary.applied > 0 || controller.outboxSummary.duplicate > 0)
                  TextButton.icon(
                    onPressed: () async {
                      final cleared = await controller.clearApplied();
                      if (context.mounted) {
                        ScaffoldMessenger.of(context).showSnackBar(
                          SnackBar(content: Text('Purged $cleared completed entries')),
                        );
                      }
                    },
                    icon: const Icon(Icons.cleaning_services, color: Colors.orange),
                    label: const Text('Clear applied', style: TextStyle(color: Colors.orange)),
                  ),
              ],
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
      required this.compatibility,
      required this.responseController,
      required this.notesController});

  final FieldAppController controller;
  final InspectionDraft draft;
  final PackageCompatibilityDecision compatibility;
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
            _TypedResponseInput(item: item, controller: responseController),
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
                  onPressed: () async {
                    if (controller.connectivity == ConnectivityState.online) {
                      final ok = await controller.submitLive(
                          packageCompatibility: compatibility,
                          notes: notesController.text);
                      if (!ok) {
                        await controller.queueForSync(
                            packageCompatibility: compatibility,
                            notes: notesController.text);
                      }
                    } else {
                      await controller.queueForSync(
                          packageCompatibility: compatibility,
                          notes: notesController.text);
                    }
                  },
                  icon: const Icon(Icons.cloud_upload_outlined),
                  label: const Text('Submit / Queue'),
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

class _TypedResponseInput extends StatelessWidget {
  const _TypedResponseInput({required this.item, required this.controller});

  final ChecklistItem item;
  final TextEditingController controller;

  @override
  Widget build(BuildContext context) {
    switch (item.responseType) {
      case ChecklistResponseType.number:
        return TextField(
          controller: controller,
          keyboardType: const TextInputType.numberWithOptions(decimal: true),
          decoration: const InputDecoration(
            labelText: 'Measurement',
            border: OutlineInputBorder(),
          ),
        );
      case ChecklistResponseType.boolean:
        return CheckboxListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('Confirmed'),
          value: controller.text == 'true',
          onChanged: (value) =>
              controller.text = value == true ? 'true' : 'false',
        );
      case ChecklistResponseType.choice:
        return DropdownButtonFormField<String>(
          initialValue:
              item.options.contains(controller.text) ? controller.text : null,
          decoration: const InputDecoration(
            labelText: 'Select approved value',
            border: OutlineInputBorder(),
          ),
          items: item.options
              .map((option) =>
                  DropdownMenuItem(value: option, child: Text(option)))
              .toList(growable: false),
          onChanged: (value) => controller.text = value ?? '',
        );
      case ChecklistResponseType.passFailNA:
        const options = <String, String>{
          'pass': 'Pass',
          'fail': 'Fail',
          'not_applicable': 'Not applicable',
        };
        return DropdownButtonFormField<String>(
          initialValue:
              options.containsKey(controller.text) ? controller.text : null,
          decoration: const InputDecoration(
            labelText: 'Assessment',
            border: OutlineInputBorder(),
          ),
          items: options.entries
              .map((entry) =>
                  DropdownMenuItem(value: entry.key, child: Text(entry.value)))
              .toList(growable: false),
          onChanged: (value) => controller.text = value ?? '',
        );
      case ChecklistResponseType.text:
        return TextField(
          controller: controller,
          decoration: const InputDecoration(
            labelText: 'Observation or measurement',
            border: OutlineInputBorder(),
          ),
        );
    }
  }
}

class _PackageCompatibilityCard extends StatelessWidget {
  const _PackageCompatibilityCard({required this.decision});

  final PackageCompatibilityDecision decision;

  @override
  Widget build(BuildContext context) {
    final color = switch (decision.state) {
      PackageCompatibilityState.current => Colors.green.shade700,
      PackageCompatibilityState.updateAvailable => Colors.blue.shade700,
      PackageCompatibilityState.needsFormUpdate => Colors.orange.shade800,
      PackageCompatibilityState.packageExpired => Colors.red.shade700,
      PackageCompatibilityState.authorityStale => Colors.red.shade700,
    };
    final icon = switch (decision.state) {
      PackageCompatibilityState.current => Icons.verified_outlined,
      PackageCompatibilityState.updateAvailable => Icons.system_update_outlined,
      PackageCompatibilityState.needsFormUpdate =>
        Icons.assignment_late_outlined,
      PackageCompatibilityState.packageExpired => Icons.schedule_outlined,
      PackageCompatibilityState.authorityStale => Icons.key_off_outlined,
    };
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(icon, color: color),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text('Package status',
                      style: Theme.of(context).textTheme.titleMedium),
                  const SizedBox(height: 4),
                  Text(decision.userMessage),
                  if (decision.requiredNewItemIds.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    Text(
                        'Required update fields: ${decision.requiredNewItemIds.join(', ')}'),
                  ],
                  if (decision.blocksFinalCompletion) ...[
                    const SizedBox(height: 4),
                    const Text(
                        'Final completion and authoritative sync remain blocked.'),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _StationLiveEventsCard extends StatelessWidget {
  const _StationLiveEventsCard({required this.events});

  final List<StationEvent> events;

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 2,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.sensors, color: Colors.blueAccent),
                const SizedBox(width: 8),
                Text(
                  'Real-Time Station Feed (${events.length})',
                  style: Theme.of(context).textTheme.titleMedium,
                ),
              ],
            ),
            const SizedBox(height: 12),
            ...events.take(5).map((e) => Padding(
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  child: Row(
                    children: [
                      Container(
                        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                        decoration: BoxDecoration(
                          color: Colors.blue.withAlpha(30),
                          borderRadius: BorderRadius.circular(4),
                        ),
                        child: Text(
                          e.type,
                          style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 12),
                        ),
                      ),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          e.payload.toString(),
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(fontSize: 12),
                        ),
                      ),
                      Text(
                        '${e.timestamp.hour.toString().padLeft(2, '0')}:${e.timestamp.minute.toString().padLeft(2, '0')}',
                        style: Theme.of(context).textTheme.bodySmall,
                      ),
                    ],
                  ),
                )),
          ],
        ),
      ),
    );
  }
}

class _LiftPlanPreviewCard extends StatefulWidget {
  const _LiftPlanPreviewCard();

  @override
  State<_LiftPlanPreviewCard> createState() => _LiftPlanPreviewCardState();
}

class _LiftPlanPreviewCardState extends State<_LiftPlanPreviewCard> {
  bool _showPlanView = true;

  @override
  Widget build(BuildContext context) {
    return Card(
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: const BorderSide(color: Color(0xFF1E293B)),
      ),
      color: const Color(0xFF0F172A),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                const Text(
                  'Offline 2D Lift Plan Preview (Phase 3)',
                  style: TextStyle(
                    color: Colors.white,
                    fontWeight: FontWeight.bold,
                    fontSize: 14,
                  ),
                ),
                TextButton.icon(
                  onPressed: () => setState(() => _showPlanView = !_showPlanView),
                  icon: Icon(_showPlanView ? Icons.layers_outlined : Icons.landscape_outlined, size: 16),
                  label: Text(_showPlanView ? 'Elevation' : 'Plan View'),
                ),
              ],
            ),
            const SizedBox(height: 12),
            SizedBox(
              height: 280,
              child: LiftPlan2DView(
                craneModel: 'Liebherr LTM 1500-8.1',
                boomLengthMeters: 36.9,
                workingRadiusMeters: 14.0,
                boomAngleDeg: 63.4,
                slewAngleDeg: 25.0,
                grossLoadTonnes: 45.0,
                ratedCapacityTonnes: 102.0,
                groundBearingFoS: 1.68,
                showPlanView: _showPlanView,
              ),
            ),
          ],
        ),
      ),
    );
  }
}

