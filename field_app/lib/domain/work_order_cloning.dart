import 'inspection_draft.dart';
import 'models.dart';

/// Represents a grouped location section for high-volume expandable work orders.
class LocationScopeGroup {
  const LocationScopeGroup({
    required this.locationId,
    required this.locationName,
    required this.assetIds,
  });

  final String locationId;
  final String locationName;
  final List<String> assetIds;

  static List<LocationScopeGroup> groupByLocation(
      List<Map<String, String>> items) {
    if (items.isEmpty) return const [];

    final Map<String, List<String>> grouped = {};
    for (final item in items) {
      final loc = (item['location_id'] ?? '').trim();
      final locId = loc.isEmpty ? 'DEFAULT_LOCATION' : loc;
      final assetId = item['asset_id'] ?? '';
      if (assetId.isNotEmpty) {
        grouped.putIfAbsent(locId, () => []).add(assetId);
      }
    }

    final sortedKeys = grouped.keys.toList()..sort();
    return sortedKeys.map((k) {
      final assets = grouped[k]!..sort();
      return LocationScopeGroup(
        locationId: k,
        locationName: k == 'DEFAULT_LOCATION' ? 'General Scope Area' : k,
        assetIds: assets,
      );
    }).toList(growable: false);
  }
}

/// SafeDraftCloner enforces ISO 17020 §6.2 proof-isolation invariants:
/// copies editable answers, measurements, tolerances, and findings while
/// strictly purging photographic evidence, attachment hashes, and signatures.
class SafeDraftCloner {
  const SafeDraftCloner._();

  /// Sanitizes raw dynamic form values when creating a cloned asset record.
  /// Evidentiary photos (`offline-evidence:*`) and signatures are stripped.
  static Map<String, dynamic> sanitizeFormValuesForCloning(
    Map<String, dynamic> sourceValues, {
    required String newAssetId,
  }) {
    final sanitized = <String, dynamic>{};

    sourceValues.forEach((key, value) {
      if (key == 'signature' || key == 'biometric_claim' || key == 'device_proof') {
        return; // Strip cryptographic seals
      }

      if (value is String) {
        if (value.startsWith('offline-evidence:') || value.startsWith('tus:')) {
          return; // Strip evidence attachments
        }
      } else if (value is List) {
        // Filter out evidence string lists
        final filteredList = value.where((item) {
          if (item is String && (item.startsWith('offline-evidence:') || item.startsWith('tus:'))) {
            return false;
          }
          return true;
        }).toList();
        if (filteredList.isNotEmpty) {
          sanitized[key] = filteredList;
        }
        return;
      }

      sanitized[key] = value;
    });

    sanitized['_is_cloned_draft'] = true;
    sanitized['_target_asset_id'] = newAssetId;
    return sanitized;
  }

  /// Clones an [InspectionDraft] for a new asset with strict proof isolation.
  static InspectionDraft cloneInspectionDraft({
    required InspectionDraft source,
    required String newInspectionId,
    required String newAssetId,
    required String recordedBy,
    DateTime? scheduledDate,
  }) {
    if (newInspectionId.trim().isEmpty || newAssetId.trim().isEmpty) {
      throw ArgumentError('newInspectionId and newAssetId must not be empty');
    }
    if (newInspectionId.trim() == source.workPack.inspectionId.trim()) {
      throw ArgumentError('newInspectionId cannot match source inspection ID');
    }

    final targetDate = (scheduledDate ?? DateTime.now()).toUtc();

    // 1. Create target workpack pointing to new asset and new inspection ID
    final targetWorkPack = InspectionWorkPack(
      inspectionId: newInspectionId,
      rootAssetId: newAssetId,
      inspectionType: source.workPack.inspectionType,
      procedureVersion: source.workPack.procedureVersion,
      packageId: source.workPack.packageId,
      packageVersion: source.workPack.packageVersion,
      schemaVersion: source.workPack.schemaVersion,
      packageHash: source.workPack.packageHash,
      scheduledDate: targetDate,
      items: source.workPack.items
          .map((i) => ChecklistItem(
                id: i.id,
                sectionId: i.sectionId,
                prompt: i.prompt,
                assetId: newAssetId,
                required: i.required,
                responseType: i.responseType,
                options: i.options,
              ))
          .toList(growable: false),
    );

    // 2. Instantiate new draft
    final targetDraft = InspectionDraft(
      context: source.context,
      workPack: targetWorkPack,
      recordedBy: recordedBy,
      createdAt: targetDate,
    );

    // 3. Deep-copy findings with ZERO evidence references
    source.findings.forEach((itemId, sf) {
      targetDraft.findings[itemId] = FindingDraft(
        id: '$newInspectionId-$itemId',
        inspectionId: newInspectionId,
        assetId: newAssetId,
        sectionId: sf.sectionId,
        itemId: sf.itemId,
        itemPrompt: sf.itemPrompt,
        response: sf.response,
        recordedBy: recordedBy,
        recordedAt: targetDate,
        measuredValue: sf.measuredValue,
        measuredUnit: sf.measuredUnit,
        severity: sf.severity,
        notes: sf.notes,
        evidence: null, // STRICT PROOF ISOLATION: No photos cloned
      );
    });

    targetDraft.status = InspectionStatus.inProgress;
    targetDraft.notes = 'Cloned from ${source.workPack.rootAssetId}';
    return targetDraft;
  }
}
