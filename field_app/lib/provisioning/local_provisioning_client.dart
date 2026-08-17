import 'dart:convert';

import 'package:http/http.dart' as http;

import '../domain/models.dart';
import '../security/transaction_signer.dart';

class ProvisionedFieldSession {
  const ProvisionedFieldSession({
    required this.context,
    required this.deviceId,
    required this.userId,
    required this.authority,
  });

  final TenantContext context;
  final String deviceId;
  final String userId;
  final OfflineAuthority authority;

  factory ProvisionedFieldSession.fromJson(Map<String, Object?> json) {
    final authorityJSON = Map<String, Object?>.from(json['authority'] as Map);
    final context = TenantContext(
      tenantId: json['tenant_id'] as String,
      organizationId: json['organization_id'] as String,
      environment: json['environment'] as String,
    );
    final scopes = List<String>.from(authorityJSON['scopes'] as List);
    return ProvisionedFieldSession(
      context: context,
      deviceId: json['device_id'] as String,
      userId: json['user_id'] as String,
      authority: OfflineAuthority(
        id: authorityJSON['authority_id'] as String,
        deviceId: json['device_id'] as String,
        context: context,
        userId: json['user_id'] as String,
        epoch: authorityJSON['authority_epoch'] as int,
        scopes: scopes,
        capabilities: scopes,
        procedureVersion: authorityJSON['procedure_version'] as String,
        issuedAt: DateTime.parse(authorityJSON['issued_at'] as String),
        expiresAt: DateTime.parse(authorityJSON['expires_at'] as String),
        signature: authorityJSON['signature'] as String,
      ),
    );
  }
}

/// Local-integration provisioning client. It never exports a device private key.
class LocalProvisioningClient {
  LocalProvisioningClient({required this.endpoint, http.Client? client})
      : _client = client ?? http.Client();

  final Uri endpoint;
  final http.Client _client;

  Future<ProvisionedFieldSession> provision(DeviceSigner signer) async {
    final keyId = signer.keyId;
    if (keyId == null || keyId.isEmpty) {
      throw StateError('a provisioned device signer requires a key id');
    }
    final deviceId = 'field-${keyId.substring(0, 32)}';
    final response = await _client.post(
      endpoint,
      headers: const {'Content-Type': 'application/json'},
      body: jsonEncode({
        'device_id': deviceId,
        'key_id': keyId,
        'public_key': await signer.publicKeyBase64(),
      }),
    );
    if (response.statusCode != 200) {
      throw StateError('local device provisioning failed: HTTP ${response.statusCode}');
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! Map) {
      throw StateError('local device provisioning returned invalid JSON');
    }
    return ProvisionedFieldSession.fromJson(Map<String, Object?>.from(decoded));
  }
}
