import 'dart:convert';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/security/pinned_http_client.dart';

void main() {
  test('rejects non-loopback cleartext', () {
    expect(
      () => PinnedHttpClient.forEndpoint(Uri.parse('http://example.com/sync')),
      throwsArgumentError,
    );
  });

  test('loopback pilot endpoint allowed', () {
    final client = PinnedHttpClient.forEndpoint(
      Uri.parse('http://127.0.0.1:18080/sync'),
      allowLoopbackHttp: true,
    );
    expect(client, isNotNull);
    client.close();
  });

  test('non-loopback https returns client (system CAs when no pins)', () {
    final client = PinnedHttpClient.forEndpoint(
      Uri.parse('https://example.com/sync'),
    );
    expect(client, isNotNull);
    client.close();
  });

  test('extractSpki returns the exact SPKI bytes from a DER certificate', () {
    // Minimal synthetic cert: SEQ { tbs SEQ { INTEGER, SPKI }, ... }.
    // SPKI = SEQ { SEQ { OID rsaEncryption, NULL }, BIT STRING key }.
    final spki = _derSeq([
      _derSeq([
        Uint8List.fromList(
            const [0x06, 0x09, 0x2A, 0x86, 0x48, 0x86, 0xF7, 0x0D, 0x01, 0x01, 0x01]),
        Uint8List.fromList(const [0x05, 0x00]),
      ]),
      _derBitString(Uint8List.fromList(const [0xDE, 0xAD, 0xBE, 0xEF])),
    ]);
    final cert = _derSeq([
      _derSeq([_derInteger(1), spki]),
      _derSeq([_derInteger(2)]),
    ]);
    expect(PinnedHttpClient.extractSpki(cert), spki);
    expect(
      PinnedHttpClient.spkiPin(cert),
      base64Encode(sha256.convert(spki).bytes),
    );
  });

  test('extractSpki rejects malformed input fail-closed', () {
    expect(
      () => PinnedHttpClient.extractSpki(Uint8List.fromList(const [0x30, 0x00])),
      throwsFormatException,
    );
    expect(
      () => PinnedHttpClient.extractSpki(Uint8List.fromList(const [1, 2, 3])),
      throwsFormatException,
    );
    expect(
      () => PinnedHttpClient.extractSpki(Uint8List.fromList(const [0x30, 0x80])),
      throwsFormatException,
    );
  });
}

Uint8List _derTlv(int tag, List<int> value) {
  final out = <int>[tag];
  if (value.length < 128) {
    out.add(value.length);
  } else {
    final lenBytes = <int>[];
    var remaining = value.length;
    while (remaining > 0) {
      lenBytes.insert(0, remaining & 0xFF);
      remaining >>= 8;
    }
    out.add(0x80 | lenBytes.length);
    out.addAll(lenBytes);
  }
  out.addAll(value);
  return Uint8List.fromList(out);
}

Uint8List _derSeq(List<Uint8List> children) => _derTlv(
      0x30,
      children.expand((child) => child).toList(),
    );

Uint8List _derInteger(int value) => _derTlv(0x02, <int>[value]);

Uint8List _derBitString(Uint8List bytes) =>
    _derTlv(0x03, <int>[0x00, ...bytes]);
