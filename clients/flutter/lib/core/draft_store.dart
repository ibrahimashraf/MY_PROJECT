import 'dart:convert';

/// LocalDraftStore models zero-latency local drafts with synchronization metadata
/// matching the ArtCraft embedded offline draft pattern.
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

/// Abstract contract for local SQLite or In-Memory Draft storage.
abstract class LocalDraftStore {
  Future<void> saveDraft(DraftRecord draft);
  Future<DraftRecord?> getDraft(String id);
  Future<List<DraftRecord>> getUnsyncedDrafts(String tenantId);
  Future<void> markSynced(String id);
}
