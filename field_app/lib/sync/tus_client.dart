import 'dart:async';
import 'dart:convert';
import 'dart:math' as math;
import 'dart:typed_data';

import 'package:http/http.dart' as http;

/// Server-side cap for a single append, mirrored from
/// `internal/storage/tus_handler.go` (`DefaultTUSChunkSize = 2 << 20`).
const int tusChunkSize = 2 * 1024 * 1024;

/// Upload session metadata returned by the offset endpoint.
class TusOffset {
  const TusOffset({
    required this.id,
    required this.size,
    required this.offset,
    required this.contentType,
    required this.checksum,
  });

  final String id;
  final int size;
  final int offset;
  final String contentType;
  final String checksum;
}

/// Upload completion result returned by the complete endpoint.
class TusCompletion {
  const TusCompletion({required this.key, required this.contentType});

  final String key;
  final String contentType;
}

/// Typed failure surfaced by the TUS client. Callers must fail closed on
/// every kind: none of these are safe to silently skip.
enum TusErrorKind {
  /// Server rejected the request (400/405) or the payload was malformed.
  badRequest,

  /// Server returned 401 — upload session is not authenticated.
  authentication,

  /// Upload id does not exist server-side (404).
  notFound,

  /// Append offset mismatched the server's confirmed offset (409).
  conflict,

  /// Transport-level failure (timeout, socket drop, 5xx) — safe to retry.
  network,

  /// A single append exceeded the server's chunk-size limit.
  tooLarge,

  /// Chunk or upload is empty; the server rejects empty appends.
  empty,

  /// Offset never converged within `maxResumeRounds` re-queries. Never
  /// blind-retry: surface this instead of looping on a stale offset.
  maxResumeExceeded,
}

class TusError implements Exception {
  const TusError(this.kind, this.message, {this.statusCode});

  final TusErrorKind kind;
  final String message;
  final int? statusCode;

  @override
  String toString() =>
      'TusError(${kind.name}${statusCode == null ? '' : ' $statusCode'}): $message';
}

/// SHA-256 hex contract enforced by the server on create.
final RegExp _sha256Hex = RegExp(r'^[a-fA-F0-9]{64}$');

/// True when [value] is a 64-char hex-encoded SHA-256 (server contract).
bool isSha256Hex(String value) => _sha256Hex.hasMatch(value.trim());

/// Typed errors:
///  * create 201 `{"id": ...}` | 400
///  * append 200 `{"offset": next}` | 409 mismatch | 404 missing
///  * offset 200 session | 404 missing
///  * complete 200 `{"key", "content_type"}` | 400 incomplete/checksum | 404
///  * abort 204 | 404
abstract interface class TusHttp {
  /// Create a new upload session, returning its server id.
  Future<String> create({
    required int size,
    required String sha256,
    required String contentType,
  });

  /// Append a chunk at [offset]. Returns the server-confirmed next offset.
  Future<int> append({
    required String id,
    required int offset,
    required Uint8List bytes,
  });

  /// Query the server-confirmed progress of an upload session.
  Future<TusOffset> offset(String id);

  /// Finalize an upload (server verifies offset == size and SHA-256).
  Future<TusCompletion> complete(String id);

  /// Abandon an upload session server-side.
  Future<void> abort(String id);
}

/// `package:http` implementation of [TusHttp] speaking the server contract in
/// `internal/storage/tus_route.go`. One instance is safe to share; it carries
/// no per-upload state.
class HttpTusHttp implements TusHttp {
  HttpTusHttp({
    required Uri baseUrl,
    required http.Client client,
    String? authToken,
    this.timeout = const Duration(seconds: 30),
  })  : _base = baseUrl.toString().replaceAll(RegExp(r'/+$'), ''),
        _client = client,
        _authToken = authToken;

  final String _base;
  final http.Client _client;
  final String? _authToken;
  final Duration timeout;

  Uri _uri(String suffix) => Uri.parse('$_base$suffix');

  Map<String, String> _headers() => {
        'content-type': 'application/json',
        if (_authToken != null) 'authorization': 'Bearer $_authToken',
      };

  Future<http.Response> _send(Future<http.Response> Function() call) async {
    try {
      return await call().timeout(timeout);
    } on TusError {
      rethrow;
    } catch (error) {
      throw TusError(
        TusErrorKind.network,
        'upload transport failed: $error',
      );
    }
  }

