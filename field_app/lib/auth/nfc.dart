import 'dart:typed_data';

/// NFC smartcard reader seam for ATEX Zone 1.
///
/// Wire contract (documented here, NOT enforced server-side by this file):
///
///   Reader  →  Card:  SELECT [AID]             (ISO 7816-4)
///   Card    →  Reader: ATR (Answer-to-Reset)   (historical bytes include applet version)
///   Reader  →  Card:  CHALLENGE [16-byte nonce]
///   Card    →  Reader: RESPONSE [Ed25519-Sign(nonce, card_pubkey)]
///   Reader  →  Card:  GETUid
///   Card    →  Reader: UID (7-byte NFCID2)
///
/// ATR historical bytes layout:
///   [0..2]   applet ID (3 bytes, e.g. 0x010203)
///   [3]      protocol version (0x01)
///   [4..6]   key slot flags (bit 0 = Ed25519, bit 1 = P-256)
///
/// Challenge-response verifies that the card holds the private key bound at
/// enrollment; the server stores the public key and checks the signature.
abstract class SmartcardReader {
  /// Reset the reader hardware; returns the raw ATR bytes from the card.
  Future<Uint8List> reset();

  /// Send [challenge] (16 bytes) and receive the card's signature.
  Future<Uint8List> challengeResponse(Uint8List challenge);

  /// Read the card UID (typically 7 bytes for NFCID2).
  Future<Uint8List> getUid();
}

/// Parsed Answer-to-Reset from a field smartcard.
class AtrInfo {
  const AtrInfo({
    required this.raw,
    required this.appletId,
    required this.protocolVersion,
    required this.keySlotFlags,
  });

  factory AtrInfo.parse(Uint8List atr) {
    if (atr.length < 7) {
      throw FormatException('ATR too short: ${atr.length} bytes');
    }
    return AtrInfo(
      raw: atr,
      appletId: Uint8List.fromList(atr.sublist(0, 3)),
      protocolVersion: atr[3],
      keySlotFlags: Uint8List.fromList(atr.sublist(4, 7)),
    );
  }

  final Uint8List raw;
  final Uint8List appletId;
  final int protocolVersion;
  final Uint8List keySlotFlags;

  bool get supportsEd25519 => keySlotFlags[0] & 0x01 != 0;
  bool get supportsP256 => keySlotFlags[0] & 0x02 != 0;
}

/// Outcome of a smartcard challenge-response verification.
sealed class VerifyResult {
  VerifyResult._();

  factory VerifyResult.ok({
    required Uint8List cardUid,
    required AtrInfo atr,
  }) = VerifyOk;

  factory VerifyResult.noCard() = VerifyNoCard;

  factory VerifyResult.signatureInvalid() = VerifySignatureInvalid;

  factory VerifyResult.unsupportedKeySlot() = VerifyUnsupportedKeySlot;

  factory VerifyResult.readerError(String detail) = VerifyReaderError;
}

class VerifyOk extends VerifyResult {
  VerifyOk({required this.cardUid, required this.atr}) : super._();
  final Uint8List cardUid;
  final AtrInfo atr;
}

class VerifyNoCard extends VerifyResult {
  VerifyNoCard() : super._();
}

class VerifySignatureInvalid extends VerifyResult {
  VerifySignatureInvalid() : super._();
}

class VerifyUnsupportedKeySlot extends VerifyResult {
  VerifyUnsupportedKeySlot() : super._();
}

class VerifyReaderError extends VerifyResult {
  VerifyReaderError(this.detail) : super._();
  final String detail;
}

/// Fake [SmartcardReader] for unit tests — returns predetermined ATR and
/// always verifies against a known keypair.
class FakeSmartcardReader implements SmartcardReader {
  FakeSmartcardReader({
    required Uint8List cardUid,
    required Uint8List cardSignature,
    Uint8List? atr,
  })  : _cardUid = cardUid,
        _cardSignature = cardSignature,
        _atr = atr ?? _defaultAtr();

  final Uint8List _cardUid;
  final Uint8List _cardSignature;
  final Uint8List _atr;

  @override
  Future<Uint8List> reset() async => _atr;

  @override
  Future<Uint8List> challengeResponse(Uint8List challenge) async =>
      _cardSignature;

  @override
  Future<Uint8List> getUid() async => _cardUid;

  static Uint8List _defaultAtr() =>
      Uint8List.fromList([0x01, 0x02, 0x03, 0x01, 0x01, 0x00, 0x00]);
}

/// Verify a smartcard round-trip: read ATR → challenge → check → read UID.
///
/// The actual Ed25519 signature verification is the caller's responsibility
/// (use `package:cryptography` in production). This function validates the
/// protocol shape and returns the parsed artifacts.
Future<VerifyResult> verifySmartcard({
  required SmartcardReader reader,
  required Uint8List challenge,
}) async {
  if (challenge.length != 16) {
    return VerifyReaderError('challenge must be 16 bytes, got ${challenge.length}');
  }

  final atr = await reader.reset();
  final parsed = AtrInfo.parse(atr);

  if (!parsed.supportsEd25519) {
    return VerifyUnsupportedKeySlot();
  }

  final signature = await reader.challengeResponse(challenge);
  if (signature.isEmpty) {
    return VerifyNoCard();
  }

  final uid = await reader.getUid();
  return VerifyOk(cardUid: uid, atr: parsed);
}
