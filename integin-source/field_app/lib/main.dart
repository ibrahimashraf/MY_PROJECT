import 'package:flutter/material.dart';
import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'application/field_app_controller.dart';
import 'domain/models.dart';
import 'storage/persistent_outbox.dart';
import 'storage/secure_key_value_store.dart';
import 'security/transaction_signer.dart';
import 'presentation/field_app.dart';
import 'provisioning/local_provisioning_client.dart';
import 'sync/http_sync_transport.dart';
import 'sync/sync_client.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  const syncEndpoint = String.fromEnvironment('INTEGIN_SYNC_ENDPOINT');
  const localProvisioningEndpoint =
      String.fromEnvironment('INTEGIN_LOCAL_PROVISIONING_ENDPOINT');
  final preferences = await SharedPreferences.getInstance();
  final KeyValueStore storage = kIsWeb
      ? SharedPreferencesKeyValueStore(preferences)
      : SecureKeyValueStore();
  final outboxStore = JsonOutboxStore(
    storage: storage,
  );
  await outboxStore.load();
  DeviceSigner? deviceSigner;
  String? deviceKeyId;
  late TenantContext context;
  late String deviceId;
  late String userId;
  late OfflineAuthority authority;
  if (!kIsWeb) {
    deviceSigner = await SecureDeviceKeyStore().loadOrCreate('field-device');
    deviceKeyId = deviceSigner.keyId;
    if (localProvisioningEndpoint.isNotEmpty) {
      final session = await LocalProvisioningClient(
        endpoint: Uri.parse(localProvisioningEndpoint),
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
  final SyncClient? syncClient = syncEndpoint.isEmpty
      ? null
      : SyncClient(
          store: outboxStore,
          transport: HttpSyncTransport(endpoint: Uri.parse(syncEndpoint)),
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
  );
  await controller.restoreOutbox();
  if (syncClient != null) {
    controller.connectivity = ConnectivityState.online;
  }
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
