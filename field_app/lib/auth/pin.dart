import 'dart:math';
import 'dart:typed_data';

/// Hash policy seam for PIN hashing.
///
/// The default [IdentityHashPolicy] passes through caller-hashed bytes unchanged,
/// preserving the current contract where the caller provides the hash.
///
/// **Production contract:** Pass scrypt (N=2^15, r=8, p=1) or argon2id output
/// as the raw bytes. NEVER pass raw PIN bytes — hash before calling enroll.
/// The policy is responsible for both hashing raw input and verifying a
/// candidate against a stored hash.
abstract class HashPolicy {
  /// Hash raw PIN bytes. The identity default returns a defensive copy.
  Uint8List hash(Uint8List rawPin);

  /// Verify [candidate] against [storedHash] in constant time.
  bool verify(Uint8List storedHash, Uint8List candidate);
}

/// Identity hash policy — caller is responsible for hashing before enroll.
/// Preserves current behavior: enroll stores caller-provided bytes, verify
/// does constant-time comparison.
class IdentityHashPolicy implements HashPolicy {
  const IdentityHashPolicy();

  @override
  Uint8List hash(Uint8List rawPin) => Uint8List.fromList(rawPin);

  @override
  bool verify(Uint8List storedHash, Uint8List candidate) {
    return _constantTimeEqual(storedHash, candidate);
  }

  static bool _constantTimeEqual(List<int> a, List<int> b) {
    var diff = a.length ^ b.length;
    final n = a.length < b.length ? b.length : a.length;
    for (var i = 0; i < n; i++) {
      diff |= (i < a.length ? a[i] : 0) ^ (i < b.length ? b[i] : 0);
    }
    return diff == 0;
  }
}

/// Persistent storage seam for PIN hash and attempt state.
///
/// Default implementation is in-memory (test-friendly). Production impl
/// wraps flutter_secure_storage or platform enclave.
abstract class PinStore {
  Future<Uint8List?> readPinHash();
  Future<void> writePinHash(Uint8List hash);
  Future<int> readFailCount();
  Future<void> writeFailCount(int count);
  Future<DateTime?> readLockoutUntil();
  Future<void> writeLockoutUntil(DateTime? dt);
  Future<void> clearAll();

  /// Duress hash persistence — separate key namespace from normal PIN.
  Future<Uint8List?> readDuressHash();
  Future<void> writeDuressHash(Uint8List hash);
  Future<void> clearDuressHash();
}

/// In-memory [PinStore] for unit tests — no persistence across isolates.
class InMemoryPinStore implements PinStore {
  Uint8List? _hash;
  int _failCount = 0;
  DateTime? _lockoutUntil;
  Uint8List? _duressHash;

  @override
  Future<Uint8List?> readPinHash() async => _hash;

  @override
  Future<void> writePinHash(Uint8List hash) async => _hash = hash;

  @override
  Future<int> readFailCount() async => _failCount;

  @override
  Future<void> writeFailCount(int count) async => _failCount = count;

  @override
  Future<DateTime?> readLockoutUntil() async => _lockoutUntil;

  @override
  Future<void> writeLockoutUntil(DateTime? dt) async => _lockoutUntil = dt;

  @override
  Future<void> clearAll() async {
    _hash = null;
    _failCount = 0;
    _lockoutUntil = null;
    _duressHash = null;
  }

  @override
  Future<Uint8List?> readDuressHash() async => _duressHash;

  @override
  Future<void> writeDuressHash(Uint8List hash) async => _duressHash = hash;

  @override
  Future<void> clearDuressHash() async => _duressHash = null;
}

/// PIN authentication gate for ATEX Zone 1 (glove-friendly).
///
/// Constant-time compare, exponential backoff, configurable lockout threshold.
/// PIN bytes are zeroed after each verification attempt.
class EnclavePinGate {
  EnclavePinGate({
    required this.store,
    this.maxAttempts = 5,
    this.baseBackoff = const Duration(seconds: 1),
    HashPolicy? hashPolicy,
  }) : hashPolicy = hashPolicy ?? const IdentityHashPolicy();

  final PinStore store;
  final int maxAttempts;
  final Duration baseBackoff;
  final HashPolicy hashPolicy;

  /// Attempt to verify [pin] against the stored hash.
  ///
  /// Returns [PinResult] — callers never see raw hash bytes.
  /// PIN bytes in [pin] are zeroed before return.
  Future<PinResult> verify(Uint8List pin) async {
    final lockoutUntil = await store.readLockoutUntil();
    if (lockoutUntil != null && DateTime.now().isBefore(lockoutUntil)) {
      _zeroPin(pin);
      return PinResult.lockedOut(until: lockoutUntil);
    }

    final stored = await store.readPinHash();
    if (stored == null) {
      _zeroPin(pin);
      return PinResult.notEnrolled();
    }

    final match = hashPolicy.verify(stored, pin);
    _zeroPin(pin);

    if (match) {
      await store.writeFailCount(0);
      await store.writeLockoutUntil(null);
      return PinResult.ok();
    }

    final failCount = await store.readFailCount() + 1;
    await store.writeFailCount(failCount);

    if (failCount >= maxAttempts) {
      final backoffMs =
          baseBackoff.inMilliseconds * pow(2, failCount - maxAttempts).toInt();
      final until = DateTime.now().add(Duration(milliseconds: backoffMs));
      await store.writeLockoutUntil(until);
      return PinResult.lockedOut(until: until);
    }

    return PinResult.wrong(attemptsRemaining: maxAttempts - failCount);
  }

  /// Enroll or replace the stored PIN hash.
  Future<void> enroll(Uint8List pin) async {
    final hashed = hashPolicy.hash(pin);
    await store.writePinHash(hashed);
    await store.writeFailCount(0);
    await store.writeLockoutUntil(null);
    _zeroPin(pin);
  }

  static void _zeroPin(Uint8List pin) {
    for (var i = 0; i < pin.length; i++) {
      pin[i] = 0;
    }
  }
}

/// Result of a PIN verification attempt.
sealed class PinResult {
  PinResult._();

  /// PIN matched; caller is authenticated.
  factory PinResult.ok() = PinOk;

  /// PIN incorrect; [attemptsRemaining] before lockout.
  factory PinResult.wrong({required int attemptsRemaining}) = PinWrong;

  /// Lockout active until [until].
  factory PinResult.lockedOut({required DateTime until}) = PinLockedOut;

  /// No PIN enrolled yet.
  factory PinResult.notEnrolled() = PinNotEnrolled;
}

class PinOk extends PinResult {
  PinOk() : super._();
}

class PinWrong extends PinResult {
  PinWrong({required this.attemptsRemaining}) : super._();
  final int attemptsRemaining;
}

class PinLockedOut extends PinResult {
  PinLockedOut({required this.until}) : super._();
  final DateTime until;
}

class PinNotEnrolled extends PinResult {
  PinNotEnrolled() : super._();
}
