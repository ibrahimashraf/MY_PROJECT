import 'dart:typed_data';

import 'pin.dart';

/// Coercion quarantine flag embedded in outbox metadata.
///
/// Never surfaced in UI — the operator who enters a duress PIN sees the same
/// success screen as a normal PIN entry. The flag is carried on locally queued
/// submissions and picked up by the server-side sync path.
const String kCoercionQuarantineFlag = 'STATE_COERCION_QUARANTINE';

/// Duress PIN authenticator — same UX path as [EnclavePinGate] but sets a
/// covert quarantine flag on the authentication result.
///
/// The duress PIN is enrolled separately from the normal PIN. On success the
/// caller receives [DuressAuthResult] which carries [isDuress]=true and a
/// quarantine metadata map suitable for attachment to outbox entries.
class DuressPin {
  DuressPin({
    required this.pinStore,
    required this.normalPinGate,
  });

  final PinStore pinStore;
  final EnclavePinGate normalPinGate;

  Uint8List? _duressHash;

  /// Enroll a duress PIN. Caller is responsible for hashing before passing.
  Future<void> enrollDuress(Uint8List pin) async {
    _duressHash = Uint8List.fromList(pin);
    for (var i = 0; i < pin.length; i++) {
      pin[i] = 0;
    }
  }

  /// Verify [pin] against the duress hash.
  ///
  /// Returns [DuressAuthResult] — same sealed hierarchy as [PinResult] so
  /// callers can pattern-match identically to normal PIN verification.
  /// On success the result carries the quarantine metadata payload.
  Future<DuressAuthResult> verify(Uint8List pin) async {
    if (_duressHash == null) {
      _zeroPin(pin);
      return DuressAuthResult.notEnrolled();
    }

    final match = _constantTimeEqual(pin, _duressHash!);
    _zeroPin(pin);

    if (match) {
      return DuressAuthResult.duressAccepted(
        quarantinePayload: _buildQuarantinePayload(),
      );
    }
    return DuressAuthResult.wrong();
  }

  /// Build the metadata map attached to every outbox entry when duress is active.
  ///
  /// The server's sync handler checks for [kCoercionQuarantineFlag] in the
  /// metadata envelope and triggers the silent alarm pipeline.  The payload
  /// is queued like normal sync traffic — no special network path.
  static Map<String, Object?> _buildQuarantinePayload() => {
        'coercion_quarantine': true,
        'flag': kCoercionQuarantineFlag,
        'queued_at': DateTime.now().toUtc().toIso8601String(),
      };

  static bool _constantTimeEqual(List<int> a, List<int> b) {
    var diff = a.length ^ b.length;
    final n = a.length < b.length ? b.length : a.length;
    for (var i = 0; i < n; i++) {
      diff |= (i < a.length ? a[i] : 0) ^ (i < b.length ? b[i] : 0);
    }
    return diff == 0;
  }

  static void _zeroPin(Uint8List pin) {
    for (var i = 0; i < pin.length; i++) {
      pin[i] = 0;
    }
  }
}

/// Silence duress alarm payload for a single submission.
///
/// Queued through the normal sync transport — no separate network channel.
Map<String, Object?> buildSilentAlarmPayload({
  required String deviceId,
  required String userId,
  required Map<String, Object?> quarantinePayload,
}) =>
    <String, Object?>{
      'event': 'SILENT_ALARM',
      'device_id': deviceId,
      'user_id': userId,
      ...quarantinePayload,
    };

/// Result of a duress PIN verification — same UX as normal [PinResult].
sealed class DuressAuthResult {
  DuressAuthResult._();

  factory DuressAuthResult.duressAccepted({
    required Map<String, Object?> quarantinePayload,
  }) = DuressAccepted;

  factory DuressAuthResult.wrong() = DuressWrong;

  factory DuressAuthResult.notEnrolled() = DuressNotEnrolled;
}

class DuressAccepted extends DuressAuthResult {
  DuressAccepted({required this.quarantinePayload}) : super._();
  final Map<String, Object?> quarantinePayload;

  /// Attach this to the outbox entry metadata — never shown in UI.
  Map<String, Object?> get outboxMetadata => {
        ...quarantinePayload,
      };
}

class DuressWrong extends DuressAuthResult {
  DuressWrong() : super._();
}

class DuressNotEnrolled extends DuressAuthResult {
  DuressNotEnrolled() : super._();
}
