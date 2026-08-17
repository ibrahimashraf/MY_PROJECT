import '../domain/models.dart';

enum OutboxState { queued, uploading, applied, duplicate, held, rejected, conflict, securityFailure }

extension OutboxStateSemantics on OutboxState {
  bool get isPending => this == OutboxState.queued || this == OutboxState.uploading || this == OutboxState.held;
  bool get isAcknowledged => this == OutboxState.applied || this == OutboxState.duplicate;
  bool get isFailure => this == OutboxState.rejected || this == OutboxState.conflict || this == OutboxState.securityFailure;
}

class OutboxEntry {
  OutboxEntry({required this.mutation}) : state = OutboxState.queued;

  OutboxEntry.fromJson(Map<String, Object?> json)
      : mutation = OfflineMutation.fromJson(
          Map<String, Object?>.from(json['mutation'] as Map),
        ),
        state = OutboxState.values.firstWhere(
          (value) => value.name.toUpperCase() == json['state'],
          orElse: () => OutboxState.queued,
        ),
        attempts = json['attempts'] as int? ?? 0,
        lastError = json['last_error'] as String?,
        acknowledgedAt = json['acknowledged_at'] == null
            ? null
            : DateTime.parse(json['acknowledged_at'] as String);

  final OfflineMutation mutation;
  OutboxState state;
  int attempts = 0;
  String? lastError;
  DateTime? acknowledgedAt;

  bool get isPending => state.isPending;

  Map<String, Object?> toJson() => {
        'mutation': mutation.toJson(),
        'state': state.name.toUpperCase(),
        'attempts': attempts,
        if (lastError != null) 'last_error': lastError,
        if (acknowledgedAt != null) 'acknowledged_at': acknowledgedAt!.toUtc().toIso8601String(),
      };
}

class OutboxSummary {
  const OutboxSummary({
    this.queued = 0,
    this.uploading = 0,
    this.held = 0,
    this.applied = 0,
    this.duplicate = 0,
    this.rejected = 0,
    this.conflict = 0,
    this.securityFailure = 0,
  });

  factory OutboxSummary.fromEntries(Iterable<OutboxEntry> entries) {
    var summary = const OutboxSummary();
    for (final entry in entries) {
      summary = summary.increment(entry.state);
    }
    return summary;
  }

  final int queued;
  final int uploading;
  final int held;
  final int applied;
  final int duplicate;
  final int rejected;
  final int conflict;
  final int securityFailure;

  int get pendingCount => queued + uploading + held;
  int get failureCount => rejected + conflict + securityFailure;

  OutboxSummary increment(OutboxState state) => switch (state) {
        OutboxState.queued => OutboxSummary(queued: queued + 1, uploading: uploading, held: held, applied: applied, duplicate: duplicate, rejected: rejected, conflict: conflict, securityFailure: securityFailure),
        OutboxState.uploading => OutboxSummary(queued: queued, uploading: uploading + 1, held: held, applied: applied, duplicate: duplicate, rejected: rejected, conflict: conflict, securityFailure: securityFailure),
        OutboxState.held => OutboxSummary(queued: queued, uploading: uploading, held: held + 1, applied: applied, duplicate: duplicate, rejected: rejected, conflict: conflict, securityFailure: securityFailure),
        OutboxState.applied => OutboxSummary(queued: queued, uploading: uploading, held: held, applied: applied + 1, duplicate: duplicate, rejected: rejected, conflict: conflict, securityFailure: securityFailure),
        OutboxState.duplicate => OutboxSummary(queued: queued, uploading: uploading, held: held, applied: applied, duplicate: duplicate + 1, rejected: rejected, conflict: conflict, securityFailure: securityFailure),
        OutboxState.rejected => OutboxSummary(queued: queued, uploading: uploading, held: held, applied: applied, duplicate: duplicate, rejected: rejected + 1, conflict: conflict, securityFailure: securityFailure),
        OutboxState.conflict => OutboxSummary(queued: queued, uploading: uploading, held: held, applied: applied, duplicate: duplicate, rejected: rejected, conflict: conflict + 1, securityFailure: securityFailure),
        OutboxState.securityFailure => OutboxSummary(queued: queued, uploading: uploading, held: held, applied: applied, duplicate: duplicate, rejected: rejected, conflict: conflict, securityFailure: securityFailure + 1),
      };
}

abstract interface class OutboxStore {
  Future<void> append(OutboxEntry entry);
  Future<List<OutboxEntry>> all();
  Future<List<OutboxEntry>> pending();
  Future<void> mark(OutboxEntry entry, OutboxState state, {String? error});
}

class InMemoryOutboxStore implements OutboxStore {
  final List<OutboxEntry> _entries = [];

  @override
  Future<void> append(OutboxEntry entry) async {
    if (_entries.any((candidate) => candidate.mutation.transactionId == entry.mutation.transactionId)) {
      return;
    }
    _entries.add(entry);
  }

  @override
  Future<List<OutboxEntry>> all() async => List.unmodifiable(_entries);

  @override
  Future<List<OutboxEntry>> pending() async => List.unmodifiable(_entries.where((entry) => entry.isPending));

  @override
  Future<void> mark(OutboxEntry entry, OutboxState state, {String? error}) async {
    entry.state = state;
    entry.attempts += 1;
    entry.lastError = error;
    if (state == OutboxState.applied || state == OutboxState.duplicate) {
      entry.acknowledgedAt = DateTime.now().toUtc();
    }
  }
}
