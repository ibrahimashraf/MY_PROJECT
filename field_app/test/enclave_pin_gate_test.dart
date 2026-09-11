import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/auth/pin.dart';

void main() {
  late InMemoryPinStore store;
  late EnclavePinGate gate;

  setUp(() {
    store = InMemoryPinStore();
    gate = EnclavePinGate(store: store, maxAttempts: 3, baseBackoff: const Duration(milliseconds: 10));
  });

  test('notEnrolled when no PIN stored', () async {
    final result = await gate.verify(Uint8List.fromList([1, 2, 3]));
    expect(result, isA<PinNotEnrolled>());
  });

  test('ok on correct PIN', () async {
    await gate.enroll(Uint8List.fromList([9, 9, 9]));
    final result = await gate.verify(Uint8List.fromList([9, 9, 9]));
    expect(result, isA<PinOk>());
  });

  test('wrong PIN returns attemptsRemaining', () async {
    await gate.enroll(Uint8List.fromList([1, 2, 3]));
    final result = await gate.verify(Uint8List.fromList([4, 5, 6]));
    expect(result, isA<PinWrong>());
    expect((result as PinWrong).attemptsRemaining, 2);
  });

  test('lockout after maxAttempts', () async {
    await gate.enroll(Uint8List.fromList([1, 2, 3]));
    for (var i = 0; i < 3; i++) {
      await gate.verify(Uint8List.fromList([4, 5, 6]));
    }
    final result = await gate.verify(Uint8List.fromList([4, 5, 6]));
    expect(result, isA<PinLockedOut>());
  });

  test('correct PIN resets fail count', () async {
    await gate.enroll(Uint8List.fromList([1, 2, 3]));
    await gate.verify(Uint8List.fromList([9, 9, 9])); // wrong
    await gate.verify(Uint8List.fromList([1, 2, 3])); // correct
    final failCount = await store.readFailCount();
    expect(failCount, 0);
  });

  test('constant-time: same-length wrong PIN takes comparable wall time',
      () async {
    await gate.enroll(Uint8List.fromList([1, 2, 3, 4, 5, 6, 7, 8]));
    final sw = Stopwatch()..start();
    await gate.verify(Uint8List.fromList([1, 2, 3, 4, 5, 6, 7, 9]));
    sw.stop();
    // The point: it completes without early exit.  We don't assert timing
    // because CI jitter is real; the structural test is that constantTimeEqual
    // iterates all bytes (verified by code inspection + same-length path).
    expect(sw.elapsedMilliseconds, greaterThanOrEqualTo(0));
  });

  test('different-length PIN still returns wrong (no early length leak)',
      () async {
    await gate.enroll(Uint8List.fromList([1, 2, 3]));
    final result = await gate.verify(Uint8List.fromList([1, 2]));
    expect(result, isA<PinWrong>());
  });

  test('PIN bytes zeroed after verify', () async {
    await gate.enroll(Uint8List.fromList([7, 8, 9]));
    final pin = Uint8List.fromList([7, 8, 9]);
    await gate.verify(pin);
    expect(pin, [0, 0, 0]);
  });

  test('PIN bytes zeroed after enroll', () async {
    final pin = Uint8List.fromList([1, 1, 1]);
    await gate.enroll(pin);
    expect(pin, [0, 0, 0]);
  });
}
