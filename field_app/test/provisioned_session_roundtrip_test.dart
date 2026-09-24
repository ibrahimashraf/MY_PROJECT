import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/provisioning/local_provisioning_client.dart';

void main() {
  test('ProvisionedFieldSession round-trips through json', () {
    final json = {
      'tenant_id': 'INTEGIN-integration-tenant',
      'organization_id': 'org-phone-01',
      'environment': 'LIVE',
      'device_id': 'field-9700b9aeac391b79adce8c681eba4b58',
      'user_id': 'inspector-1',
      'upload_token': 'v1.upload-token-for-tus',
      'authority': {
        'authority_id': 'local-authority-abc',
        'authority_epoch': 1,
        'scopes': ['inspection.perform', 'evidence.upload'],
        'procedure_version': 'integin-local-provision-v1',
        'issued_at': '2026-09-22T15:40:35.306103Z',
        'expires_at': '2099-09-22T16:40:35.306103Z',
        'signature': 'sig',
      },
    };
    final session = ProvisionedFieldSession.fromJson(json);
    final restored =
        ProvisionedFieldSession.fromJson(session.toJson());
    expect(restored.deviceId, session.deviceId);
    expect(restored.userId, session.userId);
    expect(restored.uploadToken, 'v1.upload-token-for-tus');
    expect(restored.context.tenantId, session.context.tenantId);
    expect(restored.context.organizationId, session.context.organizationId);
    expect(restored.context.environment, session.context.environment);
    expect(restored.authority.id, session.authority.id);
    expect(restored.authority.epoch, session.authority.epoch);
    expect(restored.authority.scopes, session.authority.scopes);
    expect(restored.authority.capabilities, session.authority.capabilities);
    expect(restored.authority.issuedAt.toUtc(), session.authority.issuedAt.toUtc());
    expect(restored.authority.expiresAt.toUtc(), session.authority.expiresAt.toUtc());
    expect(restored.authority.signature, session.authority.signature);
    expect(
      restored.authority.isValidAt(DateTime.now().toUtc()),
      isTrue,
    );
  });

  test('legacy cached sessions without an upload token still parse', () {
    final session = ProvisionedFieldSession.fromJson({
      'tenant_id': 't',
      'organization_id': 'o',
      'environment': 'LIVE',
      'device_id': 'field-abc',
      'user_id': 'u',
      'authority': {
        'authority_id': 'a',
        'authority_epoch': 1,
        'scopes': ['inspection.perform'],
        'procedure_version': 'v',
        'issued_at': '2026-09-22T15:40:35.306103Z',
        'expires_at': '2099-09-22T16:40:35.306103Z',
        'signature': 'sig',
      },
    });
    expect(session.uploadToken, isNull);
    expect(session.toJson(), isNot(contains('upload_token')));
  });
}
