import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;

import '../security/endpoint_guard.dart';
import '../security/pinned_http_client.dart';

class StationEvent {
  const StationEvent({
    required this.id,
    required this.type,
    required this.payload,
    required this.timestamp,
  });

  final String id;
  final String type;
  final Map<String, Object?> payload;
  final DateTime timestamp;

  factory StationEvent.fromJson(Map<String, Object?> json) {
    final rawPayload = json['payload'];
    final Map<String, Object?> payloadMap;
    if (rawPayload is Map<String, Object?>) {
      payloadMap = rawPayload;
    } else if (rawPayload is String) {
      payloadMap = jsonDecode(rawPayload) as Map<String, Object?>;
    } else {
      payloadMap = const {};
    }

    return StationEvent(
      id: json['id'] as String? ?? '',
      type: json['type'] as String? ?? 'general',
      payload: payloadMap,
      timestamp: json['timestamp'] != null
          ? DateTime.parse(json['timestamp'] as String)
          : DateTime.now().toUtc(),
    );
  }
}

/// Real-time Server-Sent Events (SSE) client for Flutter workstations and mobile dashboards.
class EventStreamClient {
  EventStreamClient({
    required this.endpoint,
    required this.tenantId,
    http.Client? client,
    bool allowLoopbackHttp = false,
  }) : _client = client ??
            PinnedHttpClient.forEndpoint(endpoint,
                allowLoopbackHttp: allowLoopbackHttp) {
    assertEndpointSafe(endpoint, allowLoopbackHttp: allowLoopbackHttp);
  }

  final Uri endpoint;
  final String tenantId;
  final http.Client _client;

  StreamSubscription<String>? _subscription;
  final _eventController = StreamController<StationEvent>.broadcast();

  Stream<StationEvent> get events => _eventController.stream;

  bool _disposed = false;
  int _retrySeconds = 1;

  Future<void> connect() async {
    _disposed = false;
    _startConnection();
  }

  Future<void> _startConnection() async {
    if (_disposed) return;
    try {
      final request = http.Request('GET', endpoint.replace(queryParameters: {'tenant_id': tenantId}))
        ..headers['Accept'] = 'text/event-stream'
        ..headers['Cache-Control'] = 'no-cache';

      final response = await _client.send(request);
      if (response.statusCode != 200) {
        _scheduleReconnect();
        return;
      }

      _retrySeconds = 1; // Reset backoff upon successful connection

      _subscription = response.stream
          .transform(utf8.decoder)
          .transform(const LineSplitter())
          .listen(
            _handleLine,
            onError: (_) => _scheduleReconnect(),
            onDone: _scheduleReconnect,
            cancelOnError: true,
          );
    } catch (_) {
      _scheduleReconnect();
    }
  }

  void _scheduleReconnect() {
    if (_disposed) return;
    _subscription?.cancel();
    _subscription = null;

    final wait = Duration(seconds: _retrySeconds);
    _retrySeconds = (_retrySeconds * 2).clamp(1, 30); // Max backoff 30s
    Future.delayed(wait, _startConnection);
  }

  void _handleLine(String line) {
    if (line.startsWith('data: ')) {
      final jsonStr = line.substring(6).trim();
      if (jsonStr.isEmpty || jsonStr.startsWith('{"status":"connected"')) {
        return;
      }
      try {
        final data = jsonDecode(jsonStr) as Map<String, Object?>;
        _eventController.add(StationEvent.fromJson(data));
      } catch (_) {
        // Ignore unparseable frames
      }
    }
  }

  Future<void> disconnect() async {
    _disposed = true;
    await _subscription?.cancel();
    _subscription = null;
  }

  void dispose() {
    _disposed = true;
    disconnect();
    _eventController.close();
    _client.close();
  }
}
