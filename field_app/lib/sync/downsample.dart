import 'dart:typed_data';

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
  }) : assert(maxDimension > 0, 'maxDimension must be positive'),
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

  /// Content type of [bytes] (may differ from the source, e.g. `image/avif`).
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

/// Production note: real encoding needs a native codec. The default policy is
/// deliberately dependency-free — it validates the config, returns the input
/// unchanged, and flags `downsampled = false`, so the upload pipeline stays
/// safe and honest until a native encoder (e.g. `package:image` or a platform
/// plugin via FFI) is wired in behind this seam.
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
      throw ArgumentError.value(config.qualityTarget, 'qualityTarget',
          'must be in (0, 1]');
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

/// Upload-pipeline entry point. Defaults to passthrough; inject a real
/// [DownsamplePolicy] once a native codec is available.
Future<DownsampleResult> downsampleForUpload(
  Uint8List source, {
  required String contentType,
  DownsamplePolicy policy = const PassthroughDownsamplePolicy(),
  DownsampleConfig config = const DownsampleConfig(),
}) async {
  return policy.downsample(source, contentType: contentType, config: config);
}