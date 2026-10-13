import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/core/coalesce.dart';
import 'package:integin_field_app/core/draft_store.dart';

void main() {
  group('Coalescing and Local Drafts Integration', () {
    test('CoalescingController debounces fast inputs into single emission', () async {
      final controller = CoalescingController(
        debounceWindow: const Duration(milliseconds: 50),
      );

      int commitCount = 0;
      Map<String, dynamic>? lastPayload;

      void onCommit(String key, Map<String, dynamic> payload) {
        commitCount++;
        lastPayload = payload;
      }

      // 1. Emit rapid sequence
      controller.emit(
        coalesceKey: 'crane.inspection',
        payload: {'swl': 100},
        onCommit: onCommit,
      );
      controller.emit(
        coalesceKey: 'crane.inspection',
        payload: {'swl': 105},
        onCommit: onCommit,
      );
      controller.emit(
        coalesceKey: 'crane.inspection',
        payload: {'swl': 110},
        onCommit: onCommit,
      );

      // Immediately before debounce, 0 commits
      expect(commitCount, 0);

      // Wait past debounce duration
      await Future.delayed(const Duration(milliseconds: 80));

      // Exactly 1 batched commit occurred with the latest state
      expect(commitCount, 1);
      expect(lastPayload?['swl'], 110);
    });

    test('InMemoryDraftStore saves and marks drafts as synced', () async {
      final store = InMemoryDraftStore();
      final draft = DraftRecord(
        id: 'draft-liebherr-101',
        tenantId: 'tenant-aramco',
        commandId: 'asset.inspection.create',
        payload: {'chassis': 'SN-9921', 'status': 'PASS'},
        localRevision: 1,
        isSynced: false,
        updatedAt: DateTime.now().toUtc(),
      );

      await store.saveDraft(draft);

      final retrieved = await store.getDraft('draft-liebherr-101');
      expect(retrieved, isNotNull);
      expect(retrieved?.payload['chassis'], 'SN-9921');
      expect(retrieved?.isSynced, isFalse);

      final unsynced = await store.getUnsyncedDrafts('tenant-aramco');
      expect(unsynced.length, 1);

      // Mark synced
      await store.markSynced('draft-liebherr-101');

      final syncedDraft = await store.getDraft('draft-liebherr-101');
      expect(syncedDraft?.isSynced, isTrue);
      expect(syncedDraft?.localRevision, 2);

      final unsyncedAfter = await store.getUnsyncedDrafts('tenant-aramco');
      expect(unsyncedAfter.isEmpty, isTrue);
    });
  });
}