  /// Maps every non-success status to a typed [TusError]. 401/409/404 follow
  /// the server contract; anything else unexpected (5xx, 3xx) is [network].
  Never _throwFor(http.Response response) {
    final kind = switch (response.statusCode) {
      400 || 405 => TusErrorKind.badRequest,
      401 || 403 => TusErrorKind.authentication,
      404 => TusErrorKind.notFound,
      409 => TusErrorKind.conflict,
      _ => TusErrorKind.network,
    };
    String message;
    try {
      final payload = jsonDecode(response.body);
      message = (payload is Map && payload['error'] is String)
          ? payload['error'] as String
          : 'http ${response.statusCode}';
    } on FormatException {
      message = 'http ${response.statusCode}';
    }
    if (response.statusCode >= 200 && response.statusCode < 300) {
      message = 'unexpected ${response.statusCode} response: '
          '${response.body.length > 200 ? '${response.body.substring(0, 200)}…' : response.body}';
    }
    throw TusError(kind, message, statusCode: response.statusCode);
  }

  @override
  Future<String> create({
    required int size,
    required String sha256,
    required String contentType,
  }) async {
    final response = await _send(() => _client.post(
          _uri(''),
          headers: _headers(),
          body: jsonEncode({
            'size': size,
            'checksum': sha256,
            'content_type': contentType,
          }),
        ));
    if (response.statusCode != 201) _throwFor(response);
    final id = (jsonDecode(response.body) as Map<String, dynamic>)['id'];
    if (id is! String || id.isEmpty) {
      throw const TusError(TusErrorKind.badRequest,
          'create response is missing a non-empty id');
    }
    return id;
  }

  @override
  Future<int> append({
    required String id,
    required int offset,
    required Uint8List bytes,
  }) async {
    final response = await _send(() => _client.post(
          _uri('/$id/chunks'),
          headers: _headers(),
          body: jsonEncode({
            'offset': offset,
            'data': base64Encode(bytes),
          }),
        ));
    if (response.statusCode != 200) _throwFor(response);
    final confirmed =
        (jsonDecode(response.body) as Map<String, dynamic>)['offset'];
    if (confirmed is! int) {
      throw const TusError(TusErrorKind.badRequest,
          'append response is missing an integer offset');
    }
    return confirmed;
  }

  @override
  Future<TusOffset> offset(String id) async {
    final response = await _send(
        () => _client.get(_uri('/$id/offset'), headers: _headers()));
    if (response.statusCode != 200) _throwFor(response);
    final payload = jsonDecode(response.body) as Map<String, dynamic>;
    return TusOffset(
      id: payload['id'] as String,
      size: (payload['size'] as num).toInt(),
      offset: (payload['offset'] as num).toInt(),
      contentType: payload['content_type'] as String,
      checksum: payload['checksum'] as String,
    );
  }

  @override
  Future<TusCompletion> complete(String id) async {
    final response = await _send(() => _client.post(
          _uri('/$id/complete'),
          headers: _headers(),
        ));
    if (response.statusCode != 200) _throwFor(response);
    final payload = jsonDecode(response.body) as Map<String, dynamic>;
    return TusCompletion(
      key: payload['key'] as String,
      contentType: payload['content_type'] as String,
    );
  }

  @override
  Future<void> abort(String id) async {
    final response = await _send(() => _client.post(
          _uri('/$id/abort'),
          headers: _headers(),
        ));
    if (response.statusCode != 204) _throwFor(response);
  }
}

/// Split [data] into contiguous [chunkSize]-byte pieces (2 MiB default,
/// matching the server cap). The final piece may be shorter.
List<Uint8List> splitChunks(Uint8List data, {int chunkSize = tusChunkSize}) {
  if (chunkSize <= 0) {
    throw ArgumentError.value(chunkSize, 'chunkSize', 'must be positive');
  }
  if (data.isEmpty) return const [];
  final chunks = <Uint8List>[];
  for (var i = 0; i < data.length; i += chunkSize) {
    final end = math.min(i + chunkSize, data.length);
    chunks.add(Uint8List.sublistView(data, i, end));
  }
  return chunks;
}

/// High-level TUS upload orchestrator over an injected [TusHttp] seam.
///
/// Chunk uploads are offset-verified: a 409 conflict re-queries the server
/// offset and resumes there, but never blind-retries the same offset beyond
/// [maxResumeRounds] rounds — it fails closed with [TusErrorKind.maxResumeExceeded].
class TusClient {
  TusClient({
    required this.http,
    this.chunkSize = tusChunkSize,
    this.maxResumeRounds = 3,
  }) {
    if (chunkSize <= 0) {
      throw ArgumentError.value(chunkSize, 'chunkSize', 'must be positive');
    }
    if (maxResumeRounds < 0) {
      throw ArgumentError.value(
          maxResumeRounds, 'maxResumeRounds', 'must not be negative');
    }
  }

