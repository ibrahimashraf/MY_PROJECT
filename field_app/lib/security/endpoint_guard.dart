/// INTEGIN Field App — Endpoint security guard.
///
/// Red-team rationale: every HTTP client in the codebase accepts a [Uri]
/// from outside (env vars, provisioning JSON, advisory config). Without a
/// guard a misconfigured prod deploy could silently target `http://` and
/// leak signed mutations and device keys in plaintext.
///
/// Rules enforced (fail-closed):
///   • Loopback (127.0.0.1 / ::1 / localhost) may use http or https.
///   • Any non-loopback host MUST use https in non-pilot builds.
///   • Empty scheme or unknown scheme → rejected.
///   • Pilot flag bypasses the https-only check for the loopback-only pilot
///     server, but the loopback restriction is still enforced by main.dart.
library;

/// Validates that [uri] is safe to use as a INTEGIN network endpoint.
///
/// Throws [ArgumentError] if the endpoint fails the policy, so callers that
/// skip validation are caught at construction time — never at first request.
///
/// [allowLoopbackHttp] should only be true in pilot/debug mode.
void assertEndpointSafe(Uri uri, {bool allowLoopbackHttp = false}) {
  if (uri.scheme.isEmpty) {
    throw ArgumentError.value(uri, 'uri', 'endpoint scheme must not be empty');
  }
  if (uri.scheme != 'https' && uri.scheme != 'http') {
    throw ArgumentError.value(
        uri, 'uri', 'endpoint scheme must be https or http, got ${uri.scheme}');
  }
  final isLoopback = uri.host == '127.0.0.1' ||
      uri.host == 'localhost' ||
      uri.host == '::1';
  if (uri.scheme == 'http' && !isLoopback) {
    throw ArgumentError.value(
      uri,
      'uri',
      // ignore: lines_longer_than_80_chars
      'cleartext http:// is forbidden for non-loopback hosts — use https:// '
      '(host: ${uri.host})',
    );
  }
  if (uri.scheme == 'http' && isLoopback && !allowLoopbackHttp) {
    throw ArgumentError.value(
      uri,
      'uri',
      'cleartext http:// on loopback is only permitted in pilot mode '
      '(host: ${uri.host})',
    );
  }
}

/// Returns true iff [uri] is a loopback address.
bool isLoopbackUri(Uri uri) =>
    uri.host == '127.0.0.1' ||
    uri.host == 'localhost' ||
    uri.host == '::1';
