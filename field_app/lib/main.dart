import 'dart:convert';
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
import 'storage/app_database.dart';
import 'storage/drift_outbox_store.dart';
import 'storage/secure_key_value_store.dart';
import 'sync/http_sync_transport.dart';
import 'sync/sync_client.dart';
import 'sync/tus_client.dart';
import 'security/pinned_http_client.dart';
import 'security/endpoint_guard.dart';

const _windowsStartupTracePath =
    String.fromEnvironment('INTEGIN_WINDOWS_STARTUP_TRACE_PATH');

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
  final secureStorage = SecureKeyValueStore();
  _traceWindowsStartup('database-open-start');
  final database = await openAppDatabase(
    readKey: secureStorage.read,
    writeKey: secureStorage.write,
  );
  _traceWindowsStartup('database-open-ready');
  final outboxStore = DriftOutboxStore(database);
  _traceWindowsStartup('outbox-ready');

  DeviceSigner? deviceSigner;
  String? deviceKeyId;
  late TenantContext context;
  late String deviceId;
  late String userId;
  late OfflineAuthority authority;
  String? uploadToken;

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
      final keyId = deviceSigner.keyId;
      if (keyId == null || keyId.length < 32) {
        throw StateError('local provisioning requires a device key id');
      }
      final expectedDeviceId = 'field-${keyId.substring(0, 32)}';
      const sessionCacheKey = 'INTEGIN.provisioned_session';
      ProvisionedFieldSession? session;
      final cachedSession = await secureStorage.read(sessionCacheKey);
      if (cachedSession != null) {
        try {
          final parsed = ProvisionedFieldSession.fromJson(
              Map<String, Object?>.from(
                  jsonDecode(cachedSession) as Map));
          if (parsed.deviceId == expectedDeviceId &&
              parsed.authority.isValidAt(DateTime.now().toUtc())) {
            session = parsed;
          }
        } catch (_) {
          // Fall through to a fresh provision on any cache decode issue.
        }
      }
      if (session == null) {
        session = await LocalProvisioningClient(
          endpoint: Uri.parse(localProvisioningEndpoint),
          allowLoopbackHttp: pilotMode ||
              isLoopbackUri(Uri.parse(localProvisioningEndpoint)),
        ).provision(deviceSigner);
        await secureStorage.write(
            sessionCacheKey, jsonEncode(session.toJson()));
      }
      context = session.context;
      deviceId = session.deviceId;
      userId = session.userId;
      authority = session.authority;
      uploadToken = session.uploadToken;
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
            allowLoopbackHttp: pilotMode || isLoopbackUri(Uri.parse(syncEndpoint)),
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
            allowLoopbackHttp: pilotMode || isLoopbackUri(Uri.parse(syncEndpoint)),
          ),
        );
  final TusClient? tusClient = syncEndpoint.isEmpty
      ? null
      : TusClient(
          http: HttpTusHttp(
            baseUrl: Uri.parse(syncEndpoint).resolve('/uploads'),
            client: PinnedHttpClient.forEndpoint(Uri.parse(syncEndpoint), allowLoopbackHttp: pilotMode || isLoopbackUri(Uri.parse(syncEndpoint))),
            allowLoopbackHttp: pilotMode || isLoopbackUri(Uri.parse(syncEndpoint)),
            // Provision-bound upload token minted with the device authority;
            // the server accepts it on /uploads without an interactive login.
            authToken: uploadToken,
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
    tusClient: tusClient,
    advisoryClient: advisoryClient,
    trace: _traceWindowsStartup,
  );
  await controller.restoreOutbox();
  // Authority packages rotate on every fresh provision: re-bind any entries
  // queued under a previous device identity so the inspector's work survives
  // rotation instead of resting in terminal failure states.
  await controller.reconcileOutboxIdentities();
  if (!kIsWeb) {
    controller.connectivity = ConnectivityState.online;
  } else if (syncClient != null) {
    controller.connectivity = ConnectivityState.online;
  }
  _traceWindowsStartup('run-app');
  runApp(INTEGINFieldApp(controller: controller));
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

class INTEGINFieldApp extends StatelessWidget {
  const INTEGINFieldApp({super.key, required this.controller});

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
