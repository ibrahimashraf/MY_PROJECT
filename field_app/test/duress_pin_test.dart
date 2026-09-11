import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/auth/duress.dart';
import 'package:integin_field_app/auth/pin.dart';

void main() {
  late InMemoryPinStore store;
  late EnclavePinGate normalGate;
  late DuressPin duress;

  setUp(() {
    store = InMemoryPinStore();
    normalGate = EnclavePinGate(store: store);
    duress = DuressPin(pinStore: store, normalPinGate: normalGate);
  });

  test('duress accepted on correct duress PIN', () async {
    await duress.enrollDuress(Uint8List.fromList([1, 1, 1]));
    final result = await duress.verify(Uint8List.fromList([1, 1, 1]));
    expect(result, isA<DuressAccepted>());
  });

  test('duress wrong on incorrect PIN', () async {
    await duress.enrollDuress(Uint8List.fromList([1, 1, 1]));
    final result = await duress.verify(Uint8List.fromList([2, 2, 2]));
    expect(result, isA<DuressWrong>());
  });

  test('notEnrolled when duress PIN not set', () async {
    final result = await duress.verify(Uint8List.fromList([1, 1, 1]));
    expect(result, isA<DuressNotEnrolled>());
  });

  test('duress accepted quarantine payload contains the flag', () async {
    await duress.enrollDuress(Uint8List.fromList([7, 7, 7]));
    final result = await duress.verify(Uint8List.fromList([7, 7, 7]));
    expect(result, isA<DuressAccepted>());
    final accepted = result as DuressAccepted;
    expect(accepted.quarantinePayload['flag'], kCoercionQuarantineFlag);
    expect(accepted.quarantinePayload['coercion_quarantine'], isTrue);
    expect(accepted.quarantinePayload.containsKey('queued_at'), isTrue);
  });

  test('duress outbox metadata is JSON-serializable', () async {
    await duress.enrollDuress(Uint8List.fromList([3]));
    final result = await duress.verify(Uint8List.fromList([3]));
    final accepted = result as DuressAccepted;
    expect(() => Map<String, Object?>.from(accepted.outboxMetadata), returnsNormally);
  });

  test('buildSilentAlarmPayload merges quarantine + device/user', () async {
    final payload = buildSilentAlarmPayload(
      deviceId: 'dev-001',
      userId: 'u-42',
      quarantinePayload: {
        'coercion_quarantine': true,
        'flag': kCoercionQuarantineFlag,
        'queued_at': '2026-01-01T00:00:00Z',
      },
    );
    expect(payload['event'], 'SILENT_ALARM');
    expect(payload['device_id'], 'dev-001');
    expect(payload['flag'], kCoercionQuarantineFlag);
  });

  test('duress PIN bytes zeroed after verify', () async {
    await duress.enrollDuress(Uint8List.fromList([5, 5, 5]));
    final pin = Uint8List.fromList([5, 5, 5]);
    await duress.verify(pin);
    expect(pin, [0, 0, 0]);
  });

  test('indistinguishable: both normal and duress are success-path sealed types', () async {
    await normalGate.enroll(Uint8List.fromList([1, 2, 3]));
    await duress.enrollDuress(Uint8List.fromList([1, 2, 3]));

    final normalResult = await normalGate.verify(Uint8List.fromList([1, 2, 3]));
    final duressResult = await duress.verify(Uint8List.fromList([1, 2, 3]));

    expect(normalResult, isA<PinOk>());
    expect(duressResult, isA<DuressAccepted>());
  });

  test('duress hash persists across simulated restart', () async {
    await duress.enrollDuress(Uint8List.fromList([9, 9, 9]));

    // Simulate restart: new DuressPin instance with same PinStore.
    final restarted = DuressPin(
      pinStore: store,
      normalPinGate: EnclavePinGate(store: store),
    );
    final result = await restarted.verify(Uint8List.fromList([9, 9, 9]));
    expect(result, isA<DuressAccepted>());
  });

  test('wipe clears duress enrollment', () async {
    await duress.enrollDuress(Uint8List.fromList([4, 4, 4]));
    final before = await duress.verify(Uint8List.fromList([4, 4, 4]));
    expect(before, isA<DuressAccepted>());

    await duress.wipeDuress();
    final after = await duress.verify(Uint8List.fromList([4, 4, 4]));
    expect(after, isA<DuressNotEnrolled>());
  });

  test('custom HashPolicy is honored on enroll and verify', () async {
    final logged = <String>[];
    final policy = _SpyHashPolicy(logged);
    final g = DuressPin(
      pinStore: store,
      normalPinGate: normalGate,
      hashPolicy: policy,
    );

    await g.enrollDuress(Uint8List.fromList([1, 2, 3]));
    expect(logged, contains('hash'));

    final result = await g.verify(Uint8List.fromList([1, 2, 3]));
    expect(result, isA<DuressAccepted>());
    expect(logged, contains('verify'));
  });

  test('custom HashPolicy verify failure rejects', () async {
    final policy = _RejectAllHashPolicy();
    final g = DuressPin(
      pinStore: store,
      normalPinGate: normalGate,
      hashPolicy: policy,
    );
    await g.enrollDuress(Uint8List.fromList([1, 2, 3]));
    final result = await g.verify(Uint8List.fromList([1, 2, 3]));
    expect(result, isA<DuressWrong>());
  });
}

class _SpyHashPolicy implements HashPolicy {
  _SpyHashPolicy(this.log);
  final List<String> log;

  @override
  Uint8List hash(Uint8List rawPin) {
    log.add('hash');
    return Uint8List.fromList(rawPin);
  }

  @override
  bool verify(Uint8List storedHash, Uint8List candidate) {
    log.add('verify');
    return const IdentityHashPolicy().verify(storedHash, candidate);
  }
}

class _RejectAllHashPolicy implements HashPolicy {
  @override
  Uint8List hash(Uint8List rawPin) => Uint8List.fromList(rawPin);

  @override
  bool verify(Uint8List storedHash, Uint8List candidate) => false;
}
