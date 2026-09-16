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
}
