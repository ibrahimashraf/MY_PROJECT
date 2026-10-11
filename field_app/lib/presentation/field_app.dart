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
import '../sync/in_app_camera.dart';


List<DynamicFormFieldDefinition> formFieldsForWorkPack(InspectionWorkPack pack) {
  return pack.items.map((item) {
    FieldPrimitiveType type;
    switch (item.responseType) {
      case ChecklistResponseType.number:
        type = FieldPrimitiveType.numericTolerance;
        break;
      case ChecklistResponseType.boolean:
      case ChecklistResponseType.passFailNA:
        type = FieldPrimitiveType.booleanPassFail;
        break;
      case ChecklistResponseType.choice:
      case ChecklistResponseType.text:
        type = FieldPrimitiveType.textInput;
        break;
    }
    return DynamicFormFieldDefinition(
      fieldId: item.id,
      label: '[${item.sectionId}] ${item.prompt}',
      type: type,
      required: item.required,
      helpText: 'Asset: ${item.assetId} · Standard/Section: ${item.sectionId}',
    );
  }).toList(growable: false);
}

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
      inspectionType: 'Multi-Discipline Engineering Inspection',
      procedureVersion: 'prepared-work-pack-v2',
      packageId: 'assigned-work-package',
      packageVersion: 1,
      schemaVersion: 1,
      packageHash: 'sha256:assigned-package-hash-pending-authority',
      scheduledDate: DateTime.now().toUtc(),
      items: const [
        ChecklistItem(
          id: 'shackle-pin-diameter',
          sectionId: 'lifting-tackle-shackle',
          prompt: 'Measure shackle pin diameter and crown wear (BS EN 13889 / ISO 2415).',
          assetId: 'shackle-asset-01',
          responseType: ChecklistResponseType.number,
        ),
        ChecklistItem(
          id: 'sling-fiber-integrity',
          sectionId: 'lifting-tackle-sling',
          prompt: 'Verify synthetic webbing sling cuts, abrasion, and core integrity (BS EN 1492 / ISO 4878).',
          assetId: 'sling-asset-02',
          responseType: ChecklistResponseType.passFailNA,
        ),
        ChecklistItem(
          id: 'pipe-wall-thickness',
          sectionId: 'process-piping',
          prompt: 'Record minimum measured UT pipe spool wall thickness in mm (ASME B31.3 / API 570).',
          assetId: 'pipe-spool-03',
          responseType: ChecklistResponseType.number,
        ),
        ChecklistItem(
          id: 'vessel-shell-distortion',
          sectionId: 'pressure-vessel',
          prompt: 'Record pressure vessel shell surface pitting, corrosion allowance, and relief tag (API 510).',
          assetId: 'vessel-asset-04',
          responseType: ChecklistResponseType.passFailNA,
        ),
        ChecklistItem(
          id: 'atex-enclosure-gap',
          sectionId: 'atex-motor',
          prompt: 'Verify Ex d flameproof enclosure joint gap clearance and bolt torques (BS EN IEC 60079-17).',
          assetId: 'motor-ex-05',
          responseType: ChecklistResponseType.number,
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
                formTitle: '${draft.workPack.inspectionType} Form',
                fields: formFieldsForWorkPack(draft.workPack),
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
                onHandoverCompleted: (handoverData) =>
                    controller.completeHandover(handoverData),
              ),
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
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Capture inspection (${draft.workPack.items.length} items)',
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 16),
            for (final item in draft.workPack.items) ...[
              Container(
                margin: const EdgeInsets.only(bottom: 16),
                padding: const EdgeInsets.all(12),
                decoration: BoxDecoration(
                  border: Border.all(color: Colors.grey.shade300),
                  borderRadius: BorderRadius.circular(8),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      '[${item.sectionId}] Asset: ${item.assetId}',
                      style: const TextStyle(
                          fontWeight: FontWeight.bold,
                          fontSize: 13,
                          color: Colors.blueGrey),
                    ),
                    const SizedBox(height: 4),
                    Text(item.prompt, style: const TextStyle(fontSize: 15)),
                    const SizedBox(height: 10),
                    _ItemResponseField(
                      item: item,
                      controller: controller,
                      draft: draft,
                    ),
                  ],
                ),
              ),
            ],
            const SizedBox(height: 8),
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
                  label: const Text('Submit / Queue Inspection'),
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

class _ItemResponseField extends StatefulWidget {
  const _ItemResponseField({
    required this.item,
    required this.controller,
    required this.draft,
  });

  final ChecklistItem item;
  final FieldAppController controller;
  final InspectionDraft draft;

  @override
  State<_ItemResponseField> createState() => _ItemResponseFieldState();
}

class _ItemResponseFieldState extends State<_ItemResponseField> {
  late final TextEditingController _textController;

  @override
  void initState() {
    super.initState();
    final existing = widget.draft.findings[widget.item.id]?.response ?? '';
    _textController = TextEditingController(text: existing);
  }

  @override
  void didUpdateWidget(covariant _ItemResponseField oldWidget) {
    super.didUpdateWidget(oldWidget);
    final existing = widget.draft.findings[widget.item.id]?.response ?? '';
    if (_textController.text != existing) {
      _textController.text = existing;
    }
  }

  @override
  void dispose() {
    _textController.dispose();
    super.dispose();
  }

  void _onChanged(String val) {
    widget.controller.recordResponse(
      item: widget.item,
      response: val,
    );
  }

  @override
  Widget build(BuildContext context) {
    switch (widget.item.responseType) {
      case ChecklistResponseType.number:
        return TextField(
          controller: _textController,
          keyboardType: const TextInputType.numberWithOptions(decimal: true),
          decoration: const InputDecoration(
            labelText: 'Measurement / Value',
            border: OutlineInputBorder(),
          ),
          onChanged: _onChanged,
        );
      case ChecklistResponseType.boolean:
        return CheckboxListTile(
          contentPadding: EdgeInsets.zero,
          title: const Text('Confirmed'),
          value: _textController.text == 'true',
          onChanged: (val) {
            final str = val == true ? 'true' : 'false';
            setState(() => _textController.text = str);
            _onChanged(str);
          },
        );
      case ChecklistResponseType.choice:
        return DropdownButtonFormField<String>(
          initialValue: widget.item.options.contains(_textController.text)
              ? _textController.text
              : null,
          decoration: const InputDecoration(
            labelText: 'Select approved value',
            border: OutlineInputBorder(),
          ),
          items: widget.item.options
              .map((option) =>
                  DropdownMenuItem(value: option, child: Text(option)))
              .toList(growable: false),
          onChanged: (val) {
            final str = val ?? '';
            setState(() => _textController.text = str);
            _onChanged(str);
          },
        );
      case ChecklistResponseType.passFailNA:
        const options = <String, String>{
          'pass': 'Pass',
          'fail': 'Fail',
          'not_applicable': 'Not applicable',
        };
        return DropdownButtonFormField<String>(
          initialValue: options.containsKey(_textController.text)
              ? _textController.text
              : null,
          decoration: const InputDecoration(
            labelText: 'Assessment',
            border: OutlineInputBorder(),
          ),
          items: options.entries
              .map((e) => DropdownMenuItem(value: e.key, child: Text(e.value)))
              .toList(growable: false),
          onChanged: (val) {
            final str = val ?? '';
            setState(() => _textController.text = str);
            _onChanged(str);
          },
        );
      case ChecklistResponseType.text:
        return TextField(
          controller: _textController,
          decoration: const InputDecoration(
            labelText: 'Observation',
            border: OutlineInputBorder(),
          ),
          onChanged: _onChanged,
        );
    }
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

