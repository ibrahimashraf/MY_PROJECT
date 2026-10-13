import 'package:flutter/services.dart';

/// A captured inspection photo, together with transport metadata.
class PhotoPickerResult {
  const PhotoPickerResult({
    required this.bytes,
    required this.contentType,
    required this.capturedAt,
    required this.metadata,
  });

  final Uint8List bytes;
  final String contentType;
  final DateTime capturedAt;
  final Map<String, dynamic> metadata;
}

abstract interface class PhotoPickerService {
  Future<Uint8List?> pickImage();
}

/// Captures a photo through the native Android/iOS host over the
/// `INTEGIN.field_app/camera` method channel.
class PlatformChannelPhotoPicker implements PhotoPickerService {
  static const MethodChannel _channel = MethodChannel('INTEGIN.field_app/camera');

  @override
  Future<Uint8List?> pickImage() async {
    return _channel.invokeMethod<Uint8List>('capturePhoto');
  }
}

/// Deterministic JPEG stub used by CI and widget tests in place of the
/// native camera. Returns valid, non-empty JPEG header bytes.
class DeterministicTestPhotoPicker implements PhotoPickerService {
  DeterministicTestPhotoPicker({Uint8List? bytes})
      : result = PhotoPickerResult(
          bytes: bytes ?? Uint8List.fromList(jpegHeaderBytes),
          contentType: 'image/jpeg',
          capturedAt: DateTime.fromMillisecondsSinceEpoch(
              kDeterministicCaptureEpochMs),
          metadata: const {'source': 'deterministic_test', 'width': 8, 'height': 8},
        );

  final PhotoPickerResult result;

  @override
  Future<Uint8List?> pickImage() async => result.bytes;
}

const int kDeterministicCaptureEpochMs = 1700000000000;

/// Minimal JPEG with a JFIF APP0 header plus SOI and EOI markers.
final List<int> jpegHeaderBytes = <int>[
  0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, // SOI + APP0
  ...'JFIF'.codeUnits, 0x00, // JFIF identifier
  0x01, 0x01, 0x01, 0x00, 0x48, 0x00, 0x48, 0x00, // version 1.01, density
  0xFF, 0xD9, // EOI
];