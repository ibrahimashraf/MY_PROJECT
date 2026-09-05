import 'models.dart';

enum ChecklistResponseType { text, number, boolean, choice, passFailNA }

class ChecklistItem {
  const ChecklistItem({
    required this.id,
    required this.sectionId,
    required this.prompt,
    required this.assetId,
    this.required = true,
    this.responseType = ChecklistResponseType.text,
    this.options = const [],
  });

  final String id;
  final String sectionId;
  final String prompt;
  final String assetId;
  final bool required;
  final ChecklistResponseType responseType;
  final List<String> options;
}

class InspectionWorkPack {
  const InspectionWorkPack({
    required this.inspectionId,
    required this.rootAssetId,
    required this.inspectionType,
    required this.procedureVersion,
    required this.packageId,
    required this.packageVersion,
    required this.schemaVersion,
    required this.packageHash,
    required this.scheduledDate,
    required this.items,
  });

  final String inspectionId;
  final String rootAssetId;
  final String inspectionType;
  final String procedureVersion;
  final String packageId;
  final int packageVersion;
  final int schemaVersion;
  final String packageHash;
  final DateTime scheduledDate;
  final List<ChecklistItem> items;
}

class LocalCompletenessResult {
  const LocalCompletenessResult({required this.missingItemIds});

  final List<String> missingItemIds;

  bool get isComplete => missingItemIds.isEmpty;
}

class InspectionDraft {
  InspectionDraft({
    required this.context,
    required this.workPack,
    required this.recordedBy,
    required this.createdAt,
  });

  final TenantContext context;
  final InspectionWorkPack workPack;
  final String recordedBy;
  final DateTime createdAt;
  final Map<String, FindingDraft> findings = {};
  InspectionStatus status = InspectionStatus.inProgress;
  String? notes;

  void recordResponse({
    required ChecklistItem item,
    required String response,
    String? note,
    Severity? severity,
  }) {
    final existing = findings[item.id];
    findings[item.id] = FindingDraft(
      id: existing?.id ?? '${workPack.inspectionId}-${item.id}',
      inspectionId: workPack.inspectionId,
      assetId: item.assetId,
      sectionId: item.sectionId,
      itemId: item.id,
      itemPrompt: item.prompt,
      response: response.trim(),
      recordedBy: recordedBy,
      recordedAt: existing?.recordedAt ?? DateTime.now().toUtc(),
      severity: severity,
      notes: note?.trim(),
      evidence: existing?.evidence,
    );
  }

  LocalCompletenessResult validateLocally() {
    final missing = workPack.items
        .where((item) =>
            item.required && (findings[item.id]?.response ?? '').isEmpty)
        .map((item) => item.id)
        .toList(growable: false);
    return LocalCompletenessResult(missingItemIds: missing);
  }

  Map<String, Object?> toPayload() => {
        'inspection_id': workPack.inspectionId,
        'root_asset_id': workPack.rootAssetId,
        'inspection_type': workPack.inspectionType,
        'procedure_version': workPack.procedureVersion,
        'work_package_id': workPack.packageId,
        'work_package_version': workPack.packageVersion,
        'work_package_schema_version': workPack.schemaVersion,
        'work_package_hash': workPack.packageHash,
        'scheduled_date': workPack.scheduledDate.toUtc().toIso8601String(),
        'status': status.name.toUpperCase(),
        'notes': notes,
        'findings': findings.values.map((finding) => finding.toJson()).toList(),
      };

  Map<String, Object?> toJson() => {
        'context': context.toJson(),
        'recorded_by': recordedBy,
        'created_at': createdAt.toUtc().toIso8601String(),
        'status': status.name,
        'notes': notes,
        'findings': findings.map((k, v) => MapEntry(k, v.toJson())),
      };

  void hydrateFindings(Map<String, Object?> rawFindings) {
    rawFindings.forEach((k, v) {
      if (v is Map<String, Object?>) {
        findings[k] = FindingDraft(
          id: v['id'] as String? ?? '',
          inspectionId: v['inspection_id'] as String? ?? workPack.inspectionId,
          assetId: v['asset_id'] as String? ?? '',
          sectionId: v['section_id'] as String? ?? '',
          itemId: v['item_id'] as String? ?? k,
          itemPrompt: v['item_prompt'] as String? ?? '',
          response: v['response'] as String? ?? '',
          recordedBy: v['recorded_by'] as String? ?? recordedBy,
          recordedAt: DateTime.tryParse(v['recorded_at'] as String? ?? '') ?? DateTime.now().toUtc(),
          measuredValue: (v['measured_value'] as num?)?.toDouble(),
          measuredUnit: v['measured_unit'] as String?,
          notes: v['notes'] as String?,
        );
      }
    });
  }
}
