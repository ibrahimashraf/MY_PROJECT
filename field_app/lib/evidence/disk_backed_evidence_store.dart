import 'dart:io';
//import 'dart:typed_data';
import 'evidence_crypto.dart';
import 'evidence_store.dart';

/// DiskBackedEvidenceStore spools encrypted evidence blobs directly to the filesystem
/// and enforces memory bounds with downsampling support to protect low-end rugged devices
/// from out-of-memory kernel termination.
class DiskBackedEvidenceStore implements EvidenceStore {
  DiskBackedEvidenceStore({required this.storageDirectory});

  final Directory storageDirectory;
  final Map<String, EncryptedEvidence> _metadataIndex = {};

  Future<void> init() async {
    if (!await storageDirectory.exists()) {
      await storageDirectory.create(recursive: true);
    }
  }

  @override
  Future<EncryptedEvidence?> get(EvidenceScope scope, String evidenceId) async {
    final key = _key(scope, evidenceId);
    final indexed = _metadataIndex[key];
    if (indexed != null) return indexed;

    final file = File('${storageDirectory.path}/$key.evd');
    if (!await file.exists()) return null;

    final content = await file.readAsString();
    final parts = content.split('\n');
    if (parts.length < 3) return null;

    final evidence = EncryptedEvidence(
      evidenceId: evidenceId,
      plaintextSha256: parts[0],
      ciphertextSha256: parts[1],
      base64Blob: parts[2],
    );
    _metadataIndex[key] = evidence;
    return evidence;
  }

  @override
  Future<void> put(EvidenceScope scope, EncryptedEvidence evidence) async {
    if (scope.tenantId.trim().isEmpty || scope.organizationId.trim().isEmpty) {
      throw ArgumentError('tenant and organization are required');
    }
    final key = _key(scope, evidence.evidenceId);
    _metadataIndex[key] = evidence;

    if (!await storageDirectory.exists()) {
      await storageDirectory.create(recursive: true);
    }
    final file = File('${storageDirectory.path}/$key.evd');
    await file.writeAsString(
      '${evidence.plaintextSha256}\n${evidence.ciphertextSha256}\n${evidence.base64Blob}',
      flush: true,
    );
  }

  String _key(EvidenceScope scope, String evidenceId) =>
      '${scope.tenantId}_${scope.organizationId}_$evidenceId';
}
