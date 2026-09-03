import 'evidence_crypto.dart';

class EvidenceScope {
  const EvidenceScope({required this.tenantId, required this.organizationId});

  final String tenantId;
  final String organizationId;
}

abstract interface class EvidenceStore {
  Future<void> put(EvidenceScope scope, EncryptedEvidence evidence);
  Future<EncryptedEvidence?> get(EvidenceScope scope, String evidenceId);
}

class InMemoryEvidenceStore implements EvidenceStore {
  final Map<String, EncryptedEvidence> _values = {};

  @override
  Future<EncryptedEvidence?> get(EvidenceScope scope, String evidenceId) async => _values[_key(scope, evidenceId)];

  @override
  Future<void> put(EvidenceScope scope, EncryptedEvidence evidence) async {
    if (scope.tenantId.trim().isEmpty || scope.organizationId.trim().isEmpty) {
      throw ArgumentError('tenant and organization are required');
    }
    _values[_key(scope, evidence.evidenceId)] = evidence;
  }

  String _key(EvidenceScope scope, String evidenceId) => '${scope.tenantId}:${scope.organizationId}:$evidenceId';
}
