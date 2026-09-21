import 'dart:convert';
import 'dart:typed_data';
import 'package:local_auth/local_auth.dart';
import 'package:http/http.dart' as http;

import 'package:integin_field_app/security/grace_state_machine.dart';
import 'package:integin_field_app/security/hardware_attestation_bridge.dart';
import 'package:integin_field_app/security/submission_seal.dart';
import 'package:integin_field_app/security/pinned_http_client.dart';
import 'package:integin_field_app/security/endpoint_guard.dart';

/// Inspection submission transport status codes returned by the
/// edge coordinator's `/api/v1/inspection/submit` endpoint.
enum SubmissionTransportStatus {
  accepted,
  awaitingReconciliation,
  rejected,
}

class SubmissionTransportResult {
  final SubmissionTransportStatus status;
  final String? reconciliationId;
  final String? serverMessage;
  final int? httpStatusCode;

  const SubmissionTransportResult({
    required this.status,
    this.reconciliationId,
    this.serverMessage,
    this.httpStatusCode,
  });
}

/// Transport layer that bridges the Phase 2 hardware attestation
/// bridge to the Phase 3 edge coordinator submission endpoint.
///
/// Lifecycle:
/// 1. Serialize [SubmissionSealPayload] into canonical bytes.
/// 2. Compute composite SHA256 digest via [SubmissionSealPayload.deriveCompositeDigest].
/// 3. Sign the digest using [MethodChannelHardwareAttestation.signDigest].
///    If the hardware reports [AUTH_REQUIRED], trigger [LocalAuthentication]
///    and retry the signing once.
/// 4. POST the signed envelope to the edge coordinator.
/// 5. Branch on response: [accepted], [awaitingReconciliation], or [rejected].
class InspectionSubmissionService {
  InspectionSubmissionService({
    HardwareAttestationProvider? attestationProvider,
    LocalAuthentication? localAuth,
  })  : _attestationProvider = attestationProvider ?? MethodChannelHardwareAttestation(),
        _localAuth = localAuth ?? LocalAuthentication();

  final HardwareAttestationProvider _attestationProvider;
  final LocalAuthentication _localAuth;

  /// Submits an inspection sealed payload to the edge coordinator.
  ///
  /// [payload] is the sealed data to commit.
  /// [deviceKeyDID] is the hardware-derived device identity.
  /// [endpoint] is the edge coordinator submission URL.
  /// [tokenID] and [leaseEpoch] are used for the seal envelope.
  /// [appEd25519Sig] is the app-level Ed25519 signature over the payload.
  Future<SubmissionTransportResult> submitInspection({
    required SubmissionSealPayload payload,
    required String deviceKeyDID,
    required Uri endpoint,
    required String tokenID,
    required int leaseEpoch,
    required Uint8List appEd25519Sig,
    required http.Client? httpClient,
  }) async {
    assertEndpointSafe(endpoint, allowLoopbackHttp: false);

    // Step 1: Compute composite digest
    final compositeDigest = payload.deriveCompositeDigest();

    // Step 2: Sign the digest with hardware key, retrying on AUTH_REQUIRED
    Uint8List signature;
    try {
      signature = await _attestationProvider.signDigest(
        alias: deviceKeyDID,
        precomputed32ByteDigest: compositeDigest,
      );
    } on SecurityException catch (e) {
      if (e.code == 'AUTH_REQUIRED') {
        signature = await _retryWithBiometric(
          alias: deviceKeyDID,
          digest: compositeDigest,
        );
      } else {
        rethrow;
      }
    }

    // Step 3: Build the submission envelope
    final submissionBody = <String, dynamic>{
      'device_key_did': deviceKeyDID,
      'token_id': tokenID,
      'lease_epoch': leaseEpoch,
      'composite_digest': base64Encode(compositeDigest),
      'hw_signature': base64Encode(signature),
      'app_ed25519_sig': base64Encode(appEd25519Sig),
      'data_payload': base64Encode(payload.dataPayload),
    };

    // Step 4: Dispatch to edge coordinator
    final client = httpClient ?? PinnedHttpClient.forEndpoint(endpoint);
    final response = await client.post(
      endpoint.resolve('/api/v1/inspection/submit'),
      headers: const {'Content-Type': 'application/json'},
      body: jsonEncode(submissionBody),
    );

    // Step 5: Branch on response
    final body = jsonDecode(response.body);
    if (body is! Map) {
      return SubmissionTransportResult(
        status: SubmissionTransportStatus.rejected,
        serverMessage: 'Invalid server response',
        httpStatusCode: response.statusCode,
      );
    }

    final statusField = body['status'] as String?;
    switch (statusField) {
      case 'ACCEPTED':
        return SubmissionTransportResult(
          status: SubmissionTransportStatus.accepted,
          serverMessage: body['message'] as String?,
          httpStatusCode: response.statusCode,
        );
      case 'AWAITING_RECONCILIATION':
        return SubmissionTransportResult(
          status: SubmissionTransportStatus.awaitingReconciliation,
          reconciliationId: body['reconciliation_id'] as String?,
          serverMessage: body['message'] as String?,
          httpStatusCode: response.statusCode,
        );
      default:
        return SubmissionTransportResult(
          status: SubmissionTransportStatus.rejected,
          serverMessage: body['message'] as String? ?? 'Submission rejected',
          httpStatusCode: response.statusCode,
        );
    }
  }

