import 'dart:convert';
import 'dart:typed_data';
import 'package:crypto/crypto.dart' as crypto;

const String tagSubmissionSeal = "INTEGIN-SEAL-v1\x00";

/// Strict field validation regex — prevents null-byte injection,
/// Unicode control characters, and other delimiter-injection vectors.
/// Allow only alphanumeric, hyphen, underscore, dot, colon, slash.
final RegExp _validFieldChar = RegExp(r'^[a-zA-Z0-9_\-\.:\/]+$');

/// Canonical SHA-256 digest of a media artifact embedded in the
/// submission seal. Maps artifact type + ID to its digest, so
/// the composite seal binds the visual evidence (photos, readings,
/// calibration certificates) alongside the metadata JSON.
class MediaArtifactDigest {
  final String artifactId;
  final String artifactType;
  final String sha256Digest; // hex-encoded 64-char string

  MediaArtifactDigest({
    required this.artifactId,
    required this.artifactType,
    required this.sha256Digest,
  })  : assert(_validFieldChar.hasMatch(artifactId),
            'artifactId must match $_validFieldChar'),
        assert(_validFieldChar.hasMatch(artifactType),
            'artifactType must match $_validFieldChar'),
        assert(
            RegExp(r'^[a-f0-9]{64}$').hasMatch(sha256Digest),
            'sha256Digest must be 64-char lowercase hex');
}

class SubmissionSealPayload {
  final String deviceKeyDID;
  final String tokenID;
  final int leaseEpoch;
  final Uint8List dataPayload;
  final Uint8List appEd25519Sig;
  final List<MediaArtifactDigest> mediaDigests;

  const SubmissionSealPayload({
    required this.deviceKeyDID,
    required this.tokenID,
    required this.leaseEpoch,
    required this.dataPayload,
    required this.appEd25519Sig,
    this.mediaDigests = const [],
  });

  Uint8List deriveCompositeDigest() {
    final tagBytes = utf8.encode(tagSubmissionSeal);

    final allBytes = <int>[];
    allBytes.addAll(tagBytes);

    // Length-prefix DeviceKeyDID and TokenID to prevent
    // boundary-shifting collisions (e.g. "tablet-1"+"002"
    // vs "tablet-10"+"02" producing the same byte stream).
    _appendLengthPrefixed(allBytes, deviceKeyDID);
    _appendLengthPrefixed(allBytes, tokenID);

    final epochBytes = ByteData(8)..setUint64(0, leaseEpoch, Endian.big);
    allBytes.addAll(epochBytes.buffer.asUint8List());

    // Canonical media artifact digests: sort lexicographically by
    // artifactID to guarantee deterministic ordering across threads.
    // Collection count prefix (uint16 BE) before the elements,
    // then each element is length-prefixed strings + raw 32 bytes.
    // SHA-256 digests are decoded from hex to raw 32 bytes before
    // packing. The Go mirror in pkg/domain/fieldtrust.go must
    // decode identically: [32]byte raw, not 64-char ASCII.
    final sortedDigests = List<MediaArtifactDigest>.from(mediaDigests)
      ..sort((a, b) => a.artifactId.compareTo(b.artifactId));

    final countData = ByteData(2)..setUint16(0, sortedDigests.length, Endian.big);
    allBytes.addAll(countData.buffer.asUint8List());

    for (final artifact in sortedDigests) {
      _appendLengthPrefixed(allBytes, artifact.artifactId);
      _appendLengthPrefixed(allBytes, artifact.artifactType);
      // Decode hex → raw 32 bytes for cryptographic packing
      final rawDigest = _hexDecode(artifact.sha256Digest);
      _appendRaw(allBytes, rawDigest);
    }

    allBytes.addAll(dataPayload);
    allBytes.addAll(appEd25519Sig);

    return Uint8List.fromList(crypto.sha256.convert(allBytes).bytes);
  }

  /// Appends a big-endian uint16 length prefix followed by the
  /// UTF-8 bytes of [s]. Maximum string length is 65535 bytes.
  static void _appendLengthPrefixed(List<int> buf, String s) {
    final bytes = utf8.encode(s);
    assert(bytes.length <= 0xFFFF, 'string exceeds uint16 max length');
    buf.add((bytes.length >> 8) & 0xFF);
    buf.add(bytes.length & 0xFF);
    buf.addAll(bytes);
  }

  /// Appends raw bytes without a length prefix (for fixed-size
  /// fields like decoded SHA-256 digests).
  static void _appendRaw(List<int> buf, Uint8List bytes) {
    buf.addAll(bytes);
  }

  /// Decodes a lowercase hex string to raw bytes.
  static Uint8List _hexDecode(String hex) {
    assert(hex.length % 2 == 0, 'hex string must have even length');
    final result = Uint8List(hex.length ~/ 2);
    for (var i = 0; i < hex.length; i += 2) {
      result[i ~/ 2] =
          int.parse(hex.substring(i, i + 2), radix: 16);
    }
    return result;
  }
}
