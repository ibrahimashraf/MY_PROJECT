import 'package:flutter/material.dart';
import 'package:http/http.dart' as http;

import '../security/attestation.dart';

/// Loopback origin of the pilot EnrollServer (cmd/INTEGIN-server, canonical
/// INTEGIN_HTTP_ADDR pilot port :18080). The Go server refuses to mount
/// enrollment off-loopback, so this screen can never target a remote origin.
const enrollLoopbackEndpoint = 'http://127.0.0.1:18080';

/// The only posture the simulator can produce: a SOFTWARE claim enrolls as
/// CLAIMED-unverified, never VERIFIED (only an offline-verified hardware chain
/// sets attestation_verified on the server).
const simEnrollmentPosture =
    'Simulator posture: the SOFTWARE key claim enrolls this device as '
    'CLAIMED-unverified, never VERIFIED.';

enum _EnrollPhase { idle, enrolling, success, error }

class EnrollmentScreen extends StatefulWidget {
  const EnrollmentScreen({
    super.key,
    required this.tenantId,
    required this.organizationId,
    this.endpoint = enrollLoopbackEndpoint,
    this.useProductionRoute = false,
    this.client,
    this.attestationProvider,
  });

  final String tenantId;
  final String organizationId;
  final String endpoint;
  final bool useProductionRoute;
  final http.Client? client;
  final AttestationProvider? attestationProvider;

  @override
  State<EnrollmentScreen> createState() => _EnrollmentScreenState();
}

class _EnrollmentScreenState extends State<EnrollmentScreen> {
  final TextEditingController _inspectorController = TextEditingController();
  final TextEditingController _modelController = TextEditingController();
  late final TextEditingController _orgController;
  _EnrollPhase _phase = _EnrollPhase.idle;
  late bool _useProduction;
  Map<String, Object?>? _record;
  String? _error;

  @override
  void initState() {
    super.initState();
    _useProduction = widget.useProductionRoute;
    _orgController = TextEditingController(text: widget.organizationId);
    _orgController.addListener(_onFieldChanged);
    _inspectorController.addListener(_onFieldChanged);
    _modelController.addListener(_onFieldChanged);
  }

  void _onFieldChanged() => setState(() {});

  @override
  void dispose() {
    _orgController.dispose();
    _inspectorController.dispose();
    _modelController.dispose();
    super.dispose();
  }

  bool get _fieldsPopulated =>
      _orgController.text.trim().isNotEmpty &&
      _inspectorController.text.trim().isNotEmpty &&
      _modelController.text.trim().isNotEmpty;

  Future<void> _enroll() async {
    final inspectorId = _inspectorController.text.trim();
    final deviceModel = _modelController.text.trim();
    final organizationId = _orgController.text.trim();
    setState(() {
      _phase = _EnrollPhase.enrolling;
      _error = null;
      _record = null;
    });
    try {
      final record = _useProduction
          ? await enrollProductionDevice(
              endpoint: Uri.parse(widget.endpoint),
              tenantId: widget.tenantId,
              organizationId: organizationId,
              userId: inspectorId,
              deviceModel: deviceModel,
              client: widget.client,
              attestationProvider: widget.attestationProvider,
            )
          : await enrollSimulatedDevice(
              endpoint: Uri.parse(widget.endpoint),
              tenantId: widget.tenantId,
              inspectorId: inspectorId,
              deviceModel: deviceModel,
              client: widget.client,
              attestationProvider: widget.attestationProvider,
            );
      if (!mounted) return;
      setState(() {
        _record = record;
        _phase = _EnrollPhase.success;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _phase = _EnrollPhase.error;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Device Enrollment')),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Pilot device enrollment',
                style: Theme.of(context).textTheme.headlineSmall),
            const SizedBox(height: 8),
            const Text(
              'Enroll against the loopback pilot EnrollServer in '
              'pkg/onboarding. This is a software simulator path.',
            ),
            const SizedBox(height: 20),
            TextField(
              controller: _orgController,
              enabled: _phase != _EnrollPhase.enrolling,
              decoration: const InputDecoration(
                labelText: 'Organization ID',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _inspectorController,
              enabled: _phase != _EnrollPhase.enrolling,
              decoration: const InputDecoration(
                labelText: 'Inspector ID',
                border: OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 16),
            TextField(
              controller: _modelController,
              enabled: _phase != _EnrollPhase.enrolling,
              decoration: const InputDecoration(
                labelText: 'Device model',
                border: OutlineInputBorder(),
              ),
            ),

            const SizedBox(height: 16),
            SwitchListTile(
              title: const Text('Production API Route (/api/v1/devices/enroll)'),
              subtitle: Text(_useProduction
                  ? 'Active: Submits to production device enrollment vector'
                  : 'Inactive: Uses pilot loopback simulator endpoints'),
              value: _useProduction,
              onChanged: _phase == _EnrollPhase.enrolling
                  ? null
                  : (v) => setState(() => _useProduction = v),
            ),
            const SizedBox(height: 24),
            Card(
              child: Padding(
                padding: const EdgeInsets.all(16),
                child: Text(simEnrollmentPosture),
              ),
            ),
            const SizedBox(height: 24),
            FilledButton.icon(
              onPressed: _phase == _EnrollPhase.enrolling || !_fieldsPopulated
                  ? null
                  : _enroll,
              icon: _phase == _EnrollPhase.enrolling
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.vpn_key_outlined),
              label: Text(_phase == _EnrollPhase.enrolling
                  ? 'Enrolling…'
                  : 'Enroll device'),
            ),
            if (_phase == _EnrollPhase.error && _error != null) ...[
              const SizedBox(height: 20),
              Text(_error!,
                  style: TextStyle(color: Theme.of(context).colorScheme.error)),
            ],
            if (_phase == _EnrollPhase.success && _record != null) ...[
              const SizedBox(height: 20),
              _EnrollmentSuccessCard(record: _record!),
            ],
          ],
        ),
      ),
    );
  }
}

class _EnrollmentSuccessCard extends StatelessWidget {
  const _EnrollmentSuccessCard({required this.record});

  final Map<String, Object?> record;

  String _string(String key) => record[key] is String ? record[key]! as String : '';

  @override
  Widget build(BuildContext context) {
    final deviceId = _string('device_id');
    final origin = _string('attestation_origin');
    final verified = record['attestation_verified'] == true;
    return Card(
      color: Colors.green.shade50,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text('Enrolled',
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 8),
            Text('Device ID: $deviceId'),
            const SizedBox(height: 4),
            Text('Attestation origin: $origin'),
            const SizedBox(height: 4),
            Text('Biometric bound: '
                '${record['attestation_biometric_bound'] == true}'),
            const SizedBox(height: 4),
            Text(
                'Attestation verified: $verified (a SOFTWARE simulator claim '
                'never yields a verified hardware chain).'),
          ],
        ),
      ),
    );
  }
}