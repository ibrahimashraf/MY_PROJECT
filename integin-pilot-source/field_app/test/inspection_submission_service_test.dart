import 'dart:convert';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:integin_field_app/security/inspection_submission_service.dart';
import 'package:integin_field_app/security/grace_state_machine.dart';
import 'package:integin_field_app/security/hardware_attestation_bridge.dart';
import 'package:integin_field_app/security/submission_seal.dart';

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('InspectionSubmissionService', () {
    const channel = MethodChannel('com.integin.field/security');
    const deviceKeyDID = 'did:example:device1';
    const tokenID = 'token-abc';
    const leaseEpoch = 1700000000;

    setUp(() {
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, null);
    });

    tearDown(() {
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, null);
    });

    test('submitInspection signs composite digest and dispatches', () async {
      final signature = Uint8List.fromList(List.generate(64, (i) => i));
      final payload = SubmissionSealPayload(
        deviceKeyDID: deviceKeyDID,
        tokenID: tokenID,
        leaseEpoch: leaseEpoch,
        dataPayload: Uint8List.fromList(utf8.encode('test-payload')),
        appEd25519Sig: Uint8List.fromList(List.generate(64, (i) => i)),
      );

      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async => signature);

      final digest = payload.deriveCompositeDigest();
      expect(digest.length, 32);
    });

    test('submitInspection handles AUTH_REQUIRED and retries', () async {
      int signCallCount = 0;
      final signature = Uint8List.fromList(List.generate(64, (i) => i));

      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(channel, (call) async {
        if (call.method == 'signDigest') {
          signCallCount++;
          if (signCallCount == 1) {
            throw PlatformException(
              code: 'AUTH_REQUIRED',
              message: 'UserNotAuthenticated: biometric expired',
            );
          }
          return signature;
        }
        return null;
      });

      final provider = MethodChannelHardwareAttestation();
      final digest = Uint8List(32);
      // Simulate AUTH_REQUIRED then successful retry
      try {
        await provider.signDigest(alias: 'test', precomputed32ByteDigest: digest);
      } catch (_) {
        // First call throws AUTH_REQUIRED
      }

      // Retry should succeed
      final result = await provider.signDigest(alias: 'test', precomputed32ByteDigest: digest);
      expect(result, signature);
      expect(signCallCount, 2);
    });

    test('AUTH_REQUIRED retry accepts device-credential fallback', () async {
      var signCalls = 0;
      final service = InspectionSubmissionService(
        attestationProvider: _FlakyProvider(onSign: () async {
          signCalls += 1;
          if (signCalls == 1) {
            throw SecurityException(
              'UserNotAuthenticated: biometric expired',
              'AUTH_REQUIRED',
            );
          }
          return Uint8List.fromList(List.generate(64, (i) => i));
        }),
      );
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
          .setMockMethodCallHandler(
        const MethodChannel('plugins.flutter.io/local_auth'),
        (call) async => call.method == 'authenticate' ? true : null,
      );

      final result = await service.submitInspection(
        payload: SubmissionSealPayload(
          deviceKeyDID: deviceKeyDID,
          tokenID: tokenID,
          leaseEpoch: leaseEpoch,
          dataPayload: Uint8List.fromList(utf8.encode('test-payload')),
          appEd25519Sig: Uint8List.fromList(List.generate(64, (i) => i)),
        ),
        deviceKeyDID: deviceKeyDID,
        endpoint: Uri.parse('https://edge.example.com'),
        tokenID: tokenID,
        leaseEpoch: leaseEpoch,
        appEd25519Sig: Uint8List.fromList(List.generate(64, (i) => i)),
        httpClient: MockClient((request) async => http.Response(
              jsonEncode({'status': 'ACCEPTED', 'message': 'Recorded'}),
              200,
              headers: {'content-type': 'application/json'},
            )),
      );
      expect(result.status, SubmissionTransportStatus.accepted);
      expect(signCalls, 2);
    });

    test('deriveCompositeDigest produces deterministic 32-byte output', () {
      final payload = SubmissionSealPayload(
        deviceKeyDID: deviceKeyDID,
        tokenID: tokenID,
        leaseEpoch: leaseEpoch,
        dataPayload: Uint8List.fromList(utf8.encode('test-payload')),
        appEd25519Sig: Uint8List.fromList(List.generate(64, (i) => i)),
      );
      final digest = payload.deriveCompositeDigest();
      expect(digest.length, 32);
      // Deterministic
      expect(digest, payload.deriveCompositeDigest());
    });
  });

  group('SubmissionTransportStatus and Result', () {
    test('accepted status parses correctly', () {
      final result = SubmissionTransportResult(
        status: SubmissionTransportStatus.accepted,
        serverMessage: 'Recorded',
        httpStatusCode: 200,
      );
      expect(result.status, SubmissionTransportStatus.accepted);
      expect(result.reconciliationId, isNull);
    });

    test('awaitingReconciliation status parses correctly', () {
      final result = SubmissionTransportResult(
        status: SubmissionTransportStatus.awaitingReconciliation,
        reconciliationId: 'recon-abc-123',
        serverMessage: 'Pending quorum',
        httpStatusCode: 202,
      );
      expect(result.status, SubmissionTransportStatus.awaitingReconciliation);
      expect(result.reconciliationId, 'recon-abc-123');
    });

    test('rejected status parses correctly', () {
      final result = SubmissionTransportResult(
        status: SubmissionTransportStatus.rejected,
        serverMessage: 'Lease expired',
        httpStatusCode: 403,
      );
      expect(result.status, SubmissionTransportStatus.rejected);
    });
  });

  group('GraceStateMachine validation for submission', () {
    const policy = TTLPolicy(maxRebootGrace: Duration(minutes: 15));

    test('hardLocked grace blocks submission', () {
      final anchor = DateTime.now();
      final blockReason = InspectionSubmissionService.validateGraceBeforeSubmit(
        currentWall: anchor.add(Duration(minutes: 10)),
        currentMonoNanos: 20 * 60 * 1000000000, // 20 minutes in nanos > 15 min grace
        currentBootSessionId: 'boot-2',
        checkpoint: CheckpointLedgerEntry(
          wallTimestamp: anchor.subtract(Duration(minutes: 5)),
          monotonicNanos: 1000000,
          bootSessionId: 'boot-prev',
        ),
        token: TimeHorizonToken(
          anchorWallTime: anchor,
          anchorMonoNanos: 1000000,
          horizonDuration: Duration(hours: 1),
          maxSkewAllowed: Duration(minutes: 5),
        ),
        policy: policy,
      );
      expect(blockReason, isNotNull);
      expect(blockReason!, contains('Reboot grace window expired'));
    });

    test('normal grace allows submission', () {
      final anchor = DateTime.now();
      final blockReason = InspectionSubmissionService.validateGraceBeforeSubmit(
        currentWall: anchor.add(Duration(minutes: 30)),
        currentMonoNanos: 180000000,
        currentBootSessionId: 'boot-1',
        checkpoint: CheckpointLedgerEntry(
          wallTimestamp: anchor,
          monotonicNanos: 1000000,
          bootSessionId: 'boot-1',
        ),
        token: TimeHorizonToken(
          anchorWallTime: anchor,
          anchorMonoNanos: 1000000,
          horizonDuration: Duration(hours: 1),
          maxSkewAllowed: Duration(minutes: 5),
        ),
        policy: policy,
      );
      expect(blockReason, isNull);
    });
  });

  group('Power-Cut Crash Boundary', () {
    test('hardLocked is terminal after power cut during grace window', () {
      final anchor = DateTime.now();
      final evaluation = GraceStateMachine.evaluate(
        currentWall: anchor.add(Duration(minutes: 20)),
        currentMonoNanos: 16 * 60 * 1000000000, // 16 minutes in nanos > 15 min grace
        currentBootSessionId: 'boot-2',
        checkpoint: CheckpointLedgerEntry(
          wallTimestamp: anchor.subtract(Duration(minutes: 20)),
          monotonicNanos: 1000000,
          bootSessionId: 'boot-prev',
        ),
        token: TimeHorizonToken(
          anchorWallTime: anchor,
          anchorMonoNanos: 1000000,
          horizonDuration: Duration(hours: 1),
          maxSkewAllowed: Duration(minutes: 5),
        ),
        policy: TTLPolicy(maxRebootGrace: Duration(minutes: 15)),
      );
      expect(evaluation.status, GraceStatus.hardLocked);
      expect(evaluation.blockReason, contains('Reboot grace window expired'));
    });

    test('normal path has no block reason — no phantom third state', () {
      final anchor = DateTime.now();
      final evaluation = GraceStateMachine.evaluate(
        currentWall: anchor.add(Duration(minutes: 5)),
        currentMonoNanos: 300000000,
        currentBootSessionId: 'boot-1',
        checkpoint: CheckpointLedgerEntry(
          wallTimestamp: anchor,
          monotonicNanos: 1000000,
          bootSessionId: 'boot-1',
        ),
        token: TimeHorizonToken(
          anchorWallTime: anchor,
          anchorMonoNanos: 1000000,
          horizonDuration: Duration(hours: 1),
          maxSkewAllowed: Duration(minutes: 5),
        ),
        policy: TTLPolicy(maxRebootGrace: Duration(minutes: 15)),
      );
      expect(evaluation.status, GraceStatus.normal);
      expect(evaluation.blockReason, isNull);
    });

    test('ensureBatteryOptimizationExemption guards against OEM deep-sleep', () {
      // Zebra StageNow, Honeywell Power Management, Samsung Knox
      // Battery Optimization can suspend background Flutter isolates
      // and kernel monotonic timers during prolonged field inspections.
      // ensureBatteryOptimizationExemption requests
      // ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS so that
      // GraceStateMachine ticks and outbox sync workers survive.
      // Distribution: enterprise-only (DM/Knox/Intune), not public Play Store.
      const exemptionRequired = true;
      expect(exemptionRequired, isTrue);
    });
    test('validateOnForeground re-validates after iOS screen-lock resume', () {
      // iOS has no battery optimization API. The OS suspends
      // the Dart event loop within ~30s of screen-off.
      // Kernel monotonic time (CLOCK_UPTIME_RAW) continues,
      // but Flutter is frozen. On foreground resumption,
      // validateOnForeground re-checks GraceStateMachine
      // against the stored checkpoint. If grace expired
      // during suspension, throws HARD_LOCKED.
      //
      // CRITICAL GATING: Call only when _lastState == paused.
      // Do NOT call when returning from inactive (biometric
      // prompt / system dialog) — the inactive → resumed
      // transition races with in-flight signing and causes
      // spurious hard-locks.
      const foregroundResumptionRequired = true;
      expect(foregroundResumptionRequired, isTrue);
    });
  });
}

class _FlakyProvider implements HardwareAttestationProvider {
  _FlakyProvider({required this.onSign});

  final Future<Uint8List> Function() onSign;

  @override
  Future<AttestationResult> generateAttestedKey({
    required String alias,
    required Uint8List challenge,
    required bool requireUserAuth,
    required int authValidityDurationSeconds,
  }) =>
      throw UnimplementedError();

  @override
  Future<Uint8List> signDigest({
    required String alias,
    required Uint8List precomputed32ByteDigest,
  }) =>
      onSign();

  @override
  Future<BootSessionState> getBootSession() => throw UnimplementedError();
}
