import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/sync/downsample.dart';

void main() {
  group('looksLikeSupportedImage', () {
    test('accepts JPEG, PNG, WebP and HEIF magic numbers', () {
      expect(
        looksLikeSupportedImage(Uint8List.fromList([0xFF, 0xD8, 0xFF, 0xE0])),
        isTrue,
      );
      expect(
        looksLikeSupportedImage(Uint8List.fromList(
            [0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00])),
        isTrue,
      );
      expect(
        looksLikeSupportedImage(Uint8List.fromList([
          0x52, 0x49, 0x46, 0x46, 0, 0, 0, 0, 0x57, 0x45, 0x42, 0x50,
        ])),
        isTrue,
      );
      expect(
        looksLikeSupportedImage(Uint8List.fromList([
          0x00, 0x00, 0x00, 0x18, // ISO-BMFF box size
          0x66, 0x74, 0x79, 0x70, // 'ftyp'
          0x68, 0x65, 0x69, 0x63, // 'heic'
        ])),
        isTrue,
      );
    });

    test('rejects non-image payloads', () {
      expect(
        looksLikeSupportedImage(Uint8List.fromList([0x25, 0x50, 0x44, 0x46])),
        isFalse,
      );
      expect(
        looksLikeSupportedImage(Uint8List.fromList([0x7F, 0x45, 0x4C, 0x46])),
        isFalse,
      );
      expect(looksLikeSupportedImage(Uint8List(0)), isFalse);
    });

    test('rejects truncated signatures', () {
      expect(looksLikeSupportedImage(Uint8List.fromList([0xFF, 0xD8])), isFalse);
      expect(
        looksLikeSupportedImage(Uint8List.fromList([0x52, 0x49, 0x46, 0x46])),
        isFalse,
      );
    });
  });
}