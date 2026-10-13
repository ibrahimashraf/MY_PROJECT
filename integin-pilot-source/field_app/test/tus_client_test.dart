import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';

import 'package:integin_field_app/sync/tus_client.dart';

final _sha = 'a' * 64;

class _Session {
  _Session({
    required this.size,
    required this.checksum,
    required this.contentType,
  });

  final int size;
  final String checksum;
  final String contentType;
  final List<int> data = [];
  int offset = 0;
}

/// Scriptable in-memory fake of the server contract in
/// `internal/storage/tus_route.go` + `tus_handler.go`.
class _FakeServer implements TusHttp {
  final Map<String, _Session> _sessions = {};
  final List<(int, int)> appendLog = [];

  /// When true, the next append drops like a mid-stream link failure.
  bool dropNextAppend = false;

  int offsetFor(String id) => _sessions[id]!.offset;

  /// Seeds a session with bytes already committed (as if a previous run
  /// left the server mid-upload).
  void seed(String id, List<int> bytes) {
    _sessions[id]!.data.addAll(bytes);
    _sessions[id]!.offset += bytes.length;
  }

  Uint8List written(String id) =>
      Uint8List.fromList(_sessions[id]!.data.toList());

  @override
  Future<String> create({
    required int size,
    required String sha256,
    required String contentType,
  }) async {
    final id = 'up-${_sessions.length + 1}';
    _sessions[id] = _Session(
      size: size,
      checksum: sha256,
      contentType: contentType,
    );
    return id;
  }

  @override
  Future<int> append({
    required String id,
    required int offset,
    required Uint8List bytes,
  }) async {
    final session = _sessions[id];
    if (session == null) {
      throw const TusError(TusErrorKind.notFound, 'upload session not found');
    }
    if (dropNextAppend) {
      dropNextAppend = false;
      throw const TusError(TusErrorKind.network, 'link dropped mid-stream');
    }
    if (offset != session.offset) {
      throw TusError(
        TusErrorKind.conflict,
        'upload offset mismatch: expected ${session.offset}, got $offset',
      );
    }
    if (offset + bytes.length > session.size) {
      throw const TusError(
          TusErrorKind.badRequest, 'chunk exceeds declared upload size');
    }
    appendLog.add((offset, bytes.length));
    session.data.addAll(bytes);
    session.offset += bytes.length;
    return session.offset;
  }

  @override
  Future<TusOffset> offset(String id) async {
    final session = _sessions[id];
    if (session == null) {
      throw const TusError(TusErrorKind.notFound, 'upload session not found');
    }
    return TusOffset(
      id: id,
      size: session.size,
      offset: session.offset,
      contentType: session.contentType,
      checksum: session.checksum,
    );
  }

  @override
  Future<TusCompletion> complete(String id) async {
    final session = _sessions[id];
    if (session == null) {
      throw const TusError(TusErrorKind.notFound, 'upload session not found');
    }
    if (session.offset != session.size) {
      throw TusError(
        TusErrorKind.badRequest,
        'upload incomplete: ${session.offset} of ${session.size} bytes received',
      );
    }
    _sessions.remove(id);
    return TusCompletion(
      key: id,
      contentType: session.contentType,
    );
  }

  @override
  Future<void> abort(String id) async {
    final existed = _sessions.remove(id) != null;
    if (!existed) {
      throw const TusError(TusErrorKind.notFound, 'upload session not found');
    }
  }
}

/// Always conflicts and never advances the confirmed offset.
class _StaticServer implements TusHttp {
  @override
  Future<int> append({
    required String id,
    required int offset,
    required Uint8List bytes,
  }) async {
    throw const TusError(TusErrorKind.conflict, 'upload offset mismatch');
  }

  @override
  Future<TusOffset> offset(String id) async => TusOffset(
        id: id,
        size: 100,
        offset: 0,
        contentType: 'application/octet-stream',
        checksum: _sha,
      );

