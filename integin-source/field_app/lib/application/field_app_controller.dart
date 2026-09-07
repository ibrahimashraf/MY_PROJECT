import 'dart:math';

import 'package:flutter/foundation.dart';

import '../domain/inspection_draft.dart';
import '../domain/models.dart';
import '../outbox/outbox.dart';
import '../security/transaction_signer.dart';
import '../sync/sync_client.dart';
import '../sync/sync_guard.dart';

class FieldAppController extends ChangeNotifier {
  FieldAppController({
    required this.context,
    required this.deviceId,
    required this.userId,
    required this.deviceState,
    required this.authority,
    this.outboxStore,
    this.deviceSigner,
    this.deviceKeyId,
    this.syncClient,
  });

  final TenantContext context;
  final String deviceId;
  final String userId;
  final DeviceTrustState deviceState;
  final OfflineAuthority authority;
  final OutboxStore? outboxStore;
  final DeviceSigner? deviceSigner;
  final String? deviceKeyId;
  final SyncClient? syncClient;
  final List<OfflineMutation> _outbox = [];
  InspectionDraft? activeDraft;
  ConnectivityState connectivity = ConnectivityState.offline;
  DateTime? lastSuccessfulSync;
  String? lastError;
  OutboxSummary outboxSummary = const OutboxSummary();
  int _sequence = 0;
  int _lastAcceptedSequence = 0;

  List<OfflineMutation> get outbox => List.unmodifiable(_outbox);
  int get queuedCount => outboxSummary.pendingCount;
  int get failedCount => outboxSummary.failureCount;
  bool get isSyncConfigured => syncClient != null;

  Future<void> restoreOutbox() async {
    if (outboxStore == null) return;
    final entries = await outboxStore!.pending();
    outboxSummary = OutboxSummary.fromEntries(await outboxStore!.all());
    _outbox
      ..clear()
      ..addAll(entries.map((entry) => entry.mutation));
    for (final mutation in _outbox) {
      if (mutation.sequenceNumber > _sequence) {
        _sequence = mutation.sequenceNumber;
      }
    }
    for (final entry in await outboxStore!.all()) {
      if ((entry.state == OutboxState.applied ||
              entry.state == OutboxState.duplicate) &&
          entry.mutation.sequenceNumber > _lastAcceptedSequence) {
        _lastAcceptedSequence = entry.mutation.sequenceNumber;
      }
    }
    notifyListeners();
  }

  bool get canWorkOffline =>
      deviceState == DeviceTrustState.trusted &&
      authority.deviceId == deviceId &&
      authority.userId == userId &&
      authority.context.tenantId == context.tenantId &&
      authority.context.organizationId == context.organizationId &&
      authority.capabilities.contains('inspection.perform') &&
      authority.isValidAt(DateTime.now().toUtc());

  void beginInspection(InspectionWorkPack workPack) {
    if (!canWorkOffline) {
      lastError = 'Offline authority is missing, expired, or not trusted.';
      notifyListeners();
      return;
    }
    activeDraft = InspectionDraft(
      context: context,
      workPack: workPack,
      recordedBy: userId,
      createdAt: DateTime.now().toUtc(),
    );
    lastError = null;
    notifyListeners();
  }

  void recordResponse({
    required ChecklistItem item,
    required String response,
    String? note,
    Severity? severity,
  }) {
    activeDraft?.recordResponse(
      item: item,
      response: response,
      note: note,
      severity: severity,
    );
    notifyListeners();
  }

