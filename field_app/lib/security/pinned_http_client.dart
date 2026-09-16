library;

import 'dart:convert';
import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';

import 'endpoint_guard.dart';

/// TLS-pinned HTTP client factory.
///
/// - Enforces [assertEndpointSafe] fail-closed (no non-loopback cleartext).
/// - Loopback endpoints return a plain client (pilot debug only, caller must
///   pass allowLoopbackHttp:true which main.dart already restricts).
/// - Non-loopback https with `INTEGIN_TLS_PINS` (comma-separated base64
///   SHA-256 SPKI pins via --dart-define) validates the leaf cert chain;
///   mismatch → connection rejected. Pins empty → system CAs (pilot; production
///   distributions must set pins — see network_security_config TODO).

class PinnedHttpClient {
  PinnedHttpClient._();

  static const pins = String.fromEnvironment('INTEGIN_TLS_PINS');

  static http.Client forEndpoint(Uri endpoint,
      {bool allowLoopbackHttp = false, HttpClient? inner}) {
    assertEndpointSafe(endpoint, allowLoopbackHttp: allowLoopbackHttp);
    final pinList = pins
        .split(',')
        .map((e) => e.trim())
        .where((e) => e.isNotEmpty)
        .toList(growable: false);
    if (isLoopbackUri(endpoint) || pinList.isEmpty) return http.Client();
    final io = inner ?? HttpClient();
    io.badCertificateCallback = (cert, host, port) {
      try {
        final spki = base64Encode(sha256.convert(cert.der).bytes);
        return pinList.contains(spki);
      } catch (_) {
        return false;
      }
    };
    return IOClient(io);
  }
}
