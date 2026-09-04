import 'dart:async';
import 'dart:convert';
import 'package:http/http.dart' as http;

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
  }) : _client = client ?? http.Client();

  final Uri endpoint;
  final String tenantId;
  final http.Client _client;

  StreamSubscription<String>? _subscription;
  final _eventController = StreamController<StationEvent>.broadcast();

  Stream<StationEvent> get events => _eventController.stream;

  Future<void> connect() async {
    final request = http.Request('GET', endpoint.replace(queryParameters: {'tenant_id': tenantId}))
      ..headers['Accept'] = 'text/event-stream'
      ..headers['Cache-Control'] = 'no-cache';

    final response = await _client.send(request);
    if (response.statusCode != 200) {
      throw StateError('SSE connection failed with HTTP ${response.statusCode}');
    }

    _subscription = response.stream
        .transform(utf8.decoder)
        .transform(const LineSplitter())
        .listen(_handleLine, onError: _eventController.addError);
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
    await _subscription?.cancel();
    _subscription = null;
  }

  void dispose() {
    disconnect();
    _eventController.close();
    _client.close();
  }
}
