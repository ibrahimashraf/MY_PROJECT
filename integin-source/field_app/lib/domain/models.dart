import 'dart:convert';

import 'package:crypto/crypto.dart';

enum ConnectivityState { online, offline, syncing, degraded, blocked }

enum DeviceTrustState { pending, trusted, restricted, locked, revoked, retired }

enum InspectionStatus {
  scheduled,
  assigned,
  inProgress,
  completed,
  pendingReview,
  returned,
  blocked,
}

enum Severity { advisory, minor, major, critical }

enum SyncOutcome {
  queued,
  applied,
  duplicate,
  held,
  rejected,
  conflict,
  securityFailure,
}

class TenantContext {
  const TenantContext({
    required this.tenantId,
    required this.organizationId,
    required this.environment,
  });

  final String tenantId;
  final String organizationId;
  final String environment;

  factory TenantContext.fromJson(Map<String, Object?> json) => TenantContext(
        tenantId: json['tenant_id'] as String,
        organizationId: json['organization_id'] as String,
        environment: json['environment'] as String,
      );

  bool get isValid =>
      tenantId.trim().isNotEmpty &&
      organizationId.trim().isNotEmpty &&
      (environment == 'TESTING' || environment == 'LIVE');

  Map<String, Object?> toJson() => {
        'tenant_id': tenantId,
        'organization_id': organizationId,
        'environment': environment,
      };
}

class OfflineAuthority {
  const OfflineAuthority({
    required this.id,
    required this.deviceId,
    required this.context,
    required this.userId,
    required this.epoch,
    required this.scopes,
    required this.capabilities,
    required this.procedureVersion,
    required this.issuedAt,
    required this.expiresAt,
    required this.signature,
  });

  final String id;
  final String deviceId;
  final TenantContext context;
  final String userId;
  final int epoch;
  final List<String> scopes;
  final List<String> capabilities;
  final String procedureVersion;
  final DateTime issuedAt;
  final DateTime expiresAt;
  final String signature;

  bool isValidAt(DateTime at) =>
      context.isValid &&
      at.toUtc().isAfter(issuedAt.toUtc()) &&
      at.toUtc().isBefore(expiresAt.toUtc());

  Map<String, Object?> toJson() => {
        'id': id,
        'device_id': deviceId,
        ...context.toJson(),
        'user_id': userId,
        'epoch': epoch,
        'scopes': [...scopes]..sort(),
        'capabilities': [...capabilities]..sort(),
        'procedure_version': procedureVersion,
        'issued_at': issuedAt.toUtc().toIso8601String(),
        'expires_at': expiresAt.toUtc().toIso8601String(),
        'signature': signature,
      };
}

class EvidenceReference {
  const EvidenceReference({
    required this.id,
    required this.kind,
    required this.capturedAt,
    required this.localUri,
    this.sha256,
  });

  final String id;
  final String kind;
  final DateTime capturedAt;
  final String localUri;
  final String? sha256;

  Map<String, Object?> toJson() => {
        'id': id,
        'kind': kind,
        'captured_at': capturedAt.toUtc().toIso8601String(),
        'local_uri': localUri,
        if (sha256 != null) 'sha256': sha256,
      };
}

class FindingDraft {
  const FindingDraft({
    required this.id,
    required this.inspectionId,
    required this.assetId,
    required this.sectionId,
    required this.itemId,
    required this.itemPrompt,
    required this.response,
    required this.recordedBy,
    required this.recordedAt,
    this.measuredValue,
    this.measuredUnit,
    this.severity,
    this.notes,
    this.evidence,
  });

  final String id;
  final String inspectionId;
  final String assetId;
  final String sectionId;
  final String itemId;
  final String itemPrompt;
  final String response;
  final String recordedBy;
  final DateTime recordedAt;
  final double? measuredValue;
  final String? measuredUnit;
  final Severity? severity;
  final String? notes;
  final List<EvidenceReference>? evidence;

