// INTEGIN Field draft migration: plans a safe update without mutating the historic draft.
import '../domain/inspection_draft.dart';

class DraftMigrationPlan {
  const DraftMigrationPlan({
    required this.sourcePackageId,
    required this.sourcePackageVersion,
    required this.targetPackageId,
    required this.targetPackageVersion,
    required this.preservedResponses,
    required this.requiredNewItemIds,
    required this.incompatibleItemIds,
  });

  final String sourcePackageId;
  final int sourcePackageVersion;
  final String targetPackageId;
  final int targetPackageVersion;
  final Map<String, String> preservedResponses;
  final List<String> requiredNewItemIds;
  final List<String> incompatibleItemIds;

  bool get requiresOperatorCompletion =>
      requiredNewItemIds.isNotEmpty || incompatibleItemIds.isNotEmpty;

  String get userMessage {
    if (!requiresOperatorCompletion) {
      return 'Compatible answers are ready for review under the updated package.';
    }
    return 'Form update requires review. Existing answers remain preserved in the original draft.';
  }
}

DraftMigrationPlan planDraftMigration({
  required InspectionDraft historicDraft,
  required InspectionWorkPack targetPackage,
}) {
  final sourceItems = {
    for (final item in historicDraft.workPack.items) item.id: item,
  };
  final preservedResponses = <String, String>{};
  final requiredNewItemIds = <String>[];
  final incompatibleItemIds = <String>[];

  for (final targetItem in targetPackage.items) {
    final sourceItem = sourceItems[targetItem.id];
    final historicResponse =
        historicDraft.findings[targetItem.id]?.response ?? '';
    if (sourceItem == null) {
      if (targetItem.required) {
        requiredNewItemIds.add(targetItem.id);
      }
      continue;
    }
    if (historicResponse.isEmpty) {
      if (targetItem.required &&
          sourceItem.responseType != targetItem.responseType) {
        incompatibleItemIds.add(targetItem.id);
      }
      continue;
    }
    if (_isResponseCompatible(
      response: historicResponse,
      sourceItem: sourceItem,
      targetItem: targetItem,
    )) {
      preservedResponses[targetItem.id] = historicResponse;
    } else {
      incompatibleItemIds.add(targetItem.id);
    }
  }

  return DraftMigrationPlan(
    sourcePackageId: historicDraft.workPack.packageId,
    sourcePackageVersion: historicDraft.workPack.packageVersion,
    targetPackageId: targetPackage.packageId,
    targetPackageVersion: targetPackage.packageVersion,
    preservedResponses: Map.unmodifiable(preservedResponses),
    requiredNewItemIds: List.unmodifiable(requiredNewItemIds),
    incompatibleItemIds: List.unmodifiable(incompatibleItemIds),
  );
}

bool _isResponseCompatible({
  required String response,
  required ChecklistItem sourceItem,
  required ChecklistItem targetItem,
}) {
  if (sourceItem.responseType != targetItem.responseType) {
    return false;
  }
  switch (targetItem.responseType) {
    case ChecklistResponseType.text:
      return true;
    case ChecklistResponseType.number:
      return num.tryParse(response) != null;
    case ChecklistResponseType.boolean:
      return response == 'true' || response == 'false';
    case ChecklistResponseType.choice:
      return targetItem.options.contains(response);
    case ChecklistResponseType.passFailNA:
      return const {'pass', 'fail', 'not_applicable'}.contains(response);
  }
}
