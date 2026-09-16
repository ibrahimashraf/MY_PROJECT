import 'dart:io';

// INTEGIN field app entry point.
// Pilot fixture bootstrap is explicitly Windows-debug-only and cannot serve as
// production device enrollment or target the acceptance control runtime.

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

import 'advisory/pilot_advisory_client.dart';
import 'application/field_app_controller.dart';
import 'domain/models.dart';
import 'pilot/pilot_fixture_bootstrap.dart';
import 'presentation/field_app.dart';
import 'provisioning/local_provisioning_client.dart';
import 'security/transaction_signer.dart';
import 'storage/persistent_outbox.dart';
import 'storage/secure_key_value_store.dart';
import 'sync/http_sync_transport.dart';
import 'sync/sync_client.dart';

const _windowsStartupTracePath =
    String.fromEnvironment('INTEGIN_WINDOWS_STARTUP_TRACE_PATH');

// A non-empty value isolates a controlled pilot walkthrough queue without
// deleting or mutating any prior local queue history.
const _pilotOutboxNamespace =
    String.fromEnvironment('INTEGIN_PILOT_OUTBOX_NAMESPACE');

void _traceWindowsStartup(String stage) {
  if (kIsWeb || _windowsStartupTracePath.isEmpty) {
    return;
  }
  try {
    File(_windowsStartupTracePath).writeAsStringSync(
      '$stage\n',
      mode: FileMode.append,
      flush: true,
    );
  } catch (_) {
    // The opt-in pilot trace must never alter startup behavior.
  }
}

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  _traceWindowsStartup('binding-ready');
  const syncEndpoint = String.fromEnvironment('INTEGIN_SYNC_ENDPOINT');
  const localProvisioningEndpoint =
      String.fromEnvironment('INTEGIN_LOCAL_PROVISIONING_ENDPOINT');
  const pilotFixturePath =
      String.fromEnvironment('INTEGIN_PILOT_WINDOWS_FIXTURE_PATH');
  const pilotMode = bool.fromEnvironment('INTEGIN_PILOT_WINDOWS_MODE');
  const pilotAdvisoryEndpoint =
      String.fromEnvironment('INTEGIN_PILOT_ADVISORY_ENDPOINT');

  _traceWindowsStartup('mode-resolved');
  if (pilotMode) {
    final endpoint = Uri.tryParse(syncEndpoint);
    if (kIsWeb ||
        endpoint == null ||
        endpoint.scheme != 'http' ||
        endpoint.host != '127.0.0.1' ||
        endpoint.port != 18080 ||
        endpoint.path != '/sync' ||
        pilotFixturePath.isEmpty ||
        localProvisioningEndpoint.isNotEmpty) {
      throw StateError(
          'pilot fixture mode requires isolated loopback pilot settings');
    }
    if (pilotAdvisoryEndpoint.isNotEmpty) {
      final advisoryUri = Uri.tryParse(pilotAdvisoryEndpoint);
      if (advisoryUri == null ||
          advisoryUri.scheme != 'http' ||
          advisoryUri.host != '127.0.0.1' ||
          advisoryUri.port != 8001 ||
          advisoryUri.path != '/v1/advisory') {
        throw StateError(
            'pilot advisory requires the isolated loopback endpoint');
      }
    }
  } else if (pilotFixturePath.isNotEmpty) {
    throw StateError('pilot fixture path requires explicit pilot Windows mode');
  }

  _traceWindowsStartup('preferences-start');
  _traceWindowsStartup('preferences-ready');
  final KeyValueStore storage = SecureKeyValueStore();
  final outboxKey = _pilotOutboxNamespace.isEmpty
      ? 'integin.outbox.v1'
      : 'integin.outbox.v1.$_pilotOutboxNamespace';
  final outboxStore = JsonOutboxStore(storage: storage, key: outboxKey);
  _traceWindowsStartup('outbox-load-start');
  await outboxStore.load();
  _traceWindowsStartup('outbox-load-ready');

  DeviceSigner? deviceSigner;
  String? deviceKeyId;
  late TenantContext context;
  late String deviceId;
  late String userId;
  late OfflineAuthority authority;

  _traceWindowsStartup('mode-resolved');
  if (pilotMode) {
    _traceWindowsStartup('pilot-bootstrap-start');
    final bootstrap = await PilotFixtureBootstrap.loadWindowsDebugFixture(
      pilotFixturePath,
      trace: _traceWindowsStartup,
    );
    deviceSigner = bootstrap.signer;
    deviceKeyId = bootstrap.signer.keyId;
    context = bootstrap.session.context;
    deviceId = bootstrap.session.deviceId;
    userId = bootstrap.session.userId;
    authority = bootstrap.session.authority;
  } else if (!kIsWeb) {
    deviceSigner = await SecureDeviceKeyStore().loadOrCreate('field-device');
    deviceKeyId = deviceSigner.keyId;
    if (localProvisioningEndpoint.isNotEmpty) {
      final session = await LocalProvisioningClient(
        endpoint: Uri.parse(localProvisioningEndpoint),
        allowLoopbackHttp: pilotMode,
      ).provision(deviceSigner);
      context = session.context;
      deviceId = session.deviceId;
      userId = session.userId;
      authority = session.authority;
    } else {
      final session = _demonstrationSession();
      context = session.context;
      deviceId = session.deviceId;
      userId = session.userId;
      authority = session.authority;
    }
  } else {
    if (localProvisioningEndpoint.isNotEmpty) {
      throw StateError('local provisioning requires native secure key storage');
    }
    final session = _demonstrationSession();
    context = session.context;
    deviceId = session.deviceId;
    userId = session.userId;
    authority = session.authority;
  }

  final PilotAdvisoryClient? advisoryClient =
      pilotMode && pilotAdvisoryEndpoint.isNotEmpty
          ? PilotAdvisoryClient(
              endpoint: Uri.parse(pilotAdvisoryEndpoint),
              allowLoopbackHttp: pilotMode,
            )
          : null;
  final SyncClient? syncClient = syncEndpoint.isEmpty
      ? null
      : SyncClient(
          store: outboxStore,
          transport: HttpSyncTransport(
            endpoint: Uri.parse(syncEndpoint),
            // Pilot mode uses http://127.0.0.1:18080 — loopback cleartext is
            // explicitly permitted and validated above. Production never hits
            // this branch with an http:// URL.
            allowLoopbackHttp: pilotMode,
          ),
        );
  final controller = FieldAppController(
    context: context,
    deviceId: deviceId,
    userId: userId,
    deviceState: DeviceTrustState.trusted,
    authority: authority,
    outboxStore: outboxStore,
    deviceSigner: deviceSigner,
    deviceKeyId: deviceKeyId,
    syncClient: syncClient,
    advisoryClient: advisoryClient,
    trace: _traceWindowsStartup,
  );
  await controller.restoreOutbox();
  if (syncClient != null) {
    controller.connectivity = ConnectivityState.online;
  }
  _traceWindowsStartup('run-app');
  runApp(InteginFieldApp(controller: controller));
}

ProvisionedFieldSession _demonstrationSession() {
  const context = TenantContext(
    tenantId: 'tenant-session',
    organizationId: 'organization-session',
    environment: 'LIVE',
  );
  final issued = DateTime.now().toUtc().subtract(const Duration(minutes: 1));
  return ProvisionedFieldSession(
    context: context,
    deviceId: 'device-session',
    userId: 'inspector-session',
    authority: OfflineAuthority(
      id: 'authority-session',
      deviceId: 'device-session',
      context: context,
      userId: 'inspector-session',
      epoch: 1,
      scopes: const ['inspection.perform'],
      capabilities: const ['inspection.perform'],
      procedureVersion: 'prepared-work-pack',
      issuedAt: issued,
      expiresAt: issued.add(const Duration(hours: 8)),
      signature: 'server-issued-authority',
    ),
  );
}

class InteginFieldApp extends StatelessWidget {
  const InteginFieldApp({super.key, required this.controller});

  final FieldAppController controller;

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'INTEGIN Field',
      theme: ThemeData(
        colorScheme: ColorScheme.fromSeed(seedColor: const Color(0xff537a68)),
        useMaterial3: true,
      ),
      home: FieldHomePage(controller: controller),
    );
  }
}
