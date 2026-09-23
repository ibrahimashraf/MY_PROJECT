import 'dart:math';

import 'package:flutter/foundation.dart';

import '../advisory/pilot_advisory_client.dart';
import '../domain/inspection_draft.dart';
import '../workpackages/package_compatibility.dart';
import '../domain/models.dart';
import '../outbox/merkle_hash_chain.dart';
import '../outbox/outbox.dart';
import '../security/transaction_signer.dart';
import '../sync/event_stream_client.dart';
import '../sync/sync_client.dart';
import '../sync/sync_guard.dart';
import '../sync/tus_client.dart';

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
    this.eventStreamClient,
    this.advisoryClient,
    this.tusClient,
    this.trace,
  }) {
    _initStream();
  }

  final TenantContext context;
  final String deviceId;
  final String userId;
  final DeviceTrustState deviceState;
  final OfflineAuthority authority;
  final OutboxStore? outboxStore;
  final DeviceSigner? deviceSigner;
  final String? deviceKeyId;
  final SyncClient? syncClient;
  final EventStreamClient? eventStreamClient;
  final TusClient? tusClient;

  /// Present only in the isolated pilot build; advisory data has no authority.
  final PilotAdvisoryClient? advisoryClient;

  /// Optional pilot diagnostic sink. It records only outcome classes and counts.
  final void Function(String)? trace;
  final List<OfflineMutation> _outbox = [];
  final List<StationEvent> recentEvents = [];
  InspectionDraft? activeDraft;
  ConnectivityState connectivity = ConnectivityState.offline;
  DateTime? lastSuccessfulSync;
  String? lastError;
  PilotAdvisoryResult? advisoryResult;
  String? advisoryError;
  bool advisoryLoading = false;
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
    trace?.call(
        'restore-reason-class-${_diagnosticReasonClass(await outboxStore!.all())}');
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

  /// Re-binds every unacknowledged outbox entry to the current device identity.
  ///
  /// Authority packages rotate on every fresh provision (new id, ~30 minute
  /// lifetime). Entries queued under a previous device id or authority fail
  /// [SyncGuard] locally and would otherwise rest in terminal failure states
  /// forever — the inspector's work silently lost until a cache clear wipes
  /// it. Reconciliation re-sequences them contiguously from
  /// [_lastAcceptedSequence], re-signs with the current [deviceSigner],
  /// re-seals the tamper-evident chain in stored order, and returns them to
  /// `queued` so the next flush can deliver them. Transaction ids are stable
  /// across the migration, so server replays stay idempotent (applied becomes
  /// a harmless duplicate). Returns the number of migrated entries.
  Future<int> reconcileOutboxIdentities() async {
    final store = outboxStore;
    final signer = deviceSigner;
    final effectiveKeyId = deviceKeyId ?? signer?.keyId;
    if (store == null ||
        signer == null ||
        effectiveKeyId == null ||
        effectiveKeyId.isEmpty) {
      return 0;
    }
    final entries = await store.all();
    final ordered = entries.toList()
      ..sort((a, b) {
        final byTime =
            a.mutation.capturedAt.compareTo(b.mutation.capturedAt);
        if (byTime != 0) {
          return byTime;
        }
        return a.mutation.sequenceNumber.compareTo(b.mutation.sequenceNumber);
      });
    final chain = CryptographicMerkleHashChain();
    var cursor = _lastAcceptedSequence;
    var migrated = 0;
    for (final entry in ordered) {
      final mutation = entry.mutation;
      if (entry.state.isAcknowledged) {
        chain.seal(mutation);
        if (mutation.sequenceNumber > _lastAcceptedSequence) {
          _lastAcceptedSequence = mutation.sequenceNumber;
        }
        continue;
      }
      if (mutation.context.tenantId != context.tenantId ||
          mutation.context.organizationId != context.organizationId) {
        chain.seal(mutation);
        continue;
      }
      if (mutation.userId != userId) {
        // Never re-attribute another inspector's work to the current user:
        // submitting it under a different identity would falsify the
        // recorded-by chain. Such entries stay put for operator review.
        chain.seal(mutation);
        continue;
      }
      final expectedSequence = cursor + 1;
      final isCurrent = mutation.deviceId == deviceId &&
          mutation.userId == userId &&
          mutation.authorityId == authority.id &&
          mutation.authorityEpoch == authority.epoch &&
          mutation.sequenceNumber == expectedSequence;
      if (isCurrent) {
        chain.seal(mutation);
        cursor = expectedSequence;
        continue;
      }
      cursor = expectedSequence;
      String signature;
      try {
        signature = await signer.signV1(
          transactionId: mutation.transactionId,
          tenantId: context.tenantId,
          organizationId: context.organizationId,
          environment: context.environment,
          deviceId: deviceId,
          userId: userId,
          sequenceNumber: cursor,
          operation: mutation.operation,
          entityId: mutation.entityId,
          payloadHash: mutation.payloadHash,
          authorityId: authority.id,
          authorityEpoch: authority.epoch,
          capturedAt: mutation.capturedAt,
          keyId: effectiveKeyId,
        );
      } catch (error) {
        // Fail closed mid-migration: already-replaced entries are durable and
        // the remainder retry on the next launch. Never strand startup here.
        lastError = 'Identity reconciliation paused: $error';
        notifyListeners();
        break;
      }
      final rebound = OfflineMutation(
        transactionId: mutation.transactionId,
        context: mutation.context,
        deviceId: deviceId,
        userId: userId,
        sequenceNumber: cursor,
        operation: mutation.operation,
        entityId: mutation.entityId,
        payload: mutation.payload,
        capturedAt: mutation.capturedAt,
        authorityId: authority.id,
        authorityEpoch: authority.epoch,
        signatureAlgorithm: 'Ed25519',
        keyId: effectiveKeyId,
        signature: signature,
      );
      final replacement = OutboxEntry(
        mutation: rebound,
        chainHash: chain.seal(rebound),
      )
        ..attempts = entry.attempts
        ..lastError =
            'Re-bound to current device identity; previously: ${entry.lastError ?? 'pending'}.';
      await store.replace(replacement);
      migrated += 1;
    }
    if (migrated > 0) {
      _sequence = cursor;
      final pending = await store.pending();
      _outbox
        ..clear()
        ..addAll(pending.map((entry) => entry.mutation));
      outboxSummary = OutboxSummary.fromEntries(await store.all());
      trace?.call('reconcile-migrated-$migrated');
      notifyListeners();
    }
    return migrated;
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

  Future<bool> queueForSync(
      {String? notes,
      PackageCompatibilityDecision? packageCompatibility}) async {
    final draft = activeDraft;
    if (draft == null) {
      lastError = 'Open an assigned inspection before submitting.';
      notifyListeners();
      return false;
    }
    if (packageCompatibility != null &&
        !packageCompatibility.allowsAuthoritativeSync) {
      lastError = packageCompatibility.userMessage;
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
    // SECURITY: a device-bound signer is mandatory for every submitted transaction.
    // There is deliberately no static local-key fallback; enrollment must supply this signer.
    final signer = deviceSigner;
    if (signer == null) {
      throw StateError(
          'Device signer unavailable; device enrollment is required before submission.');
    }
    final signature = await signer.signV1(
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
          (throw StateError('device key id is required for Ed25519 signing')),
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
    // Re-derive from stored states rather than pairing outcomes to a
    // separately fetched pending list by index: store ordering across two
    // reads is not a contract any caller should rely on.
    for (final entry in await outboxStore!.all()) {
      if ((entry.state == OutboxState.applied ||
              entry.state == OutboxState.duplicate) &&
          entry.mutation.sequenceNumber > _lastAcceptedSequence) {
        _lastAcceptedSequence = entry.mutation.sequenceNumber;
      }
    }
    outboxSummary = OutboxSummary.fromEntries(await outboxStore!.all());
    trace?.call(
        'sync-outcomes-${outcomes.map((outcome) => outcome.name).join('-')}');
    trace?.call(
      'sync-summary-pending-${outboxSummary.pendingCount}-failures-${outboxSummary.failureCount}',
    );
    trace?.call(
        'sync-reason-class-${_diagnosticReasonClass(await outboxStore!.all())}');
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

  bool get hasAuthoritativeOutcome =>
      outboxSummary.applied > 0 || outboxSummary.duplicate > 0;

  /// Replays the exact accepted transaction through the server idempotency path.
  Future<void> replayLatestAuthoritative() async {
    if (syncClient == null || outboxStore == null) {
      lastError = 'Replay is unavailable for this field session.';
      notifyListeners();
      return;
    }
    final entries = await outboxStore!.all();
    OutboxEntry? entry;
    for (final candidate in entries.reversed) {
      if (candidate.state == OutboxState.applied ||
          candidate.state == OutboxState.duplicate) {
        entry = candidate;
        break;
      }
    }
    if (entry == null) {
      lastError = 'No authoritative transaction is available for replay.';
      notifyListeners();
      return;
    }
    connectivity = ConnectivityState.syncing;
    lastError = null;
    notifyListeners();
    final outcome = await syncClient!.replay(entry);
    outboxSummary = OutboxSummary.fromEntries(await outboxStore!.all());
    if (outcome == SyncOutcome.applied || outcome == SyncOutcome.duplicate) {
      connectivity = ConnectivityState.online;
      lastSuccessfulSync = DateTime.now().toUtc();
    } else if (outcome == SyncOutcome.rejected ||
        outcome == SyncOutcome.securityFailure) {
      connectivity = ConnectivityState.blocked;
      lastError = 'Replay requires operator review.';
    } else {
      connectivity = ConnectivityState.degraded;
    }
    trace?.call('pilot-replay-outcome-${outcome.name}');
    notifyListeners();
  }

  /// Requests a monitoring signal only; it never mutates primary field state.
  Future<void> requestPilotAdvisory(String inspectionId) async {
    final client = advisoryClient;
    if (client == null) {
      advisoryError = 'Pilot advisory service is not configured.';
      notifyListeners();
      return;
    }
    advisoryLoading = true;
    advisoryError = null;
    notifyListeners();
    try {
      advisoryResult = await client.requestMonitoring(
        tenantId: context.tenantId,
        inspectionId: inspectionId,
        evidenceRefs: const [],
      );
      trace?.call('pilot-advisory-nonblocking');
    } catch (_) {
      advisoryError =
          'Pilot advisory request is unavailable; primary work remains unchanged.';
    } finally {
      advisoryLoading = false;
      notifyListeners();
    }
  }

  /// Maps a whitelisted pilot error category without exposing diagnostic text.
  String _diagnosticReasonClass(Iterable<OutboxEntry> entries) {
    final reasons =
        entries.map((entry) => entry.lastError?.toLowerCase() ?? '').join(' ');
    for (final marker in const [
      'sequence',
      'authority',
      'signature',
      'device',
      'tenant',
      'organization',
      'environment',
      'payload',
      'transport',
      'http'
    ]) {
      if (reasons.contains(marker)) return marker;
    }
    return reasons.isEmpty ? 'none' : 'other';
  }

  void _initStream() {
    if (eventStreamClient == null) return;
    eventStreamClient!.events.listen((event) {
      recentEvents.insert(0, event);
      if (recentEvents.length > 20) {
        recentEvents.removeLast();
      }
      notifyListeners();
    }, onError: (_) {
      // Stream error does not break local app operation
    });
    eventStreamClient!.connect().catchError((_) {
      // Offline fallback: ignore stream connection failure
    });
  }

  @override
  void dispose() {
    outboxStore?.close();
    eventStreamClient?.dispose();
    super.dispose();
  }

  String _transactionId() =>
      'tx-${DateTime.now().microsecondsSinceEpoch}-${Random().nextInt(9999)}';
}