  @override
  Future<String> create({
    required int size,
    required String sha256,
    required String contentType,
  }) async =>
      'up-static';

  @override
  Future<TusCompletion> complete(String id) async =>
      throw UnimplementedError();

  @override
  Future<void> abort(String id) async => throw UnimplementedError();
}

/// Always conflicts, but a phantom writer advances the confirmed offset a
/// little on every re-query — progress that never converges.
class _SlowAdvanceServer implements TusHttp {
  int _offset = 0;

  @override
  Future<int> append({
    required String id,
    required int offset,
    required Uint8List bytes,
  }) async {
    throw const TusError(TusErrorKind.conflict, 'upload offset mismatch');
  }

  @override
  Future<TusOffset> offset(String id) async {
    _offset += 25;
    if (_offset > 100) _offset = 100;
    return TusOffset(
      id: id,
      size: 100,
      offset: _offset,
      contentType: 'application/octet-stream',
      checksum: _sha,
    );
  }

  @override
  Future<String> create({
    required int size,
    required String sha256,
    required String contentType,
  }) async =>
      'up-slow';

  @override
  Future<TusCompletion> complete(String id) async =>
      throw UnimplementedError();

  @override
  Future<void> abort(String id) async => throw UnimplementedError();
}

/// Accepts an append but confirms the wrong offset (contract violation).
class _MismatchedAppendServer implements TusHttp {
  @override
  Future<int> append({
    required String id,
    required int offset,
    required Uint8List bytes,
  }) async =>
      offset + bytes.length + 1;

  @override
  Future<TusOffset> offset(String id) async =>
      throw UnimplementedError();

  @override
  Future<String> create({
    required int size,
    required String sha256,
    required String contentType,
  }) async =>
      'up-mismatch';

  @override
  Future<TusCompletion> complete(String id) async =>
      throw UnimplementedError();

  @override
  Future<void> abort(String id) async => throw UnimplementedError();
}

