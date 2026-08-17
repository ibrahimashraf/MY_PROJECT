import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/pilot/pilot_fixture_bootstrap.dart';

const _fixtureDirectory =
    r'C:\MY PROJECT\private\integin-secrets\pilot-live-fixtures';

void main() {
  test('rejects a fixture path outside the protected pilot directory', () async {
    await expectLater(
      PilotFixtureBootstrap.loadWindowsDebugFixture(r'C:\temp\not-pilot.json'),
      throwsA(isA<StateError>()),
    );
  });

  test('reconstructs an isolated pilot session from a bounded fixture',
      () async {
    final directory = Directory(_fixtureDirectory);
    await directory.create(recursive: true);
    final fixture = File(
      '${directory.path}${Platform.pathSeparator}flutter-bootstrap-test.json',
    );
    await fixture.writeAsString(
      jsonEncode({
        'tenant_id': 'pilot-tenant-a',
        'organization_id': 'integin-integration-org',
        'user_id': 'integin-integration-user',
        'device_id': 'pilot-fixture-device',
        'authority_id': 'pilot-fixture-authority',
        'key_id': 'pilot-fixture-key',
        'private_key_base64': base64Encode(List<int>.filled(32, 7)),
      }),
    );
    try {
      final bootstrap =
          await PilotFixtureBootstrap.loadWindowsDebugFixture(fixture.path);

      expect(bootstrap.session.context.tenantId, 'pilot-tenant-a');
      expect(bootstrap.session.context.environment, 'TESTING');
      expect(bootstrap.session.authority.id, 'pilot-fixture-authority');
      expect(bootstrap.signer.keyId, 'pilot-fixture-key');
    } finally {
      if (await fixture.exists()) {
        await fixture.delete();
      }
    }
  });
}
