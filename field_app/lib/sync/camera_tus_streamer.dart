import 'package:crypto/crypto.dart' as crypto;

import 'package:integin_field_app/sync/camera_picker.dart';
import 'package:integin_field_app/sync/tus_client.dart';

/// Connects captured image byte streams directly to the chunked, resumable
/// TusClient upload pipeline. The SHA-256 digest is computed on the captured
/// bytes immediately — before any transport I/O — and declared to the upload
/// session, so the server's complete-time verification compares against the
/// exact bytes observed at capture, never a mutated copy.
class CameraTusStreamer {
  CameraTusStreamer({
    required this.picker,
    required this.tus,
  });

  final PhotoPickerService picker;
  final TusClient tus;

  /// Captures one inspection photo and uploads it. Fails closed when the
  /// camera returns nothing ([StateError]) or the upload pipeline errors.
  ///
  /// Returns the [TusCompletion] with its [TusCompletion.sha256Hex] populated
  /// from the capture-time digest, so downstream persistence of the
  /// completion carries the cryptographic proof of what was uploaded.
  Future<TusCompletion> captureAndUpload() async {
    final bytes = await picker.pickImage();
    if (bytes == null || bytes.isEmpty) {
      throw StateError('camera capture returned no image bytes');
    }
    final digest = crypto.sha256.convert(bytes).toString();
    final id = await tus.upload(
      data: bytes,
      size: bytes.length,
      sha256: digest,
      contentType: 'image/jpeg',
    );
    final completed = await tus.complete(id);
    return TusCompletion(
      key: completed.key,
      contentType: completed.contentType,
      sha256Hex: digest,
    );
  }
}