void main() {
  group('TusClient round trip', () {
    test('splits into 2 MiB pieces, uploads all, completes', () async {
      final data = Uint8List.fromList(
          List<int>.generate(2 * tusChunkSize + 100, (i) => i % 251));
      final server = _FakeServer();
      final client = TusClient(http: server);

      final id = await client.createUpload(
        size: data.length,
        sha256: _sha,
        contentType: 'image/png',
      );

      final chunks = splitChunks(data);
      expect(chunks, hasLength(3));
      expect(chunks[0].length, tusChunkSize);
      expect(chunks[1].length, tusChunkSize);
      expect(chunks[2].length, 100);
      expect(chunks.fold<int>(0, (sum, c) => sum + c.length), data.length);
      expect([...chunks[0], ...chunks[1], ...chunks[2]], data);

      var next = 0;
      for (final chunk in chunks) {
        await client.uploadChunk(id: id, offset: next, bytes: chunk);
        next += chunk.length;
      }

      expect(server.offsetFor(id), data.length);
      final written = server.written(id);
      final completion = await client.complete(id);
      expect(completion.key, id);
      expect(completion.contentType, 'image/png');
      expect(written, data);
    });

    test('mid-upload link drop is surfaced and retries resume via same id',
        () async {
      final data = Uint8List.fromList(
          List<int>.generate(2 * tusChunkSize + 100, (i) => i % 251));
      final server = _FakeServer();
      final client = TusClient(http: server);

      final id = await client.createUpload(
        size: data.length,
        sha256: _sha,
        contentType: 'image/png',
      );
      final chunks = splitChunks(data);

      await client.uploadChunk(id: id, offset: 0, bytes: chunks[0]);
      expect(server.offsetFor(id), tusChunkSize);

      // The second append drops mid-stream: the error is surfaced, and no
      // bytes reach the server.
      server.dropNextAppend = true;
      await expectLater(
        client.uploadChunk(id: id, offset: tusChunkSize, bytes: chunks[1]),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.network)
            .having((e) => e.message, 'message', contains('dropped'))),
      );
      expect(server.offsetFor(id), tusChunkSize);
      expect(server.appendLog, isNot(contains((tusChunkSize, tusChunkSize))));

      // Same session id resumes from the server-confirmed offset.
      await client.uploadChunk(
          id: id, offset: server.offsetFor(id), bytes: chunks[1]);
      await client.uploadChunk(
          id: id, offset: server.offsetFor(id), bytes: chunks[2]);
      expect(server.offsetFor(id), data.length);

      final written = server.written(id);
      final completion = await client.complete(id);
      expect(completion.key, id);
      expect(written, data);
    });

    test('complete fails closed when the session was never fully uploaded',
        () async {
      final server = _FakeServer();
      final client = TusClient(http: server);
      final id = await client.createUpload(
        size: 100,
        sha256: _sha,
        contentType: 'image/png',
      );
      await client.uploadChunk(id: id, offset: 0, bytes: Uint8List(50));
      await expectLater(
        client.complete(id),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)
            .having((e) => e.message, 'message', contains('incomplete'))),
      );
    });
  });

  group('TusClient resume semantics', () {
    test('409 conflict re-queries the offset and resumes in place', () async {
      final server = _FakeServer();
      final chunk =
          Uint8List.fromList(List<int>.generate(tusChunkSize, (i) => i % 251));
      final id = await server.create(
        size: tusChunkSize,
        sha256: _sha,
        contentType: 'image/png',
      );
      // The first MiB was already committed server-side by a prior run.
      server.seed(id, chunk.sublist(0, 1024 * 1024));

      final client = TusClient(http: server);
      await client.uploadChunk(id: id, offset: 0, bytes: chunk);

      expect(server.offsetFor(id), tusChunkSize);
      // Exactly one conflict-triggered resume: first append rejected (409),
      // second append only the remaining MiB at the re-queried offset.
      expect(server.appendLog.length, 1);
      expect(server.appendLog[0], (1024 * 1024, 1024 * 1024));
      expect(server.written(id), chunk);
    });

    test('server already past the chunk end is a no-op, not a replay',
        () async {
      final server = _FakeServer();
      final client = TusClient(http: server);
      final id = await client.createUpload(
        size: 100,
        sha256: _sha,
        contentType: 'application/octet-stream',
      );
      await client.uploadChunk(id: id, offset: 0, bytes: Uint8List(100));
      expect(server.appendLog, hasLength(1));

      // Caller retries from offset 0, but the server is at 100: the whole
      // chunk is already committed, so the resume terminates as a no-op
      // without replaying any bytes.
      await client.uploadChunk(id: id, offset: 0, bytes: Uint8List(100));
      expect(server.appendLog, hasLength(1));
      expect(server.offsetFor(id), 100);
    });

    test('never blind-retries a stale offset', () async {
      final client = TusClient(http: _StaticServer());
      await expectLater(
        client.uploadChunk(
            id: 'up-static', offset: 0, bytes: Uint8List(100)),
        throwsA(isA<TusError>()
            .having(
                (e) => e.kind, 'kind', TusErrorKind.maxResumeExceeded)),
      );
    });

    test('max resume rounds are enforced even when offset keeps advancing',
        () async {
      final client = TusClient(http: _SlowAdvanceServer());
      await expectLater(
        client.uploadChunk(
            id: 'up-slow', offset: 0, bytes: Uint8List(100)),
        throwsA(isA<TusError>()
            .having(
                (e) => e.kind, 'kind', TusErrorKind.maxResumeExceeded)),
      );
    });

    test('a server-confirmed offset that skips ahead surfaces as an error',
        () async {
      final client = TusClient(http: _MismatchedAppendServer());
      await expectLater(
        client.uploadChunk(
            id: 'up-mismatch', offset: 0, bytes: Uint8List(10)),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)
            .having((e) => e.message, 'message', contains('confirmed offset'))),
      );
    });
  });

  group('TusClient abort', () {
    test('abort removes the session server-side', () async {
      final server = _FakeServer();
      final client = TusClient(http: server);
      final id = await client.createUpload(
        size: 100,
        sha256: _sha,
        contentType: 'application/octet-stream',
      );
      await client.abort(id);
      await expectLater(
        client.offset(id),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.notFound)),
      );
    });
  });

  group('TusClient fail-closed validation', () {
    test('create rejects empty size, non-hex checksum and blank content type',
        () async {
      final client = TusClient(http: _FakeServer());
      await expectLater(
        client.createUpload(size: 0, sha256: _sha, contentType: 'image/png'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.empty)),
      );
      await expectLater(
        client.createUpload(size: 10, sha256: 'no', contentType: 'image/png'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)),
      );
      await expectLater(
        client.createUpload(size: 10, sha256: _sha, contentType: '   '),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)),
      );
    });

    test('upload rejects empty data and size mismatch before any request',
        () async {
      final server = _FakeServer();
      final client = TusClient(http: server);
      await expectLater(
        client.upload(
          data: Uint8List(0),
          size: 0,
          sha256: _sha,
          contentType: 'image/png',
        ),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.empty)),
      );
      await expectLater(
        client.upload(
          data: Uint8List(10),
          size: 11,
          sha256: _sha,
          contentType: 'image/png',
        ),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)),
      );
      expect(server._sessions, isEmpty);
    });

    test('uploadChunk rejects empty and oversized chunks', () async {
      final client = TusClient(http: _FakeServer());
      await expectLater(
        client.uploadChunk(
            id: 'up-1', offset: 0, bytes: Uint8List(0)),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.empty)),
      );
      await expectLater(
        client.uploadChunk(
            id: 'up-1',
            offset: 0,
            bytes: Uint8List(tusChunkSize + 1)),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.tooLarge)),
      );
    });
  });

  group('HttpTusHttp contract mapping', () {
    late Uri baseUrl;
    setUp(() {
      baseUrl = Uri.parse('https://INTEGIN.test/uploads');
    });

    test('create posts size/checksum/content_type and maps 201', () async {
      late Map<String, dynamic> sent;
      final transport = HttpTusHttp(
        baseUrl: baseUrl,
        authToken: 'tok-1',
        client: MockClient((request) async {
          expect(request.method, 'POST');
          expect(request.url.path, '/uploads');
          expect(request.headers['authorization'], 'Bearer tok-1');
          expect(request.headers['content-type'], 'application/json');
          sent = jsonDecode(request.body) as Map<String, dynamic>;
          return http.Response('{"id":"up-42"}', 201);
        }),
      );
      final id = await transport.create(
        size: 10,
        sha256: _sha,
        contentType: 'image/png',
      );
      expect(id, 'up-42');
      expect(sent, {'size': 10, 'checksum': _sha, 'content_type': 'image/png'});
    });

    test('create surfaces 400 and 401 as typed errors', () async {
      final bad = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient(
            (_) async => http.Response('{"error":"checksum is invalid"}', 400)),
      );
      await expectLater(
        bad.create(size: 1, sha256: _sha, contentType: 'image/png'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)
            .having((e) => e.message, 'message', 'checksum is invalid')
            .having((e) => e.statusCode, 'statusCode', 400)),
      );

      final denied = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((_) async =>
            http.Response('{"error":"authentication_failed"}', 401)),
      );
      await expectLater(
        denied.create(size: 1, sha256: _sha, contentType: 'image/png'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.authentication)),
      );
    });

    test('create surfaces a missing id as badRequest', () async {
      final transport = HttpTusHttp(
        baseUrl: baseUrl,
        client:
            MockClient((_) async => http.Response('{}', 201)),
      );
      await expectLater(
        transport.create(size: 1, sha256: _sha, contentType: 'image/png'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)),
      );
    });

    test('append base64-encodes the chunk and maps 200/409/404', () async {
      final bytes = Uint8List.fromList([1, 2, 3, 255]);
      late Map<String, dynamic> sent;
      final ok = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((request) async {
          expect(request.url.path, '/uploads/up-1/chunks');
          sent = jsonDecode(request.body) as Map<String, dynamic>;
          return http.Response('{"offset":4}', 200);
        }),
      );
      expect(
          await ok.append(id: 'up-1', offset: 0, bytes: bytes), 4);
      expect(sent['offset'], 0);
      expect(base64Decode(sent['data'] as String), bytes);

      final conflict = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((_) async =>
            http.Response('{"error":"upload offset mismatch"}', 409)),
      );
      await expectLater(
        conflict.append(id: 'up-1', offset: 0, bytes: bytes),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.conflict)
            .having((e) => e.statusCode, 'statusCode', 409)),
      );

      final missing = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient(
            (_) async => http.Response('{"error":"not found"}', 404)),
      );
      await expectLater(
        missing.append(id: 'up-nope', offset: 0, bytes: bytes),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.notFound)),
      );
    });

    test('append surfaces a non-integer confirmed offset as badRequest',
        () async {
      final transport = HttpTusHttp(
        baseUrl: baseUrl,
        client:
            MockClient((_) async => http.Response('{"offset":"x"}', 200)),
      );
      await expectLater(
        transport.append(id: 'up-1', offset: 0, bytes: Uint8List(1)),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)),
      );
    });

    test('offset maps 200 session and 404', () async {
      final ok = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((request) async {
          expect(request.method, 'GET');
          expect(request.url.path, '/uploads/up-1/offset');
          return http.Response(
            '{"id":"up-1","size":10,"offset":5,'
            '"content_type":"image/png","checksum":"$_sha"}',
            200,
          );
        }),
      );
      final info = await ok.offset('up-1');
      expect(info.id, 'up-1');
      expect(info.size, 10);
      expect(info.offset, 5);
      expect(info.contentType, 'image/png');
      expect(info.checksum, _sha);

      final missing = HttpTusHttp(
        baseUrl: baseUrl,
        client:
            MockClient((_) async => http.Response('{"error":"not found"}', 404)),
      );
      await expectLater(
        missing.offset('up-nope'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.notFound)),
      );
    });

    test('complete maps 200 key/content_type and 400 incomplete', () async {
      final ok = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((request) async {
          expect(request.method, 'POST');
          expect(request.url.path, '/uploads/up-1/complete');
          return http.Response(
              '{"key":"up-1","content_type":"image/png"}', 200);
        }),
      );
      final done = await ok.complete('up-1');
      expect(done.key, 'up-1');
      expect(done.contentType, 'image/png');

      final incomplete = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((_) async => http
            .Response('{"error":"upload incomplete: 0 of 10 bytes received"}', 400)),
      );
      await expectLater(
        incomplete.complete('up-1'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.badRequest)
            .having((e) => e.message, 'message', contains('incomplete'))),
      );
    });

    test('abort maps 204 and 404', () async {
      final ok = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((request) async {
          expect(request.method, 'POST');
          expect(request.url.path, '/uploads/up-1/abort');
          return http.Response('', 204);
        }),
      );
      await ok.abort('up-1');

      final missing = HttpTusHttp(
        baseUrl: baseUrl,
        client:
            MockClient((_) async => http.Response('{"error":"not found"}', 404)),
      );
      await expectLater(
        missing.abort('up-nope'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.notFound)),
      );
    });

    test('transport failures and 5xx map to network', () async {
      final offline = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient((_) async => throw http.ClientException('offline')),
      );
      await expectLater(
        offline.create(size: 1, sha256: _sha, contentType: 'image/png'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.network)),
      );

      final down = HttpTusHttp(
        baseUrl: baseUrl,
        client: MockClient(
            (_) async => http.Response('{"error":"nope"}', 503)),
      );
      await expectLater(
        down.create(size: 1, sha256: _sha, contentType: 'image/png'),
        throwsA(isA<TusError>()
            .having((e) => e.kind, 'kind', TusErrorKind.network)),
      );
    });
  });
}