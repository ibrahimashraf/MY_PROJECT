import 'dart:async';
import 'dart:convert';
import 'dart:math';

import 'package:http/http.dart' as http;

import '../security/transaction_signer.dart';
import 'package_manifest_client.dart';

typedef ManifestRequestIdGenerator = String Function();

/// Public, non-secret categories suitable for Field status messaging.
enum PackageManifestTransportFailure {
  configuration,
  unauthorized,
  replayed,
  assignmentUnavailable,
  unavailable,
  rejected,
  invalidResponse,
}

/// A transport failure that intentionally omits server internals and proof data.
class PackageManifestTransportException implements Exception {
  const PackageManifestTransportException(this.failure, this.message);

  final PackageManifestTransportFailure failure;
  final String message;

  @override
  String toString() => 'PackageManifestTransportException: $message';
}

/// The immutable, signed proof payload sent to the future manifest endpoint.
class ManifestReadProof {
  const ManifestReadProof({
    required this.requestId,
    required this.deviceId,
    required this.authorityId,
    required this.authorityEpoch,
    required this.inspectionId,
    required this.issuedAt,
    required this.expiresAt,
    required this.keyId,
    required this.signature,
  });

  final String requestId;
  final String deviceId;
  final String authorityId;
  final int authorityEpoch;
  final String inspectionId;
  final DateTime issuedAt;
  final DateTime expiresAt;
  final String keyId;
  final String signature;

  Map<String, dynamic> toJson() => <String, dynamic>{
        'purpose': manifestReadProofPurpose,
        'request_id': requestId,
        'device_id': deviceId,
        'authority_id': authorityId,
        'authority_epoch': authorityEpoch,
        'inspection_id': inspectionId,
        'issued_at': canonicalTimestamp(issuedAt),
        'expires_at': canonicalTimestamp(expiresAt),
        'signature_algorithm': manifestReadProofSignatureAlgorithm,
        'key_id': keyId,
        'signature': signature,
      };
}

/// Fetches one server-signed package manifest using a device-bound proof.
///
/// The transport never verifies, binds, or caches a response. Callers must pass
/// a result through [PackageManifestVerifier] before local persistence. It has
/// no default endpoint, so current unmounted server composition cannot receive
/// a request unless a future pilot caller explicitly provides one.
class PackageManifestTransport {
  PackageManifestTransport({
    http.Client? client,
    ManifestRequestIdGenerator? requestIdGenerator,
    this.proofLifetime = const Duration(minutes: 5),
    this.requestTimeout = const Duration(seconds: 15),
  })  : _client = client ?? http.Client(),
        _requestIdGenerator = requestIdGenerator ?? _secureRequestId {
    if (proofLifetime <= Duration.zero) {
      throw ArgumentError.value(
        proofLifetime,
        'proofLifetime',
        'must be positive',
      );
    }
    if (requestTimeout <= Duration.zero) {
      throw ArgumentError.value(
        requestTimeout,
        'requestTimeout',
        'must be positive',
      );
    }
  }

  final http.Client _client;
  final ManifestRequestIdGenerator _requestIdGenerator;
  final Duration proofLifetime;
  final Duration requestTimeout;

  /// Requests one manifest for a device, authority, and inspection binding.
  ///
  /// [now] is supplied by the caller for deterministic protocol testing. The
  /// server remains authoritative for proof acceptance and lifetime limits.
  Future<SignedPackageManifest> fetch({
    required Uri endpoint,
    required DeviceSigner signer,
    required String deviceId,
    required String authorityId,
    required int authorityEpoch,
    required String inspectionId,
    required DateTime now,
  }) async {
    _validateRequest(
      endpoint: endpoint,
      signer: signer,
      deviceId: deviceId,
      authorityId: authorityId,
      authorityEpoch: authorityEpoch,
      inspectionId: inspectionId,
    );
    final requestId = _requestIdGenerator();
    if (requestId.trim().isEmpty) {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.configuration,
        'manifest request identifier is not configured',
      );
    }
    final keyId = signer.keyId!;
    final issuedAt = now.toUtc();
    final expiresAt = issuedAt.add(proofLifetime);
    final signature = await signer.signManifestProof(
      requestId: requestId,
      deviceId: deviceId,
      authorityId: authorityId,
      authorityEpoch: authorityEpoch,
      inspectionId: inspectionId,
      issuedAt: issuedAt,
      expiresAt: expiresAt,
      keyId: keyId,
    );
    final proof = ManifestReadProof(
      requestId: requestId,
      deviceId: deviceId,
      authorityId: authorityId,
      authorityEpoch: authorityEpoch,
      inspectionId: inspectionId,
      issuedAt: issuedAt,
      expiresAt: expiresAt,
      keyId: keyId,
      signature: signature,
    );
    late http.Response response;
    try {
      response = await _client
          .post(
            endpoint,
            headers: const <String, String>{
              'accept': 'application/json',
              'content-type': 'application/json',
            },
            body: jsonEncode(<String, dynamic>{'proof': proof.toJson()}),
          )
          .timeout(requestTimeout);
    } on TimeoutException {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.unavailable,
        'package manifest request timed out',
      );
    } on http.ClientException {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.unavailable,
        'package manifest service is unavailable',
      );
    }
    if (response.statusCode != 200) {
      throw PackageManifestTransportException(
        _failureForStatus(response.statusCode),
        'package manifest request was not accepted',
      );
    }
    try {
      final decoded = jsonDecode(response.body);
      if (decoded is! Map) {
        throw const FormatException('manifest response is not an object');
      }
      return SignedPackageManifest(Map<String, dynamic>.from(decoded));
    } on FormatException {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.invalidResponse,
        'package manifest response is invalid',
      );
    } on TypeError {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.invalidResponse,
        'package manifest response is invalid',
      );
    }
  }

  static void _validateRequest({
    required Uri endpoint,
    required DeviceSigner signer,
    required String deviceId,
    required String authorityId,
    required int authorityEpoch,
    required String inspectionId,
  }) {
    if (!endpoint.isAbsolute ||
        (endpoint.scheme != 'https' && endpoint.scheme != 'http')) {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.configuration,
        'manifest endpoint is invalid',
      );
    }
    if (signer.keyId == null || signer.keyId!.trim().isEmpty) {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.configuration,
        'device signing key identifier is not configured',
      );
    }
    if (deviceId.trim().isEmpty ||
        authorityId.trim().isEmpty ||
        inspectionId.trim().isEmpty ||
        authorityEpoch <= 0) {
      throw const PackageManifestTransportException(
        PackageManifestTransportFailure.configuration,
        'manifest request binding is incomplete',
      );
    }
  }

  static PackageManifestTransportFailure _failureForStatus(int statusCode) {
    if (statusCode == 401 || statusCode == 403) {
      return PackageManifestTransportFailure.unauthorized;
    }
    if (statusCode == 409) return PackageManifestTransportFailure.replayed;
    if (statusCode == 404) {
      return PackageManifestTransportFailure.assignmentUnavailable;
    }
    if (statusCode >= 500) return PackageManifestTransportFailure.unavailable;
    return PackageManifestTransportFailure.rejected;
  }

  static String _secureRequestId() {
    final random = Random.secure();
    final bytes = List<int>.generate(16, (_) => random.nextInt(256));
    final hex =
        bytes.map((value) => value.toRadixString(16).padLeft(2, '0')).join();
    return '${hex.substring(0, 8)}-${hex.substring(8, 12)}-'
        '${hex.substring(12, 16)}-${hex.substring(16, 20)}-'
        '${hex.substring(20)}';
  }
}
