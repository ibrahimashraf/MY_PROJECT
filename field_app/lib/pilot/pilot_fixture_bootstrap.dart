// INTEGIN pilot-only Flutter session bootstrap.
// This module is intentionally for Windows debug acceptance only. It is not a
// production enrollment, recovery, or credential-distribution mechanism.

import 'dart:convert';
import 'dart:io';

import 'package:cryptography/dart.dart';

import '../domain/models.dart';
import '../provisioning/local_provisioning_client.dart';
import '../security/transaction_signer.dart';

const _pilotFixtureRoot =
    r'C:\MY PROJECT\private\integin-secrets\pilot-live-fixtures\';

class PilotFixtureBootstrap {
  PilotFixtureBootstrap._({required this.session, required this.signer});

  final ProvisionedFieldSession session;
  final DeviceSigner signer;

  static Future<PilotFixtureBootstrap> loadWindowsDebugFixture(
    String fixturePath, {
    void Function(String stage)? trace,
  }) async {
    if (!Platform.isWindows) {
      throw StateError('pilot fixture bootstrap is Windows-debug-only');
    }
    final normalizedPath = fixturePath.replaceAll('/', r'\');
    if (!normalizedPath.startsWith(_pilotFixtureRoot)) {
      throw StateError(
          'pilot fixture bootstrap requires the protected pilot fixture directory');
    }

    trace?.call('fixture-read-start');
    final rawFixture = await File(fixturePath).readAsString();
    trace?.call('fixture-read-ready');
    final decoded = jsonDecode(rawFixture);
    trace?.call('fixture-decode-ready');
    if (decoded is! Map<String, Object?>) {
      throw StateError('pilot fixture must be a JSON object');
    }
    final fixture = decoded;
    final required = <String>[
      'tenant_id',
      'organization_id',
      'user_id',
      'device_id',
      'authority_id',
      'key_id',
      'private_key_base64',
    ];
    for (final key in required) {
      if (fixture[key] is! String || (fixture[key] as String).isEmpty) {
        throw StateError('pilot fixture is missing required session material');
      }
    }

    final tenantId = fixture['tenant_id'] as String;
    final organizationId = fixture['organization_id'] as String;
    final userId = fixture['user_id'] as String;
    final deviceId = fixture['device_id'] as String;
    final authorityId = fixture['authority_id'] as String;
    final keyId = fixture['key_id'] as String;
    final privateKeyBytes =
        base64Decode(fixture['private_key_base64'] as String);
    if (privateKeyBytes.length != 32 && privateKeyBytes.length != 64) {
      throw ArgumentError(
          'Pilot fixture Ed25519 private key must be 32-byte seed or 64-byte private key.');
    }
    final seed = privateKeyBytes.length == 32
        ? privateKeyBytes
        : privateKeyBytes.sublist(0, 32);
    trace?.call('fixture-private-key-length-${privateKeyBytes.length}');
    trace?.call('fixture-ed25519-seed-length-${seed.length}');
    trace?.call('fixture-signer-start');
    late final DeviceSigner signer;
    try {
      signer = DeviceSigner(
        await DartEd25519(sha512: const DartSha512()).newKeyPairFromSeed(seed),
        keyId: keyId,
      );
    } catch (error) {
      trace?.call('fixture-signer-error-${error.runtimeType}');
      rethrow;
    }
    trace?.call('fixture-signer-ready');
    final issuedAt =
        DateTime.now().toUtc().subtract(const Duration(minutes: 1));
    final context = TenantContext(
      tenantId: tenantId,
      organizationId: organizationId,
      environment: 'TESTING',
    );

    return PilotFixtureBootstrap._(
      signer: signer,
      session: ProvisionedFieldSession(
        context: context,
        deviceId: deviceId,
        userId: userId,
        authority: OfflineAuthority(
          id: authorityId,
          deviceId: deviceId,
          context: context,
          userId: userId,
          epoch: 1,
          scopes: const ['inspection.perform'],
          capabilities: const ['inspection.perform'],
          procedureVersion: 'pilot-matrix-fixture-v1',
          issuedAt: issuedAt,
          expiresAt: issuedAt.add(const Duration(hours: 1)),
          signature: 'pilot-fixture-not-production-enrollment',
        ),
      ),
    );
  }
}