  /// Retries [signDigest] after prompting the user for biometric
  /// authentication via [LocalAuthentication].
  ///
  /// Returns the signature on success. Throws [SecurityException]
  /// if the user cancels or if signing fails again.
  Future<Uint8List> _retryWithBiometric({
    required String alias,
    required Uint8List digest,
  }) async {
    final authenticated = await _localAuth.authenticate(
      localizedReason: 'Authenticate to complete inspection submission',
      biometricOnly: true,
    );
    if (!authenticated) {
      throw SecurityException(
        'Biometric authentication cancelled by user',
        'AUTH_REQUIRED',
      );
    }
    return _attestationProvider.signDigest(
      alias: alias,
      precomputed32ByteDigest: digest,
    );
  }

  /// Validates that a grace evaluation allows submission.
  /// Returns null if the evaluation permits submission, or the
  /// block reason if submission should be blocked.
  static String? validateGraceBeforeSubmit({
    required DateTime currentWall,
    required int currentMonoNanos,
    required String currentBootSessionId,
    required CheckpointLedgerEntry checkpoint,
    required TimeHorizonToken token,
    required TTLPolicy policy,
  }) {
    final evaluation = GraceStateMachine.evaluate(
      currentWall: currentWall,
      currentMonoNanos: currentMonoNanos,
      currentBootSessionId: currentBootSessionId,
      checkpoint: checkpoint,
      token: token,
      policy: policy,
    );

    if (evaluation.status == GraceStatus.hardLocked) {
      return evaluation.blockReason;
    }
    return null;
  }

  /// Requests the system to exempt the app from OEM battery
  /// optimization daemons (Zebra StageNow, Honeywell Power
  /// Management, Samsung Knox Battery Optimization) that would
  /// suspend background Flutter isolates and kernel monotonic
  /// timers during prolonged field inspections.
  ///
  /// Distribution: This permission is restricted by Google Play
  /// Console automated review to public Store apps. Integin is
  /// distributed exclusively via enterprise sideloading, private
  /// corporate app stores, or MDM profiles (Knox, SOTI, Intune)
  /// on rugged terminals (Zebra, Honeywell, Samsung). Play Store
  /// submission is explicitly out of scope.
  ///
  /// **Phantom Process Killer / LMK note:** Manifest permission
  /// alone is insufficient on Android 12+. Samsung Knox and Zebra
  /// StageNow freeze Flutter processes after 15–30 minutes of
  /// screen-off unless the app holds an active foreground service
  /// (startForeground(id, notification)). For background outbox
  /// flush during disconnected shifts, pair this exemption with a
  /// native ForegroundService. Without it, outbox sync halts while
  /// the tablet is clipped to the inspector's harness.
  ///
  /// Must be called before entering disconnected mode.
  /// Returns true if already exempted or exemption granted.
  static Future<bool> ensureBatteryOptimizationExemption() async {
    // Android-only: request ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS
    // The platform-specific implementation is handled via the
    // method channel 'com.integin.field/battery' in the native
    // SecurityPlugin, which delegates to
    // PowerManager.requestIgnoreBatteryOptimization().
    // iOS has no equivalent API — see validateOnForeground below.
    try {
      final result = await MethodChannel('com.integin.field/battery')
          .invokeMethod<bool>('requestExemption');
      return result ?? false;
    } catch (_) {
      // If the channel is unavailable (simulator or non-Android),
      // treat as exempted.
      return true;
    }
  }

  /// Re-validates the grace state when the iOS app returns
  /// from the background after a screen lock or app switch.
  ///
  /// iOS has no battery optimization API, so the OS suspends
  /// the Dart event loop within ~30 seconds of screen-off.
  /// Kernel monotonic time (CLOCK_UPTIME_RAW) continues running,
  /// but the Flutter event loop is frozen. On resumption, this
  /// method must immediately re-evaluate the current monotonic
  /// timestamp against the stored checkpoint. If the cold-boot
  /// grace window has expired while the thread was suspended,
  /// the app transitions to hardLocked and refuses signing
  /// until the user re-authenticates and re-anchors to the
  /// edge appliance.
  ///
  /// **Lifecycle gating:** Call only when returning from
  /// [AppLifecycleState.paused] (true backgrounding / screen lock).
  /// Do NOT call when returning from [AppLifecycleState.inactive]
  /// (biometric prompt, system dialog) — the biometric overlay
  /// causes `inactive → resumed` transitions that race with the
  /// in-flight signing call and would cause spurious hard-locks.
  ///
  /// Usage in WidgetsBindingObserver.didChangeAppLifecycleState:
  /// ```dart
  /// AppLifecycleState? _lastState;
  /// void didChangeAppLifecycleState(AppLifecycleState state) {
  ///   if (state == AppLifecycleState.resumed && _lastState == AppLifecycleState.paused) {
  ///     InspectionSubmissionService.validateOnForeground(...);
  ///   }
  ///   _lastState = state;
  /// }
  /// ```
  static Future<void> validateOnForeground({
    required DateTime currentWall,
    required int currentMonoNanos,
    required String currentBootSessionId,
    required CheckpointLedgerEntry checkpoint,
    required TimeHorizonToken token,
    required TTLPolicy policy,
  }) async {
    final blockReason = validateGraceBeforeSubmit(
      currentWall: currentWall,
      currentMonoNanos: currentMonoNanos,
      currentBootSessionId: currentBootSessionId,
      checkpoint: checkpoint,
      token: token,
      policy: policy,
    );
    if (blockReason != null) {
      throw SecurityException(
        'Foreground resumption failed grace validation: $blockReason',
        'HARD_LOCKED',
      );
    }
  }
}