  Map<String, Object?> toJson() => {
        'id': id,
        'inspection_id': inspectionId,
        'asset_id': assetId,
        'section_id': sectionId,
        'item_id': itemId,
        'item_prompt': itemPrompt,
        'response': response,
        'recorded_by': recordedBy,
        'recorded_at': recordedAt.toUtc().toIso8601String(),
        if (measuredValue != null) 'measured_value': measuredValue,
        if (measuredUnit != null) 'measured_unit': measuredUnit,
        if (severity != null) 'severity': severity!.name.toUpperCase(),
        if (notes != null) 'notes': notes,
        if (evidence != null)
          'evidence_refs': evidence!.map((item) => item.toJson()).toList(),
      };
}

class OfflineMutation {
  OfflineMutation({
    required this.transactionId,
    required this.context,
    required this.deviceId,
    required this.userId,
    required this.sequenceNumber,
    required this.operation,
    required this.entityId,
    required this.payload,
    required this.capturedAt,
    required this.authorityId,
    required this.authorityEpoch,
    required this.signature,
    this.protocolVersion = 'v1',
    this.signatureAlgorithm = 'HMAC-SHA256',
    this.keyId,
  }) : payloadHash = sha256.convert(utf8.encode(canonicalJson(payload))).toString();

  final String protocolVersion;
  final String transactionId;
  final TenantContext context;
  final String deviceId;
  final String userId;
  final int sequenceNumber;
  final String operation;
  final String entityId;
  final Map<String, Object?> payload;
  final DateTime capturedAt;
  final String authorityId;
  final int authorityEpoch;
  final String signatureAlgorithm;
  final String? keyId;
  final String signature;
  final String payloadHash;

  factory OfflineMutation.fromJson(Map<String, Object?> json) {
    final payload = Map<String, Object?>.from(json['payload'] as Map);
    final mutation = OfflineMutation(
      transactionId: json['transaction_id'] as String,
      protocolVersion: json['protocol_version'] as String? ?? 'v1',
      context: TenantContext.fromJson(json),
      deviceId: json['device_id'] as String,
      userId: json['user_id'] as String,
      sequenceNumber: json['sequence_number'] as int,
      operation: json['operation'] as String,
      entityId: json['entity_id'] as String,
      payload: payload,
      capturedAt: DateTime.parse(json['captured_at'] as String),
      authorityId: json['authority_id'] as String,
      authorityEpoch: json['authority_epoch'] as int,
      signatureAlgorithm: json['signature_algorithm'] as String? ?? 'HMAC-SHA256',
      keyId: json['key_id'] as String?,
      signature: json['signature'] as String,
    );
    if (mutation.payloadHash != json['payload_hash']) {
      throw const FormatException('offline mutation payload hash mismatch');
    }
    return mutation;
  }

  Map<String, Object?> toJson() => {
        'protocol_version': protocolVersion,
        'transaction_id': transactionId,
        ...context.toJson(),
        'device_id': deviceId,
        'user_id': userId,
        'sequence_number': sequenceNumber,
        'operation': operation,
        'entity_id': entityId,
        'payload': payload,
        'payload_hash': payloadHash,
        'captured_at': capturedAt.toUtc().toIso8601String(),
        'authority_id': authorityId,
        'authority_epoch': authorityEpoch,
        'signature_algorithm': signatureAlgorithm,
        if (keyId != null) 'key_id': keyId,
        'signature': signature,
      };
}

String canonicalJson(Object? value) {
  Object? normalize(Object? input) {
    if (input is Map) {
      final keys = input.keys.map((key) => key.toString()).toList()..sort();
      return <String, Object?>{
        for (final key in keys) key: normalize(input[key]),
      };
    }
    if (input is Iterable) return input.map(normalize).toList(growable: false);
    if (input is DateTime) return input.toUtc().toIso8601String();
    return input;
  }

  return jsonEncode(normalize(value));
}
