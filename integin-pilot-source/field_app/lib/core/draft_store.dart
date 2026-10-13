import 'dart:convert';

class DraftRecord {
  final String id;
  final String tenantId;
  final String commandId;
  final Map<String, dynamic> payload;
  final int localRevision;
  final bool isSynced;
  final DateTime updatedAt;

  DraftRecord({
    required this.id,
    required this.tenantId,
    required this.commandId,
    required this.payload,
    required this.localRevision,
    required this.isSynced,
    required this.updatedAt,
  });

  Map<String, dynamic> toMap() {
    return {
      'id': id,
      'tenant_id': tenantId,
      'command_id': commandId,
      'payload_json': jsonEncode(payload),
      'local_revision': localRevision,
      'is_synced': isSynced ? 1 : 0,
      'updated_at': updatedAt.toIso8601String(),
    };
  }

  factory DraftRecord.fromMap(Map<String, dynamic> map) {
    return DraftRecord(
      id: map['id'] as String,
      tenantId: map['tenant_id'] as String,
      commandId: map['command_id'] as String,
      payload: jsonDecode(map['payload_json'] as String) as Map<String, dynamic>,
      localRevision: map['local_revision'] as int,
      isSynced: (map['is_synced'] as int) == 1,
      updatedAt: DateTime.parse(map['updated_at'] as String),
    );
  }
}

abstract class LocalDraftStore {
  Future<void> saveDraft(DraftRecord draft);
  Future<DraftRecord?> getDraft(String id);
  Future<List<DraftRecord>> getUnsyncedDrafts(String tenantId);
  Future<void> markSynced(String id);
}

/// InMemoryDraftStore is a lightweight offline-first cache implementation.
class InMemoryDraftStore implements LocalDraftStore {
  final Map<String, DraftRecord> _storage = {};

  @override
  Future<void> saveDraft(DraftRecord draft) async {
    _storage[draft.id] = draft;
  }

  @override
  Future<DraftRecord?> getDraft(String id) async {
    return _storage[id];
  }

  @override
  Future<List<DraftRecord>> getUnsyncedDrafts(String tenantId) async {
    return _storage.values
        .where((d) => d.tenantId == tenantId && !d.isSynced)
        .toList();
  }

  @override
  Future<void> markSynced(String id) async {
    final existing = _storage[id];
    if (existing != null) {
      _storage[id] = DraftRecord(
        id: existing.id,
        tenantId: existing.tenantId,
        commandId: existing.commandId,
        payload: existing.payload,
        localRevision: existing.localRevision + 1,
        isSynced: true,
        updatedAt: DateTime.now().toUtc(),
      );
    }
  }
}
