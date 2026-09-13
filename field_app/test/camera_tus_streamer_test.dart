import 'package:crypto/crypto.dart' as crypto;
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:integin_field_app/sync/camera_picker.dart';
import 'package:integin_field_app/sync/camera_tus_streamer.dart';
import 'package:integin_field_app/sync/tus_client.dart';

/// In-memory TUS server fake mirroring the resumable upload contract:
/// create declares size/checksum, append advances a verified offset, complete
/// returns the session key.
class _FakeTusHttp implements TusHttp {
  final Map<String, List<int>> _bytes = {};
  final Map<String, String> _checksums = {};
  int _next = 1;
  String? recordedChecksum;

  List<int> bytes(String id) => List<int>.from(_bytes[id]!);

  @override
  Future<String> create({
    required int size,
    required String sha256,
    required String contentType,
  }) async {
    recordedChecksum = sha256;
    final id = 'up-$_next';
    _next += 1;
    _bytes[id] = <int>[];
    _checksums[id] = sha256;
    return id;
  }

  @override
  Future<int> append({
    required String id,
    required int offset,
    required Uint8List bytes,
  }) async {
    final written = _bytes[id]!;
    if (offset != written.length) {
      throw TusError(
        TusErrorKind.conflict,
        'upload offset mismatch: expected ${written.length}, got $offset',
      );
    }
    written.addAll(bytes);
    return written.length;
  }

  @override
  Future<TusOffset> offset(String id) async => TusOffset(
        id: id,
        size: _bytes[id]!.length,
        offset: _bytes[id]!.length,
        contentType: 'image/jpeg',
        checksum: _checksums[id]!,
      );

  @override
  Future<TusCompletion> complete(String id) async => TusCompletion(
        key: id,
        contentType: 'image/jpeg',
        sha256Hex: _checksums[id]!,
      );

  @override
  Future<void> abort(String id) async {}
}

class _NullPicker implements PhotoPickerService {
  @override
  Future<Uint8List?> pickImage() async => null;
}

void main() {
  test('capture digest is computed before transport and surfaced on completion',
      () async {
    final picker = DeterministicTestPhotoPicker();
    final server = _FakeTusHttp();
    final streamer = CameraTusStreamer(
      picker: picker,
      tus: TusClient(http: server),
    );

    final bytes = await picker.pickImage();
    final expected = crypto.sha256.convert(bytes!).toString();

    final completion = await streamer.captureAndUpload();

    // The digest reached the upload session before any bytes moved: the
    // server fake records the checksum declared on create.
    expect(server.recordedChecksum, expected);
    expect(completion.sha256Hex, expected);
    expect(completion.contentType, 'image/jpeg');
    expect(completion.key, isNotEmpty);
  });

  test('uploaded bytes match captured bytes and reproduce the digest',
      () async {
    final bytes = Uint8List.fromList(jpegHeaderBytes);
    final server = _FakeTusHttp();
    final streamer = CameraTusStreamer(
      picker: DeterministicTestPhotoPicker(bytes: bytes),
      tus: TusClient(http: server),
    );

    final completion = await streamer.captureAndUpload();

    final written = server.bytes(completion.key);
    expect(written, orderedEquals(bytes));
    expect(crypto.sha256.convert(written).toString(), completion.sha256Hex);
  });

  test('capture returning no photo fails closed with StateError', () async {
    final streamer = CameraTusStreamer(
      picker: _NullPicker(),
      tus: TusClient(http: _FakeTusHttp()),
    );
    await expectLater(streamer.captureAndUpload(), throwsStateError);
  });
}