import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/domain/inspection_draft.dart';
import 'package:integin_field_app/domain/models.dart';
import 'package:integin_field_app/domain/work_order_cloning.dart';

void main() {
  group('LocationScopeGroup', () {
    test('groups items deterministically by location', () {
      final items = [
        {'location_id': 'LOC-BAY-2', 'asset_id': 'SHACKLE-201'},
        {'location_id': 'LOC-BAY-1', 'asset_id': 'HOOK-101'},
        {'location_id': 'LOC-BAY-1', 'asset_id': 'HOOK-102'},
        {'location_id': '', 'asset_id': 'SLING-001'},
      ];

      final groups = LocationScopeGroup.groupByLocation(items);
      expect(groups.length, 3);

      expect(groups[0].locationId, 'DEFAULT_LOCATION');
      expect(groups[0].assetIds, ['SLING-001']);

      expect(groups[1].locationId, 'LOC-BAY-1');
      expect(groups[1].assetIds, ['HOOK-101', 'HOOK-102']);

      expect(groups[2].locationId, 'LOC-BAY-2');
      expect(groups[2].assetIds, ['SHACKLE-201']);
    });
  });

  group('SafeDraftCloner', () {
    test('sanitizeFormValuesForCloning strips photos, signatures and marks draft', () {
      final source = {
        'throat_opening_mm': 45.2,
        'latch_engagement': true,
        'hook_crack_photo': 'offline-evidence:sha256-photo-hash-12345',
        'additional_photos': [
          'offline-evidence:sha256-photo-hash-67890',
          'regular-text-entry'
        ],
        'signature': 'ed25519-sig-bytes',
        'device_proof': 'apple-secure-enclave-assertion',
      };

      final sanitized = SafeDraftCloner.sanitizeFormValuesForCloning(
        source,
        newAssetId: 'HOOK-TARGET-999',
      );

      // Preserves measurements & boolean answers
      expect(sanitized['throat_opening_mm'], 45.2);
      expect(sanitized['latch_engagement'], true);

      // Strictly strips evidence & signatures
      expect(sanitized.containsKey('hook_crack_photo'), isFalse);
      expect(sanitized.containsKey('signature'), isFalse);
      expect(sanitized.containsKey('device_proof'), isFalse);
      expect(sanitized['additional_photos'], ['regular-text-entry']);

      // Adds audit markers
      expect(sanitized['_is_cloned_draft'], isTrue);
      expect(sanitized['_target_asset_id'], 'HOOK-TARGET-999');
    });

    test('cloneInspectionDraft copies findings with strict proof isolation', () {
      final now = DateTime.utc(2026, 10, 5, 12, 0);
      const context = TenantContext(
        tenantId: 'tenant-sa-01',
        organizationId: 'org-aramco',
        environment: 'TESTING',
      );

      final workPack = InspectionWorkPack(
        inspectionId: 'INSP-SRC-001',
        rootAssetId: 'CRANE-HOOK-01',
        inspectionType: 'crane_inspection',
        procedureVersion: 'v1.0',
        packageId: 'PKG-HOOK',
        packageVersion: 1,
        schemaVersion: 1,
        packageHash: 'hash-pkg',
        scheduledDate: now,
        items: const [
          ChecklistItem(
            id: 'item-throat',
            sectionId: 'sec-hook',
            prompt: 'Throat opening mm',
            assetId: 'CRANE-HOOK-01',
          ),
        ],
      );

      final srcDraft = InspectionDraft(
        context: context,
        workPack: workPack,
        recordedBy: 'inspector-bob',
        createdAt: now,
      );

      srcDraft.findings['item-throat'] = FindingDraft(
        id: 'f-1',
        inspectionId: srcDraft.workPack.inspectionId,
        assetId: srcDraft.workPack.rootAssetId,
        sectionId: 'sec-hook',
        itemId: 'item-throat',
        itemPrompt: 'Throat opening mm',
        response: 'MEASURED',
        recordedBy: 'inspector-bob',
        recordedAt: now,
        notes: 'Deformed past 10%',
        severity: Severity.critical,
        evidence: [
          EvidenceReference(
            id: 'ev-1',
            kind: 'photo',
            capturedAt: now,
            localUri: 'file:///photo.jpg',
            sha256: 'digest-12345',
          )
        ],
      );

      // Perform safe cloning
      final clonedDraft = SafeDraftCloner.cloneInspectionDraft(
        source: srcDraft,
        newInspectionId: 'INSP-TARGET-002',
        newAssetId: 'CRANE-HOOK-02',
        recordedBy: 'inspector-alice',
        scheduledDate: now.add(const Duration(hours: 1)),
      );

      // Identity & State checks
      expect(clonedDraft.workPack.inspectionId, 'INSP-TARGET-002');
      expect(clonedDraft.workPack.rootAssetId, 'CRANE-HOOK-02');
      expect(clonedDraft.recordedBy, 'inspector-alice');
      expect(clonedDraft.status, InspectionStatus.inProgress);
      expect(clonedDraft.notes, 'Cloned from CRANE-HOOK-01');

      // Findings checks
      expect(clonedDraft.findings.length, 1);
      final clonedFinding = clonedDraft.findings['item-throat']!;
      expect(clonedFinding.inspectionId, 'INSP-TARGET-002');
      expect(clonedFinding.assetId, 'CRANE-HOOK-02');
      expect(clonedFinding.response, 'MEASURED');
      expect(clonedFinding.notes, 'Deformed past 10%');
      expect(clonedFinding.severity, Severity.critical);

      // ZERO EVIDENCE BLEED ASSERTION
      expect(clonedFinding.evidence, isNull);
    });

    test('cloneInspectionDraft rejects empty IDs or identical IDs', () {
      final now = DateTime.utc(2026, 10, 5, 12, 0);
      const context = TenantContext(
        tenantId: 't1',
        organizationId: 'org1',
        environment: 'TESTING',
      );
      final workPack = InspectionWorkPack(
        inspectionId: 'INSP-01',
        rootAssetId: 'ASSET-01',
        inspectionType: 'type',
        procedureVersion: 'v1',
        packageId: 'pkg',
        packageVersion: 1,
        schemaVersion: 1,
        packageHash: 'hash',
        scheduledDate: now,
        items: const [],
      );
      final src = InspectionDraft(
        context: context,
        workPack: workPack,
        recordedBy: 'actor',
        createdAt: now,
      );

      expect(
        () => SafeDraftCloner.cloneInspectionDraft(
          source: src,
          newInspectionId: 'INSP-01', // same ID
          newAssetId: 'ASSET-02',
          recordedBy: 'actor',
        ),
        throwsArgumentError,
      );

      expect(
        () => SafeDraftCloner.cloneInspectionDraft(
          source: src,
          newInspectionId: '',
          newAssetId: 'ASSET-02',
          recordedBy: 'actor',
        ),
        throwsArgumentError,
      );

      expect(
        () => SafeDraftCloner.cloneInspectionDraft(
          source: src,
          newInspectionId: 'INSP-02',
          newAssetId: '',
          recordedBy: 'actor',
        ),
        throwsArgumentError,
      );
    });
  });
}
