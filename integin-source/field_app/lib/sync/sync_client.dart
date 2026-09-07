import '../domain/models.dart';
import '../outbox/outbox.dart';
import 'sync_guard.dart';

class SyncResponse {
  const SyncResponse(
      {required this.outcome, this.reason, this.expectedSequence});

  final SyncOutcome outcome;
  final String? reason;
  final int? expectedSequence;
}

abstract interface class SyncTransport {
  Future<SyncResponse> submit(OfflineMutation mutation);
}

class SyncClient {
  const SyncClient({required this.store, required this.transport});

  final OutboxStore store;
  final SyncTransport transport;

  Future<List<SyncOutcome>> flush(
      {required SyncGuard guard, DateTime? at}) async {
    final results = <SyncOutcome>[];
    var acceptedSequence = guard.lastSequence;
    for (final entry in await store.pending()) {
      final localCheck = SyncGuard(
        context: guard.context,
        deviceId: guard.deviceId,
        userId: guard.userId,
        deviceState: guard.deviceState,
        authority: guard.authority,
        lastSequence: acceptedSequence,
      ).check(entry.mutation, at ?? DateTime.now().toUtc());
      if (!localCheck.allowed) {
        final outcome = (localCheck.reason ?? '').contains('sequence')
            ? SyncOutcome.held
            : SyncOutcome.securityFailure;
        await store.mark(entry, _stateFor(outcome), error: localCheck.reason);
        results.add(outcome);
        continue;
      }

      await store.mark(entry, OutboxState.uploading);
      SyncResponse response;
      try {
        response = await transport.submit(entry.mutation);
      } catch (error) {
        await store.mark(entry, OutboxState.queued,
            error: 'Transport unavailable; will retry. $error');
        results.add(SyncOutcome.queued);
        continue;
      }
      await store.mark(entry, _stateFor(response.outcome),
          error: response.reason);
      results.add(response.outcome);
      if (response.outcome == SyncOutcome.applied ||
          response.outcome == SyncOutcome.duplicate) {
        acceptedSequence = entry.mutation.sequenceNumber;
      }
    }
    return results;
  }

  OutboxState _stateFor(SyncOutcome outcome) => switch (outcome) {
        SyncOutcome.queued => OutboxState.queued,
        SyncOutcome.applied => OutboxState.applied,
        SyncOutcome.duplicate => OutboxState.duplicate,
        SyncOutcome.held => OutboxState.held,
        SyncOutcome.rejected => OutboxState.rejected,
        SyncOutcome.conflict => OutboxState.conflict,
        SyncOutcome.securityFailure => OutboxState.securityFailure,
      };
}
