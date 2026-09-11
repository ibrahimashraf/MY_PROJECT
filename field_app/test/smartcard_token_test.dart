import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/auth/nfc.dart';

void main() {
  test('FakeSmartcardReader round-trip: reset → challenge → uid', () async {
    final uid = Uint8List.fromList([0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF, 0x11]);
    final sig = Uint8List.fromList(List.generate(64, (i) => i & 0xFF));
    final reader = FakeSmartcardReader(cardUid: uid, cardSignature: sig);

    final atr = await reader.reset();
    expect(atr.length, 7);

    final parsed = AtrInfo.parse(atr);
    expect(parsed.protocolVersion, 0x01);
    expect(parsed.supportsEd25519, isTrue);
    expect(parsed.supportsP256, isFalse);

    final response = await reader.challengeResponse(Uint8List(16));
    expect(response, sig);

    final readUid = await reader.getUid();
    expect(readUid, uid);
  });

  test('verifySmartcard returns ok for valid fake reader', () async {
    final uid = Uint8List.fromList([0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07]);
    final sig = Uint8List.fromList(List.generate(64, (i) => 0x42));
    final reader = FakeSmartcardReader(cardUid: uid, cardSignature: sig);
    final challenge = Uint8List(16);

    final result = await verifySmartcard(reader: reader, challenge: challenge);
    expect(result, isA<VerifyOk>());
    final ok = result as VerifyOk;
    expect(ok.cardUid, uid);
    expect(ok.atr.supportsEd25519, isTrue);
  });

  test('verifySmartcard rejects wrong challenge length', () async {
    final uid = Uint8List.fromList([0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07]);
    final sig = Uint8List.fromList(List.generate(64, (i) => 0x42));
    final reader = FakeSmartcardReader(cardUid: uid, cardSignature: sig);

    final result = await verifySmartcard(reader: reader, challenge: Uint8List(8));
    expect(result, isA<VerifyReaderError>());
  });

  test('ATrInfo.parse rejects short ATR', () {
    expect(() => AtrInfo.parse(Uint8List(3)), throwsFormatException);
  });

  test('ATrInfo key slot flags parsed correctly', () {
    // byte[0] = 0x03 → bit 0 (Ed25519) and bit 1 (P-256) both set
    final atr = Uint8List.fromList([0x01, 0x02, 0x03, 0x01, 0x03, 0x00, 0x00]);
    final parsed = AtrInfo.parse(atr);
    expect(parsed.supportsEd25519, isTrue);
    expect(parsed.supportsP256, isTrue);
  });
}