  Future<bool> queueForSync({String? notes}) async {
    final draft = activeDraft;
    if (draft == null) {
      lastError = 'Open an assigned inspection before submitting.';
      notifyListeners();
      return false;
    }
    final completeness = draft.validateLocally();
    if (!completeness.isComplete) {
      lastError = 'Complete every required checklist item before queuing.';
      notifyListeners();
      return false;
    }
    draft.notes = notes;
    draft.status = InspectionStatus.completed;
    _sequence += 1;
    final payload = draft.toPayload();
    final transactionId = _transactionId();
    final capturedAt = DateTime.now().toUtc();
    final hashOnly = OfflineMutation(
      transactionId: 'hash-only',
      context: context,
      deviceId: deviceId,
      userId: userId,
      sequenceNumber: _sequence,
      operation: 'hash-only',
      entityId: draft.workPack.inspectionId,
      payload: payload,
      capturedAt: capturedAt,
      authorityId: authority.id,
      authorityEpoch: authority.epoch,
      signature: '',
    );
    final signature = deviceSigner == null
        ? signTransaction(
            secret: 'local-development-key',
            transactionId: transactionId,
            tenantId: context.tenantId,
            deviceId: deviceId,
            userId: userId,
            sequenceNumber: _sequence,
            operation: 'InspectionSubmitted',
            payloadHash: hashOnly.payloadHash,
          )
        : await deviceSigner!.signV1(
            transactionId: transactionId,
            tenantId: context.tenantId,
            organizationId: context.organizationId,
            environment: context.environment,
            deviceId: deviceId,
            userId: userId,
            sequenceNumber: _sequence,
            operation: 'InspectionSubmitted',
            entityId: draft.workPack.inspectionId,
            payloadHash: hashOnly.payloadHash,
            authorityId: authority.id,
            authorityEpoch: authority.epoch,
            capturedAt: capturedAt,
            keyId: deviceKeyId ??
                (throw StateError(
                    'device key id is required for Ed25519 signing')),
          );
    final mutation = OfflineMutation(
      transactionId: transactionId,
      context: context,
      deviceId: deviceId,
      userId: userId,
      sequenceNumber: _sequence,
      operation: 'InspectionSubmitted',
      entityId: draft.workPack.inspectionId,
      payload: payload,
      capturedAt: capturedAt,
      authorityId: authority.id,
      authorityEpoch: authority.epoch,
      signatureAlgorithm: deviceSigner == null ? 'HMAC-SHA256' : 'Ed25519',
      keyId: deviceKeyId,
      signature: signature,
    );
    _outbox.add(mutation);
    if (outboxStore != null) {
      await outboxStore!.append(OutboxEntry(mutation: mutation));
      outboxSummary = OutboxSummary.fromEntries(await outboxStore!.all());
    }
    activeDraft = null;
    lastError = null;
    notifyListeners();
    return true;
  }

  Future<List<SyncOutcome>> flushOutbox({DateTime? at}) async {
    if (syncClient == null || outboxStore == null) {
      lastError = 'Live sync is not configured for this field session.';
      notifyListeners();
      return const [];
    }
    connectivity = ConnectivityState.syncing;
    lastError = null;
    notifyListeners();
    final pending = await outboxStore!.pending();
    final outcomes = await syncClient!.flush(
      guard: SyncGuard(
        context: context,
        deviceId: deviceId,
        userId: userId,
        deviceState: deviceState,
        authority: authority,
        lastSequence: _lastAcceptedSequence,
      ),
      at: at,
    );
    for (var index = 0;
        index < outcomes.length && index < pending.length;
        index += 1) {
      final outcome = outcomes[index];
      if ((outcome == SyncOutcome.applied ||
              outcome == SyncOutcome.duplicate) &&
          pending[index].mutation.sequenceNumber > _lastAcceptedSequence) {
        _lastAcceptedSequence = pending[index].mutation.sequenceNumber;
      }
    }
    outboxSummary = OutboxSummary.fromEntries(await outboxStore!.all());
    if (outcomes.any((outcome) =>
        outcome == SyncOutcome.securityFailure ||
        outcome == SyncOutcome.rejected)) {
      connectivity = ConnectivityState.blocked;
      lastError = 'One or more queued changes require operator review.';
    } else if (outcomes.any((outcome) =>
        outcome == SyncOutcome.queued ||
        outcome == SyncOutcome.held ||
        outcome == SyncOutcome.conflict)) {
      connectivity = ConnectivityState.degraded;
    } else {
      connectivity = ConnectivityState.online;
      if (outcomes.any((outcome) =>
          outcome == SyncOutcome.applied || outcome == SyncOutcome.duplicate)) {
        lastSuccessfulSync = DateTime.now().toUtc();
      }
    }
    notifyListeners();
    return outcomes;
  }

  String _transactionId() =>
      'tx-${DateTime.now().microsecondsSinceEpoch}-${Random().nextInt(9999)}';
}
