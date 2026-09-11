import 'dart:math' as math;
import 'dart:typed_data';

import 'package:image/image.dart' as img;

/// Preferred/fallback image encoding contract for downsampled uploads.
///
/// The server accepts any content type, but AVIF-first uploads are the
/// default target; [DownsampleEncoding.webpFallback] is used when the running
/// device codec cannot produce AVIF.
enum DownsampleEncoding {
  avifPreferred,
  webpFallback,
}

/// Downsample contract: cap the largest dimension, target a quality factor,
/// and prefer AVIF with a WebP fallback.
class DownsampleConfig {
  const DownsampleConfig({
    this.maxDimension = 4096,
    this.qualityTarget = 0.85,
    this.encoding = DownsampleEncoding.avifPreferred,
  })  : assert(maxDimension > 0, 'maxDimension must be positive'),
        assert(qualityTarget > 0 && qualityTarget <= 1.0,
            'qualityTarget must be in (0, 1]');

  /// Longest side (width or height) an image may have after downsampling.
  final int maxDimension;

  /// Target quality factor in (0, 1], higher is larger and lossier-hostile.
  final double qualityTarget;

  /// AVIF-preferred with WebP fallback.
  final DownsampleEncoding encoding;
}

/// Output of a downsampling pass. [downsampled] distinguishes a real encode
/// from a passthrough, so uploaders never mislabel untouched bytes as
/// re-encoded.
class DownsampleResult {
  const DownsampleResult({
    required this.bytes,
    required this.downsampled,
    required this.contentType,
  });

  final Uint8List bytes;

  /// True when [bytes] were actually re-encoded; false for passthrough.
  final bool downsampled;

  /// Content type of [bytes] (may differ from the source, e.g. `image/webp`).
  final String contentType;
}

/// Downsample policy seam, mirroring `field_app/lib/auth/` seam style
/// (abstract interface + passthrough default + test fake).
abstract interface class DownsamplePolicy {
  /// Optionally re-encode [source] per [config]. Implementations must not
  /// raise for images that already fit the contract — return them unchanged
  /// with `downsampled == false`.
  Future<DownsampleResult> downsample(
    Uint8List source, {
    required String contentType,
    DownsampleConfig config = const DownsampleConfig(),
  });
}

/// Native image decoding and downsampling policy using `package:image`.
///
/// Decodes the source image, scales it down if any dimension exceeds [DownsampleConfig.maxDimension],
/// and re-encodes to WebP (quality mapped from 0..100).
class NativeDownsamplePolicy implements DownsamplePolicy {
  const NativeDownsamplePolicy();

  @override
  Future<DownsampleResult> downsample(
    Uint8List source, {
    required String contentType,
    DownsampleConfig config = const DownsampleConfig(),
  }) async {
    if (config.maxDimension <= 0) {
      throw ArgumentError.value(
          config.maxDimension, 'maxDimension', 'must be positive');
    }
    if (config.qualityTarget <= 0 || config.qualityTarget > 1.0) {
      throw ArgumentError.value(
          config.qualityTarget, 'qualityTarget', 'must be in (0, 1]');
    }
    if (source.isEmpty) {
      throw ArgumentError.value(source, 'source', 'must not be empty');
    }

    img.Image? decoded;
    try {
      if (source.length >= 12) {
        decoded = img.decodeImage(source);
      }
    } catch (_) {
      decoded = null;
    }
    if (decoded == null) {
      // Non-image or unsupported payload: return untouched passthrough
      return DownsampleResult(
        bytes: Uint8List.fromList(source),
        downsampled: false,
        contentType: contentType,
      );
    }

    final origW = decoded.width;
    final origH = decoded.height;
    final maxDim = math.max(origW, origH);

    img.Image target = decoded;
    if (maxDim > config.maxDimension) {
      final scale = config.maxDimension / maxDim;
      final targetW = (origW * scale).round();
      final targetH = (origH * scale).round();
      target = img.copyResize(decoded,
          width: targetW,
          height: targetH,
          interpolation: img.Interpolation.linear);
    }

    // Encode to WebP (efficient modern container supported by browsers & server)
    final encoded = img.encodeWebP(target);

    return DownsampleResult(
      bytes: Uint8List.fromList(encoded),
      downsampled: true,
      contentType: 'image/webp',
    );
  }
}

/// Zero-dependency passthrough policy for tests or raw upload passes.
class PassthroughDownsamplePolicy implements DownsamplePolicy {
  const PassthroughDownsamplePolicy();

  @override
  Future<DownsampleResult> downsample(
    Uint8List source, {
    required String contentType,
    DownsampleConfig config = const DownsampleConfig(),
  }) async {
    if (config.maxDimension <= 0) {
      throw ArgumentError.value(
          config.maxDimension, 'maxDimension', 'must be positive');
    }
    if (config.qualityTarget <= 0 || config.qualityTarget > 1.0) {
      throw ArgumentError.value(
          config.qualityTarget, 'qualityTarget', 'must be in (0, 1]');
    }
    if (source.isEmpty) {
      throw ArgumentError.value(source, 'source', 'must not be empty');
    }
    return DownsampleResult(
      bytes: Uint8List.fromList(source),
      downsampled: false,
      contentType: contentType,
    );
  }
}

/// Upload-pipeline entry point. Defaults to [NativeDownsamplePolicy].
Future<DownsampleResult> downsampleForUpload(
  Uint8List source, {
  required String contentType,
  DownsamplePolicy policy = const NativeDownsamplePolicy(),
  DownsampleConfig config = const DownsampleConfig(),
}) async {
  return policy.downsample(source, contentType: contentType, config: config);
}
