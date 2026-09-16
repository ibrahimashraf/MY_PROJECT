library;

import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:crypto/crypto.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';

import 'endpoint_guard.dart';

/// TLS-pinned HTTP client factory.
///
/// - Enforces [assertEndpointSafe] fail-closed (no non-loopback cleartext).
/// - Loopback endpoints return a plain client (pilot debug only, caller must
///   pass allowLoopbackHttp:true which main.dart restricts to pilot mode).
/// - Non-loopback https with `INTEGIN_TLS_PINS` (comma-separated base64
///   SHA-256 **SPKI** pins via --dart-define) validates the leaf certificate:
///   the SubjectPublicKeyInfo is DER-parsed out of the cert and its SHA-256
///   must match a pin; mismatch → connection rejected. SPKI (not whole-cert)
///   hashing survives re-issuance with key continuity.
/// - Pins empty → system CAs, unless `INTEGIN_REQUIRE_TLS_PINS=true` is set
///   (production builds), which throws instead of silently downgrading.
class PinnedHttpClient {
  PinnedHttpClient._();

  static const pins = String.fromEnvironment('INTEGIN_TLS_PINS');

  static const requirePins = bool.fromEnvironment('INTEGIN_REQUIRE_TLS_PINS');

  static http.Client forEndpoint(Uri endpoint,
      {bool allowLoopbackHttp = false, HttpClient? inner}) {
    assertEndpointSafe(endpoint, allowLoopbackHttp: allowLoopbackHttp);
    final pinList = pins
        .split(',')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList(growable: false);
    if (isLoopbackUri(endpoint)) return http.Client();
    if (pinList.isEmpty) {
      if (requirePins) {
        throw StateError(
          'TLS SPKI pins are required in this build (INTEGIN_TLS_PINS is empty) '
          'for $endpoint',
        );
      }
      return http.Client();
    }
    final io = inner ?? HttpClient();
    io.badCertificateCallback = (cert, host, port) {
      try {
        return pinList.contains(spkiPin(cert.der));
      } catch (_) {
        return false;
      }
    };
    return IOClient(io);
  }

  /// Base64 SHA-256 of the DER-encoded SubjectPublicKeyInfo in [certDer].
  /// Throws [FormatException] on malformed input (fail-closed: rejected).
  static String spkiPin(Uint8List certDer) =>
      base64Encode(sha256.convert(extractSpki(certDer)).bytes);

  /// Extracts the full DER encoding (tag + length + value) of the
  /// SubjectPublicKeyInfo: the first SEQUENCE child of tbsCertificate whose
  /// children are exactly [SEQUENCE (algorithm), BIT STRING (key)].
  static Uint8List extractSpki(Uint8List certDer) {
    final top = _DerNode.parse(certDer, 0);
    if (top.tag != 0x30 || top.children.isEmpty) {
      throw const FormatException('certificate is not a DER SEQUENCE');
    }
    final tbs = top.children[0];
    if (tbs.tag != 0x30) {
      throw const FormatException('tbsCertificate is not a SEQUENCE');
    }
    for (final child in tbs.children) {
      if (child.tag == 0x30 &&
          child.children.length == 2 &&
          child.children[0].tag == 0x30 &&
          child.children[1].tag == 0x03) {
        return child.raw;
      }
    }
    throw const FormatException('SubjectPublicKeyInfo not found');
  }
}

/// Minimal definite-length DER node. Only constructed SEQUENCE (0x30) and
/// context-specific [0] (0xA0, certificate version) are descended into;
/// everything else is kept as opaque content. Indefinite lengths rejected.
class _DerNode {
  _DerNode._(this.tag, this.raw, this.children);

  final int tag;
  final Uint8List raw;
  final List<_DerNode> children;

  static _DerNode parse(Uint8List bytes, int offset) {
    if (offset + 2 > bytes.length) {
      throw const FormatException('truncated DER header');
    }
    final tag = bytes[offset];
    var pos = offset + 1;
    var length = bytes[pos++];
    if (length == 0x80) {
      throw const FormatException('indefinite DER length rejected');
    }
    if (length & 0x80 != 0) {
      final count = length & 0x7F;
      if (count == 0 || count > 4 || pos + count > bytes.length) {
        throw const FormatException('invalid DER length');
      }
      length = 0;
      for (var i = 0; i < count; i += 1) {
        length = (length << 8) | bytes[pos++];
      }
    }
    if (pos + length > bytes.length) {
      throw const FormatException('truncated DER value');
    }
    final raw = bytes.sublist(offset, pos + length);
    final children = <_DerNode>[];
    if (tag == 0x30 || tag == 0xA0) {
      var childPos = pos;
      while (childPos < pos + length) {
        final child = parse(bytes, childPos);
        children.add(child);
        childPos += child.raw.length;
      }
      if (childPos != pos + length) {
        throw const FormatException('DER children overrun value');
      }
    }
    return _DerNode._(tag, raw, children);
  }
}
