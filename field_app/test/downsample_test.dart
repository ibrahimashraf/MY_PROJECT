import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';

import 'package:integin_field_app/sync/downsample.dart';

class _CapturingPolicy implements DownsamplePolicy {
  String? lastContentType;
  DownsampleConfig? lastConfig;

  @override
  Future<DownsampleResult> downsample(
    Uint8List source, {
    required String contentType,
    DownsampleConfig config = const DownsampleConfig(),
  }) async {
    lastContentType = contentType;
    lastConfig = config;
    return DownsampleResult(
      bytes: Uint8List.fromList([9, 9, 9]),
      downsampled: true,
      contentType: 'image/avif',
    );
  }
}

void main() {
  group('DownsampleConfig contract', () {
    test('defaults to 4096 max dimension, 0.85 quality, AVIF-preferred',
        () {
      const config = DownsampleConfig();
      expect(config.maxDimension, 4096);
      expect(config.qualityTarget, 0.85);
      expect(config.encoding, DownsampleEncoding.avifPreferred);
    });

    test('rejects an invalid contract at construction', () {
      expect(
        () => DownsampleConfig(maxDimension: 0),
        throwsA(isA<AssertionError>()),
      );
      expect(
        () => DownsampleConfig(qualityTarget: 0),
        throwsA(isA<AssertionError>()),
      );
      expect(
        () => DownsampleConfig(qualityTarget: 1.5),
        throwsA(isA<AssertionError>()),
      );
    });
  });

  group('PassthroughDownsamplePolicy', () {
    test('returns the input unchanged, flags downsampled=false, keeps type',
        () async {
      const policy = PassthroughDownsamplePolicy();
      final source = Uint8List.fromList([1, 2, 3, 4]);
      final result = await policy.downsample(
        source,
        contentType: 'image/jpeg',
        config: const DownsampleConfig(
          maxDimension: 1024,
          qualityTarget: 0.5,
          encoding: DownsampleEncoding.webpFallback,
        ),
      );
      expect(result.downsampled, isFalse);
      expect(result.contentType, 'image/jpeg');
      expect(result.bytes, source);
    });

    test('copies defensively: mutating the result never touches the source',
        () async {
      const policy = PassthroughDownsamplePolicy();
      final source = Uint8List.fromList([1, 2, 3, 4]);
      final result = await policy.downsample(
        source,
        contentType: 'image/png',
      );
      result.bytes[0] = 99;
      expect(source[0], 1);
    });

    test('accepts the default config', () async {
      const policy = PassthroughDownsamplePolicy();
      final result = await policy.downsample(
        Uint8List.fromList([1]),
        contentType: 'image/png',
      );
      expect(result.downsampled, isFalse);
    });

    test('rejects empty input fail-closed', () async {
      const policy = PassthroughDownsamplePolicy();
      await expectLater(
        policy.downsample(Uint8List(0), contentType: 'image/png'),
        throwsA(isA<ArgumentError>()),
      );
    });
  });

  group('downsampleForUpload', () {
    test('defaults to the passthrough policy', () async {
      final source = Uint8List.fromList([1, 2, 3]);
      final result =
          await downsampleForUpload(source, contentType: 'image/jpeg');
      expect(result.downsampled, isFalse);
      expect(result.bytes, source);
    });

    test('honors an injected policy and forwards the contract', () async {
      final fake = _CapturingPolicy();
      final result = await downsampleForUpload(
        Uint8List.fromList([1, 2, 3]),
        contentType: 'image/jpeg',
        policy: fake,
        config: const DownsampleConfig(
          maxDimension: 800,
          qualityTarget: 0.6,
          encoding: DownsampleEncoding.webpFallback,
        ),
      );
      expect(result.downsampled, isTrue);
      expect(result.contentType, 'image/avif');
      expect(result.bytes, Uint8List.fromList([9, 9, 9]));
      expect(fake.lastContentType, 'image/jpeg');
      expect(fake.lastConfig!.maxDimension, 800);
      expect(fake.lastConfig!.qualityTarget, 0.6);
      expect(fake.lastConfig!.encoding, DownsampleEncoding.webpFallback);
    });
  });
}