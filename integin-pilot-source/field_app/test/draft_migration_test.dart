// INTEGIN Field migration tests: no update may silently alter or discard a historic package-bound draft.
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/workpackages/draft_migration.dart';

InspectionWorkPack _package({
  required int version,
  required String hash,
  required List<ChecklistItem> items,
}) =>
    InspectionWorkPack(
      inspectionId: 'inspection-1',
      rootAssetId: 'asset-1',
      inspectionType: 'thorough-inspection',
      procedureVersion: 'v$version',
      packageId: 'package-1',
      packageVersion: version,
      schemaVersion: version,
      packageHash: hash,
      scheduledDate: DateTime.utc(2026, 8, 17),
      items: items,
    );

InspectionDraft _draft(InspectionWorkPack package, {String response = 'pass'}) {
  final draft = InspectionDraft(
    context: const TenantContext(
      tenantId: 'tenant-1',
      organizationId: 'org-1',
      environment: 'LIVE',
    ),
    workPack: package,
    recordedBy: 'user-1',
    createdAt: DateTime.utc(2026, 8, 17, 9),
  );
  draft.recordResponse(item: package.items.first, response: response);
  return draft;
}

void main() {
  const condition = ChecklistItem(
    id: 'condition',
    sectionId: 'main',
    prompt: 'Condition',
    assetId: 'asset-1',
    responseType: ChecklistResponseType.passFailNA,
  );
  final original = _package(
    version: 1,
    hash: 'sha256:${'a' * 64}',
    items: const [condition],
  );

  test('preserves compatible historic response without changing source binding',
      () {
    final historic = _draft(original);
    final target = _package(
      version: 2,
      hash: 'sha256:${'b' * 64}',
      items: const [condition],
    );

    final plan =
        planDraftMigration(historicDraft: historic, targetPackage: target);

    expect(plan.preservedResponses, {'condition': 'pass'});
    expect(plan.requiresOperatorCompletion, isFalse);
    expect(historic.workPack.packageVersion, 1);
    expect(historic.findings['condition']!.response, 'pass');
  });

  test(
      'requires review for a newly required target field but retains prior answers',
      () {
    final historic = _draft(original);
    final target = _package(
      version: 2,
      hash: 'sha256:${'b' * 64}',
      items: const [
        condition,
        ChecklistItem(
          id: 'capacity',
          sectionId: 'main',
          prompt: 'Capacity',
          assetId: 'asset-1',
          responseType: ChecklistResponseType.number,
        ),
      ],
    );

    final plan =
        planDraftMigration(historicDraft: historic, targetPackage: target);

    expect(plan.preservedResponses, {'condition': 'pass'});
    expect(plan.requiredNewItemIds, ['capacity']);
    expect(plan.requiresOperatorCompletion, isTrue);
  });

  test('does not map a response into a changed field type', () {
    final historic = _draft(original);
    final target = _package(
      version: 2,
      hash: 'sha256:${'b' * 64}',
      items: const [
        ChecklistItem(
          id: 'condition',
          sectionId: 'main',
          prompt: 'Condition',
          assetId: 'asset-1',
          responseType: ChecklistResponseType.number,
        ),
      ],
    );

    final plan =
        planDraftMigration(historicDraft: historic, targetPackage: target);

    expect(plan.preservedResponses, isEmpty);
    expect(plan.incompatibleItemIds, ['condition']);
    expect(historic.findings['condition']!.response, 'pass');
  });
}