  final TusHttp http;
  final int chunkSize;
  final int maxResumeRounds;

  /// Create an upload session, validating the server's fail-closed contract
  /// (positive size, hex SHA-256, non-empty content type) before any request.
  Future<String> createUpload({
    required int size,
    required String sha256,
    required String contentType,
  }) async {
    if (size <= 0) {
      throw const TusError(TusErrorKind.empty, 'upload size must be positive');
    }
    if (!isSha256Hex(sha256)) {
      throw const TusError(
          TusErrorKind.badRequest, 'checksum must be a hex-encoded SHA-256');
    }
    if (contentType.trim().isEmpty) {
      throw const TusError(
          TusErrorKind.badRequest, 'content type is required');
    }
    return http.create(
      size: size,
      sha256: sha256.trim().toLowerCase(),
      contentType: contentType,
    );
  }

  /// Append [bytes] at a base [offset], resuming across 409 offset conflicts.
  ///
  /// On a conflict the server's confirmed offset is re-queried; if it has
  /// advanced within this chunk the remaining suffix is uploaded from there.
  /// If the server offset has not advanced past this chunk's start
  /// ([TusErrorKind.maxResumeExceeded]) or a non-conflict status is returned,
  /// the failure is surfaced — never a silent loop on a stale offset.
  Future<void> uploadChunk({
    required String id,
    required int offset,
    required Uint8List bytes,
  }) async {
    if (bytes.isEmpty) {
      throw const TusError(
          TusErrorKind.empty, 'chunk must not be empty');
    }
    if (bytes.length > chunkSize) {
      throw TusError(
        TusErrorKind.tooLarge,
        'chunk of ${bytes.length} bytes exceeds the $chunkSize byte limit',
      );
    }
    final end = offset + bytes.length;
    var start = offset;
    var resumeRounds = 0;
    while (start < end) {
      final remaining = bytes.sublist(start - offset);
      int serverOffset;
      try {
        serverOffset = await http.append(
          id: id,
          offset: start,
          bytes: remaining,
        );
      } on TusError catch (error) {
        if (error.kind != TusErrorKind.conflict) rethrow;
        resumeRounds += 1;
        if (resumeRounds > maxResumeRounds) {
          throw TusError(
            TusErrorKind.maxResumeExceeded,
            'upload offset did not converge after $maxResumeRounds '
                'resume rounds; aborting to avoid replaying data at a '
                'stale offset (chunk start $offset, current $start, end $end)',
          );
        }
        final server = await http.offset(id);
        if (server.offset <= start || server.offset < offset) {
          throw TusError(
            TusErrorKind.maxResumeExceeded,
            'cannot resume at server offset ${server.offset}: it does not '
                'advance past this chunk start ($start) — data would be '
                'overwritten; aborting',
          );
        }
        start = server.offset;
        continue;
      }
      // Offset-verify: the server must confirm exactly what we sent.
      if (serverOffset != start + remaining.length) {
        throw TusError(
          TusErrorKind.badRequest,
          'server confirmed offset $serverOffset but expected '
              '${start + remaining.length}',
        );
      }
      return;
    }
  }

  /// Upload [data] in 2 MiB pieces. Declares the session first, then appends
  /// every chunk at its verified offset. Returns the session id.
  Future<String> upload({
    required Uint8List data,
    required int size,
    required String sha256,
    required String contentType,
  }) async {
    if (data.isEmpty) {
      throw const TusError(TusErrorKind.empty, 'upload data must not be empty');
    }
    if (data.length != size) {
      throw TusError(
        TusErrorKind.badRequest,
        'data length ${data.length} does not match declared size $size',
      );
    }
    final id = await createUpload(
      size: size,
      sha256: sha256,
      contentType: contentType,
    );
    var next = 0;
    for (final chunk in splitChunks(data, chunkSize: chunkSize)) {
      await uploadChunk(id: id, offset: next, bytes: chunk);
      next += chunk.length;
    }
    return id;
  }

  /// Query the server-confirmed offset of a session.
  Future<TusOffset> offset(String id) => http.offset(id);

  /// Finalize a session; the server verifies offset == size and SHA-256.
  Future<TusCompletion> complete(String id) => http.complete(id);

  /// Abandon a session server-side.
  Future<void> abort(String id) => http.abort(id);
}